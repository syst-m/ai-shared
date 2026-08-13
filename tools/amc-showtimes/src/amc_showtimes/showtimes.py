"""Showtime retrieval and filtering."""

from __future__ import annotations

import logging
from datetime import datetime, timedelta, timezone
from typing import Any

from .client import AMCClient, AMCClientError
from .formats import matches_formats, resolve_formats
from .models import Showtime

logger = logging.getLogger(__name__)

PACIFIC = timezone(timedelta(hours=-7))


def get_showtimes(
    client: AMCClient,
    theater_number: int,
    date: str | None = None,
) -> list[Showtime]:
    """Fetch showtimes for a theater on a given date.

    Args:
        client: Configured AMCClient instance.
        theater_number: AMC theater number.
        date: Date in YYYY-MM-DD format. None for all upcoming showtimes.

    Returns:
        List of parsed Showtime objects.
    """
    try:
        data = client.get_showtimes(theater_number, date=date)
        return _parse_showtimes(data)
    except AMCClientError as exc:
        logger.warning("Failed to fetch showtimes: %s", exc)
        return []


def filter_by_formats(
    showtimes: list[Showtime], formats: list[str]
) -> list[Showtime]:
    """Filter showtimes by premium format codes.

    Args:
        showtimes: List of Showtime objects to filter.
        formats: Format codes or aliases (e.g., ['dolby', 'imax']).

    Returns:
        Filtered list matching at least one requested format.
    """
    resolved = resolve_formats(formats)
    if not resolved:
        return showtimes

    return [
        st for st in showtimes
        if any(a.code in resolved for a in st.attributes)
    ]


def filter_after_time(
    showtimes: list[Showtime], hour: int, minute: int = 0
) -> list[Showtime]:
    """Filter showtimes to only those at or after the given local time.

    Compares against the hour/minute portion of showDateTimeLocal
    (timezone-agnostic comparison).

    Args:
        showtimes: List of Showtime objects.
        hour: Hour (24h format).
        minute: Minutes.

    Returns:
        Filtered list of showtimes starting at or after the specified time.
    """
    cutoff_minutes = hour * 60 + minute
    result: list[Showtime] = []
    for st in showtimes:
        local_dt = st.showDateTimeLocal
        show_minutes = local_dt.hour * 60 + local_dt.minute
        if show_minutes >= cutoff_minutes:
            result.append(st)
    return result


def _parse_showtimes(data: dict[str, Any]) -> list[Showtime]:
    """Parse the raw API response into Showtime objects."""
    items = []
    # Handle both top-level showtime lists and embedded structures
    for key in ("showtimes", "values"):
        collection = (data.get("_embedded", {}) or {}).get(key, [])
        if not collection:
            collection = data.get(key, [])
        if collection:
            items.extend(collection)

    # If we still have nothing, maybe the response is a flat list at root
    if not items and isinstance(data.get("values"), list):
        items = data["values"]

    parsed: list[Showtime] = []
    for raw in items:
        try:
            parsed.append(_parse_showtime(raw))
        except (KeyError, ValueError) as exc:
            logger.debug("Skipping unparseable showtime: %s", exc)
    return parsed


def _parse_showtime(data: dict[str, Any]) -> Showtime:
    """Parse a single showtime dict."""
    attrs = []
    for attr in data.get("attributes") or []:
        attrs.append({
            "id": attr.get("id", 0),
            "code": attr.get("code", ""),
            "name": attr.get("name", ""),
        })

    prices = []
    for p in data.get("ticketPrices") or []:
        prices.append({
            "priceTypeCode": p.get("priceTypeCode", ""),
            "retailPrice": float(p.get("retailPrice", 0) or 0),
            "promotionalDiscount": float(p.get("promotionalDiscount", 0) or 0),
            "salePrice": float(p.get("salePrice", 0) or 0),
        })

    # Handle datetime fields — they may be strings or already parsed
    utc_dt = _parse_datetime(data.get("showDateTimeUtc", ""))
    local_dt = _parse_datetime(data.get("showDateTimeLocal", ""))

    return Showtime(
        id=data["id"],
        movieId=data["movieId"],
        movieName=data["movieName"],
        showDateTimeUtc=utc_dt,
        showDateTimeLocal=local_dt,
        utcOffset=data.get("utcOffset", "-07:00"),
        theatreId=data["theatreId"],
        auditorium=str(data.get("auditorium", "")),
        layoutId=data["layoutId"],
        performanceNumber=data["performanceNumber"],
        runTime=data.get("runTime", 0),
        mpaaRating=data.get("mpaaRating", ""),
        genre=data.get("genre", ""),
        isAlmostSoldOut=data.get("isAlmostSoldOut", False),
        isSoldOut=data.get("isSoldOut", False),
        isCanceled=data.get("isCanceled", False),
        attributes=attrs,  # type: ignore[arg-type]
        ticketPrices=prices,  # type: ignore[arg-type]
    )


def _parse_datetime(value: str | datetime) -> datetime:
    """Parse an ISO datetime string, handling various formats."""
    if isinstance(value, datetime):
        return value
    if not value:
        return datetime.now(timezone.utc)

    # Try standard ISO format first
    try:
        return datetime.fromisoformat(str(value))
    except (ValueError, TypeError):
        pass

    # Fallback: strip timezone suffix and parse
    s = str(value).strip()
    if s.endswith("Z"):
        s = s[:-1] + "+00:00"
    try:
        return datetime.fromisoformat(s)
    except (ValueError, TypeError):
        return datetime.now(timezone.utc)

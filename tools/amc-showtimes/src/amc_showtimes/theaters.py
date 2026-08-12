"""Theater lookup and discovery."""

from __future__ import annotations

import logging
from typing import Any

from .client import AMCClient, AMCClientError
from .models import Theater

logger = logging.getLogger(__name__)

# Pre-known theaters — fast path without API calls.
KNOWN_THEATERS: dict[str, int] = {
    "metreon": 8,
    "amc-metreon": 8,
    "metreon-15": 8,
    "mercado": 17,
    "amc-mercado": 17,
    "mercado-6": 17,
}


def lookup_theater_number(name: str) -> int | None:
    """Resolve a theater alias/name to an AMC theater number.

    Checks known aliases first, then falls back to the API.

    Args:
        name: Theater name or alias (e.g., 'metreon', 'mercado').

    Returns:
        Theater number if found, None otherwise.
    """
    # Fast path: check known aliases (case-insensitive)
    lower = name.strip().lower()
    for alias, number in KNOWN_THEATERS.items():
        if alias == lower or alias.replace("-", "") in lower.replace("-", ""):
            return number

    return None


def get_theater(client: AMCClient, theater_number: int) -> Theater | None:
    """Fetch full theater details from the API.

    Args:
        client: Configured AMCClient instance.
        theater_number: AMC theater number.

    Returns:
        Theater object or None if not found / error.
    """
    try:
        data = client.get_theater(theater_number)
        return _parse_theater(data)
    except AMCClientError as exc:
        logger.warning("Failed to fetch theater %d: %s", theater_number, exc)
        return None


def search_theaters(
    client: AMCClient, query: str
) -> list[Theater]:
    """Search theaters by name.

    Args:
        client: Configured AMCClient instance.
        query: Search term.

    Returns:
        List of matching Theater objects.
    """
    try:
        data = client.search_theaters(query)
        items = (data.get("_embedded", {}) or {}).get("theatres", [])
        return [_parse_theater(t) for t in items if _parse_theater(t)]
    except AMCClientError as exc:
        logger.warning("Theater search failed: %s", exc)
        return []


def _parse_theater(data: dict[str, Any]) -> Theater | None:
    """Parse a theater dict from the API response."""
    if not data:
        return None
    try:
        location_data = data.get("location", {}) or {}
        location = {
            "street": location_data.get("street", ""),
            "city": location_data.get("city", ""),
            "stateCode": location_data.get("stateCode", ""),
            "zip": location_data.get("zip", ""),
            "latitude": float(location_data.get("latitude", 0) or 0),
            "longitude": float(location_data.get("longitude", 0) or 0),
        }
        attributes = []
        for attr in data.get("attributes") or []:
            attributes.append({
                "id": attr.get("id", 0),
                "code": attr.get("code", ""),
                "name": attr.get("name", ""),
            })
        return Theater(
            id=data["id"],
            number=data["number"],
            name=data["name"],
            shortName=data.get("shortName", ""),
            location=location,  # type: ignore[arg-type]
            phone=data.get("phone", ""),
            attributes=attributes,
        )
    except (KeyError, TypeError) as exc:
        logger.warning("Failed to parse theater data: %s", exc)
        return None

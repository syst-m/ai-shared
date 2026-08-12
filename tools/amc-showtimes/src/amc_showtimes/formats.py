"""Format codes and filtering logic."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Sequence


@dataclass(frozen=True)
class Format:
    """A display format supported by AMC theaters."""

    code: str          # API attribute code (e.g., 'IMAX')
    name: str          # Human-readable label
    is_premium: bool   # Whether this is considered a premium format


# All known formats — canonical source of truth.
FORMATS: list[Format] = [
    Format(code="IMAX", name="IMAX", is_premium=True),
    Format(code="IMAX_LASER", name="IMAX with Laser", is_premium=True),
    Format(code="DOLBY_CINEMA", name="Dolby Cinema", is_premium=True),
    Format(code="PRIME", name="PRIME Cinema", is_premium=True),
    Format(code="4DX", name="4DX", is_premium=True),
    Format(code="SCREENX", name="ScreenX", is_premium=True),
    Format(code="LASED", name="Laser Digital", is_premium=False),
]

# Quick lookup maps
FORMAT_BY_CODE: dict[str, Format] = {f.code: f for f in FORMATS}
PREMIUM_CODES: set[str] = {f.code for f in FORMATS if f.is_premium}

# User-friendly aliases (lowercase) → canonical code
_FORMAT_ALIASES: dict[str, str] = {
    # IMAX variants
    "imax": "IMAX",
    "imax-laser": "IMAX_LASER",
    "imax_laser": "IMAX_LASER",
    "imaxlaser": "IMAX_LASER",
    # Dolby
    "dolby": "DOLBY_CINEMA",
    "dolby-cinema": "DOLBY_CINEMA",
    "dolby_cinema": "DOLBY_CINEMA",
    # PRIME
    "prime": "PRIME",
    # 4DX
    "4dx": "4DX",
    # ScreenX
    "screenx": "SCREENX",
    "screen-x": "SCREENX",
    # Standard laser (non-premium)
    "laser": "LASED",
    "lased": "LASED",
}


def resolve_format(code: str) -> Format | None:
    """Resolve a format code or alias to its canonical Format.

    Args:
        code: A format code (e.g., 'IMAX') or alias (e.g., 'dolby', 'imax-laser').

    Returns:
        The resolved Format, or None if unrecognized.
    """
    # Direct code lookup first
    if code in FORMAT_BY_CODE:
        return FORMAT_BY_CODE[code]
    # Alias lookup (case-insensitive)
    alias = _FORMAT_ALIASES.get(code.strip().lower())
    if alias and alias in FORMAT_BY_CODE:
        return FORMAT_BY_CODE[alias]
    return None


def resolve_formats(codes: list[str]) -> set[str]:
    """Resolve a list of format codes/aliases to canonical codes.

    Args:
        codes: List of format codes or aliases.

    Returns:
        Set of resolved canonical codes. Unrecognized codes are dropped.
    """
    resolved: set[str] = set()
    for code in codes:
        fmt = resolve_format(code)
        if fmt:
            resolved.add(fmt.code)
    return resolved


def matches_formats(
    showtime_attributes: list[dict[str, str]],
    format_codes: set[str],
) -> bool:
    """Check whether a showtime's attributes match any requested formats.

    A showtime matches if ANY of its attribute codes are in the requested set.

    Args:
        showtime_attributes: List of attribute dicts from the API, each with at
            least a 'code' key.
        format_codes: Set of canonical format codes to match against.

    Returns:
        True if any attribute code matches.
    """
    if not format_codes:
        return True  # No filter = everything matches
    for attr in showtime_attributes:
        code = attr.get("code", "")
        if code and code in format_codes:
            return True
    return False


def filter_showtimes_by_format(
    showtimes: Sequence[dict[str, Any]],
    formats: list[str],
) -> list[dict[str, Any]]:
    """Filter a list of raw showtime dicts by premium format codes.

    Args:
        showtimes: Raw showtime dicts from the API response.
        formats: List of format codes or aliases to filter by.

    Returns:
        Filtered list of showtime dicts matching at least one requested format.
    """
    resolved = resolve_formats(formats)
    if not resolved:
        return list(showtimes)  # No valid filters = pass through all

    filtered: list[dict[str, Any]] = []
    for st in showtimes:
        attrs = st.get("attributes") or []
        if matches_formats(attrs, resolved):
            filtered.append(st)
    return filtered


def is_premium_format(code: str) -> bool:
    """Check whether a format code is considered premium."""
    fmt = resolve_format(code)
    return fmt.is_premium if fmt else False


def list_premium_codes() -> list[str]:
    """Return all premium format codes."""
    return sorted(PREMIUM_CODES)

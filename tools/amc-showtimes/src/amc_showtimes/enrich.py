"""IMDB and Rotten Tomatoes rating enrichment."""

from __future__ import annotations

import json
import logging
import time
from pathlib import Path
from typing import Any

import requests

from .models import EnrichmentRatings

logger = logging.getLogger(__name__)

# OMDb API — free tier, needs API key.
# Sign up at http://www.omdbapi.com/apikey.aspx
OMDB_BASE = "http://www.omdbapi.com"
OMDB_KEY_PATH = Path.home() / ".openclaw" / ".omdb-api-key"

# Cache directory for enrichment results (TTL 24h)
_CACHE_DIR = Path.home() / ".cache" / "amc-showtimes"
_CACHE_TTL = 86400  # 24 hours


def enrich_movie(
    title: str,
    year: int | None = None,
    omdb_key: str | None = None,
) -> EnrichmentRatings:
    """Enrich a movie with IMDB and Rotten Tomatoes ratings.

    Uses the OMDb API to fetch IMDB scores and Rotten Tomatoes data.

    Args:
        title: Movie title to search for.
        year: Optional release year to disambiguate.
        omdb_key: OMDb API key (falls back to file if not provided).

    Returns:
        EnrichmentRatings with whatever data was available.
    """
    # Check cache first
    cached = _load_cache(title, year)
    if cached is not None:
        logger.debug("Cache hit for '%s'", title)
        return cached

    # Resolve API key
    if omdb_key is None:
        if OMDB_KEY_PATH.exists():
            omdb_key = OMDB_KEY_PATH.read_text().strip()
        else:
            logger.warning(
                "No OMDb API key found. Enrichment requires a free key from "
                "http://www.omdbapi.com/apikey.aspx"
            )
            return EnrichmentRatings()

    # Call OMDb API
    try:
        params: dict[str, Any] = {
            "t": title,
            "apikey": omdb_key,
            "r": "json",
        }
        if year:
            params["y"] = str(year)

        resp = requests.get(OMDB_BASE, params=params, timeout=5)
        if resp.status_code != 200:
            logger.warning("OMDb API returned %d", resp.status_code)
            return EnrichmentRatings()

        data = resp.json()
        if data.get("Response") == "False":
            logger.info(
                "OMDb not found for '%s': %s", title, data.get("Error", "")
            )
            return EnrichmentRatings()

        ratings = _parse_omdb_ratings(data)
        # Cache result
        _save_cache(title, year, ratings)
        return ratings

    except requests.RequestException as exc:
        logger.warning("OMDb request failed: %s", exc)
        return EnrichmentRatings()


def _parse_omdb_ratings(data: dict[str, Any]) -> EnrichmentRatings:
    """Parse OMDb response into EnrichmentRatings."""
    ratings_data = data.get("Ratings", [])
    imdb_score = None
    rt_critics = None
    rt_audience = None

    for r in ratings_data:
        source = r.get("Source", "").lower()
        value = _parse_rating_value(r.get("Value", ""))
        if "imdb" in source or "internet movie database" in source:
            imdb_score = value
        elif "rotten tomatoes" in source or "tomatometer" in source:
            rt_critics = _to_integer_percent(value)
        elif "audience" in source and ("rotten" in source or "tomato" in source):
            rt_audience = _to_integer_percent(value)

    return EnrichmentRatings(
        imdb_score=imdb_score,
        imdb_id=data.get("imdbID", ""),
        rt_critics_score=rt_critics,
        rt_audience_score=rt_audience,
        rt_url=data.get("metascore", ""),  # placeholder — OMDb doesn't give RT URLs
    )


def _parse_rating_value(value: str) -> float | None:
    """Parse a rating value string like '8.1/10' or '95%' to a number."""
    if not value:
        return None
    try:
        # Handle "X.Y/10" format
        if "/" in value:
            return float(value.split("/")[0])
        # Handle "XX%" format
        if "%" in value:
            return float(value.replace("%", ""))
        return float(value)
    except (ValueError, IndexError):
        return None


def _to_integer_percent(value: float | None) -> int | None:
    """Convert a rating to 0-100 integer. Handles both /10 and % formats."""
    if value is None:
        return None
    if value > 10:
        # Already a percentage
        return min(100, max(0, int(round(value))))
    else:
        # Assume /10 scale
        return int(round(value * 10))


# ------------------------------------------------------------------ #
#  Cache helpers                                                      #
# ------------------------------------------------------------------ #


def _cache_key(title: str, year: int | None) -> str:
    """Generate a cache file path for a movie."""
    safe = "".join(c if c.isalnum() else "_" for c in title.lower())[:60]
    year_part = f"_{year}" if year else ""
    return f"{safe}{year_part}.json"


def _load_cache(title: str, year: int | None) -> EnrichmentRatings | None:
    """Load cached enrichment data if within TTL."""
    cache_file = _CACHE_DIR / _cache_key(title, year)
    if not cache_file.exists():
        return None

    try:
        age = time.time() - cache_file.stat().st_mtime
        if age > _CACHE_TTL:
            logger.debug("Cache expired for '%s'", title)
            return None

        data = json.loads(cache_file.read_text())
        return EnrichmentRatings(**data)
    except (json.JSONDecodeError, OSError, TypeError):
        return None


def _save_cache(
    title: str, year: int | None, ratings: EnrichmentRatings
) -> None:
    """Save enrichment data to cache."""
    try:
        _CACHE_DIR.mkdir(parents=True, exist_ok=True)
        cache_file = _CACHE_DIR / _cache_key(title, year)
        cache_file.write_text(ratings.model_dump_json())
    except OSError:
        logger.debug("Failed to write cache for '%s'", title)

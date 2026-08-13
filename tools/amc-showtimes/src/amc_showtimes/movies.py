"""Movie details and catalog."""

from __future__ import annotations

import logging
from typing import Any

from .client import AMCClient, AMCClientError
from .models import Movie, ShowtimeAttribute

logger = logging.getLogger(__name__)


def get_movie(client: AMCClient, movie_id: int) -> Movie | None:
    """Fetch movie details by ID.

    Args:
        client: Configured AMCClient instance.
        movie_id: AMC movie ID.

    Returns:
        Movie object or None if not found / error.
    """
    try:
        data = client.get_movie(movie_id)
        return _parse_movie(data)
    except AMCClientError as exc:
        logger.warning("Failed to fetch movie %d: %s", movie_id, exc)
        return None


def search_movies(
    client: AMCClient, query: str, page: int = 1, size: int = 50
) -> list[Movie]:
    """Search the movie catalog.

    Args:
        client: Configured AMCClient instance.
        query: Search term.
        page: Page number (1-based).
        size: Results per page.

    Returns:
        List of matching Movie objects.
    """
    try:
        data = client.list_movies(page=page, size=size, search=query)
        items = []
        # Handle both flat and embedded structures
        for key in ("movies", "values"):
            collection = (data.get("_embedded", {}) or {}).get(key, [])
            if not collection:
                collection = data.get(key, [])
            if collection:
                items.extend(collection)
        return [_parse_movie(m) for m in items if _parse_movie(m)]
    except AMCClientError as exc:
        logger.warning("Movie search failed: %s", exc)
        return []


def list_now_playing(client: AMCClient, page: int = 1) -> list[Movie]:
    """List currently playing movies.

    Args:
        client: Configured AMCClient instance.
        page: Page number.

    Returns:
        List of Movie objects.
    """
    try:
        data = client.list_movies(page=page, size=50)
        items = []
        for key in ("movies", "values"):
            collection = (data.get("_embedded", {}) or {}).get(key, [])
            if not collection:
                collection = data.get(key, [])
            if collection:
                items.extend(collection)
        return [_parse_movie(m) for m in items if _parse_movie(m)]
    except AMCClientError as exc:
        logger.warning("Movie listing failed: %s", exc)
        return []


def _parse_movie(data: dict[str, Any]) -> Movie | None:
    """Parse a movie dict from the API response."""
    if not data:
        return None
    try:
        attributes: list[ShowtimeAttribute] = []
        for attr in data.get("attributes") or []:
            attributes.append(
                ShowtimeAttribute(
                    id=attr.get("id", 0),
                    code=attr.get("code", ""),
                    name=attr.get("name", ""),
                )
            )
        # Cast list can be various formats
        cast = data.get("castList") or []
        if isinstance(cast, list):
            cast_names: list[str] = [str(c) for c in cast if c]
        else:
            cast_names = []

        return Movie(
            id=data["id"],
            title=data.get("title", data.get("movieName", "Unknown")),
            runtime=int(data.get("runtime", 0) or 0),
            mpaaRating=data.get("mpaaRating", ""),
            genre=data.get("genre", ""),
            synopsis=data.get("synopsis", ""),
            imageUrl=data.get("imageUrl", ""),
            attributes=attributes,
            castList=cast_names,
        )
    except (KeyError, TypeError) as exc:
        logger.warning("Failed to parse movie data: %s", exc)
        return None

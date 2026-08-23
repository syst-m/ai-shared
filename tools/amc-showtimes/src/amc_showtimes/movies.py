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
    """Search the movie catalog by name.

    The AMC API uses the ``name`` query parameter for movie name search and
    requires it (requests without a search criterion return 400). Results
    are restricted to the currently-playing slate.

    Args:
        client: Configured AMCClient instance.
        query: Search term (matched against movie name).
        page: Page number (1-based).
        size: Results per page.

    Returns:
        List of matching Movie objects.
    """
    try:
        data = client.list_movies(name=query, page=page, size=size)
        return _parse_movies_from_response(data)
    except AMCClientError as exc:
        logger.warning("Movie search failed: %s", exc)
        return []


def list_now_playing(client: AMCClient, page: int = 1, size: int = 50) -> list[Movie]:
    """List currently playing movies.

    The AMC API requires a search term; using ``name="*"`` returns a
    broad catalog listing. For the full listing, paginate through results
    (see :func:`list_all_movies`).

    Args:
        client: Configured AMCClient instance.
        page: Page number (1-based).
        size: Results per page.

    Returns:
        List of Movie objects.
    """
    try:
        data = client.list_movies(name="*", page=page, size=size)
        return _parse_movies_from_response(data)
    except AMCClientError as exc:
        logger.warning("Movie listing failed: %s", exc)
        return []


def list_all_movies(client: AMCClient, page_size: int = 100) -> list[Movie]:
    """List ALL movies by following pagination.

    Args:
        client: Configured AMCClient instance.
        page_size: Results per page (max 100).

    Returns:
        List of all Movie objects.
    """
    try:
        data = client.get_all_movies(name="*", page_size=page_size)
        return _parse_movies_from_response(data)
    except AMCClientError as exc:
        logger.warning("Full movie listing failed: %s", exc)
        return []


def _parse_movies_from_response(data: dict[str, Any]) -> list[Movie]:
    """Parse a movie collection from an API response."""
    items: list[dict[str, Any]] = []
    embedded = data.get("_embedded", {}) or {}
    for key in ("movies", "values"):
        collection = embedded.get(key, [])
        if collection:
            items.extend(collection)
    if not items:
        for key in ("movies", "values"):
            if isinstance(data.get(key), list):
                items = data[key]
                break

    return [_parse_movie(m) for m in items if _parse_movie(m)]


def _parse_movie(data: dict[str, Any]) -> Movie | None:
    """Parse a movie dict from the API response.

    Handles both the single-movie detail shape (``title``, ``runtime``,
    ``castList``) and the catalog list-item shape (``name``,
    ``starringActors``, no runtime/genre).
    """
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

        cast = _parse_cast_list(data)

        media = data.get("media") or {}
        return Movie(
            id=data["id"],
            title=data.get("title") or data.get("name") or "Unknown",
            runtime=int(data.get("runtime") or data.get("runTime") or 0),
            mpaaRating=data.get("mpaaRating", ""),
            genre=data.get("genre", ""),
            synopsis=data.get("synopsis", ""),
            imageUrl=data.get("imageUrl") or media.get("posterStandard", ""),
            attributes=attributes,
            castList=cast,
        )
    except (KeyError, TypeError) as exc:
        logger.warning("Failed to parse movie data: %s", exc)
        return None


def _parse_cast_list(data: dict[str, Any]) -> list[str]:
    """Extract a cast list from either ``castList`` or ``starringActors``."""
    cast = data.get("castList")
    if isinstance(cast, list):
        return [str(c) for c in cast if c]
    if isinstance(cast, str) and cast.strip():
        return [c.strip() for c in cast.replace("&", ",").split(",") if c.strip()]

    actors = data.get("starringActors")
    if isinstance(actors, str) and actors.strip():
        return [c.strip() for c in actors.replace("&", ",").split(",") if c.strip()]
    return []

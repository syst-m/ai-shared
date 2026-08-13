"""Test fixtures and mocked AMC API responses."""

from __future__ import annotations

import json
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest


# ------------------------------------------------------------------ #
#  Mocked AMC API responses — realistic HAL-style JSON               #
# ------------------------------------------------------------------ #

MOCK_MOVIE_DUNE = {
    "id": 98765,
    "title": "Dune: Part Two",
    "runtime": 166,
    "mpaaRating": "PG-13",
    "genre": "Sci-Fi",
    "synopsis": "Paul Atreides unites with Chani and the Fremen while on a warpath of revenge against the conspirators who destroyed his family.",
    "imageUrl": "https://images.amctheatres.com/lin Pictures/dune2.jpg",
    "attributes": [
        {"id": 10, "code": "IMAX", "name": "IMAX"},
        {"id": 20, "code": "DOLBY_CINEMA", "name": "Dolby Cinema"},
        {"id": 30, "code": "PRIME", "name": "PRIME Cinema"},
    ],
    "castList": ["Timothée Chalamet", "Zendaya", "Rebecca Ferguson"],
}

MOCK_MOVIE_LIST = {
    "_embedded": {
        "movies": [MOCK_MOVIE_DUNE, {
            "id": 98766,
            "title": "Inside Out 2",
            "runtime": 100,
            "mpaaRating": "PG",
            "genre": "Animation",
            "synopsis": "Riley enters her teenage years and new emotions arrive.",
            "imageUrl": "",
            "attributes": [
                {"id": 50, "code": "IMAX_LASER", "name": "IMAX with Laser"},
            ],
            "castList": ["Amy Poehler"],
        }]
    },
    "_links": {},
    "hasNextPage": False,
    "hasPreviousPage": False,
    "numberOfPages": 1,
    "pageSize": 50,
    "pageNumber": 1,
    "totalPages": 1,
    "totalElements": 2,
}

MOCK_THEATER_METREON = {
    "id": 8001,
    "number": 8,
    "name": "AMC Metreon 15",
    "shortName": "Metreon 15",
    "location": {
        "street": "135 4th St",
        "city": "San Francisco",
        "stateCode": "CA",
        "zip": "94103",
        "latitude": 37.7846,
        "longitude": -122.4038,
    },
    "phone": "(415) 369-6200",
    "attributes": [
        {"id": 10, "code": "PRIME", "name": "PRIME Cinema"},
        {"id": 20, "code": "DOLBY_CINEMA", "name": "DOLBY CINEMA"},
    ],
}

MOCK_THEATER_MERCADO = {
    "id": 17001,
    "number": 17,
    "name": "AMC Mercado 6",
    "shortName": "Mercado 6",
    "location": {
        "street": "555 El Camino Real",
        "city": "Mountain View",
        "stateCode": "CA",
        "zip": "94040",
        "latitude": 37.3955,
        "longitude": -122.0787,
    },
    "phone": "(650) 969-7626",
    "attributes": [],
}

MOCK_SHOWTIMES = {
    "_embedded": {
        "showtimes": [
            {
                "id": 100001,
                "movieId": 98765,
                "movieName": "Dune: Part Two",
                "showDateTimeUtc": "2026-08-12T02:00:00Z",
                "showDateTimeLocal": "2026-08-11T19:00:00-07:00",
                "utcOffset": "-07:00",
                "theatreId": 8,
                "auditorium": 1,
                "layoutId": 99001,
                "performanceNumber": 550001,
                "runTime": 166,
                "mpaaRating": "PG-13",
                "genre": "Sci-Fi",
                "isAlmostSoldOut": False,
                "isSoldOut": False,
                "isCanceled": False,
                "attributes": [
                    {"id": 20, "code": "DOLBY_CINEMA", "name": "Dolby Cinema"},
                    {"id": 99, "code": "LASED", "name": "Digital"},
                ],
                "ticketPrices": [
                    {
                        "priceTypeCode": "A",
                        "retailPrice": 18.00,
                        "promotionalDiscount": 0.0,
                        "salePrice": 18.00,
                    }
                ],
            },
            {
                "id": 100002,
                "movieId": 98765,
                "movieName": "Dune: Part Two",
                "showDateTimeUtc": "2026-08-12T05:30:00Z",
                "showDateTimeLocal": "2026-08-11T22:30:00-07:00",
                "utcOffset": "-07:00",
                "theatreId": 8,
                "auditorium": 2,
                "layoutId": 99002,
                "performanceNumber": 550002,
                "runTime": 166,
                "mpaaRating": "PG-13",
                "genre": "Sci-Fi",
                "isAlmostSoldOut": True,
                "isSoldOut": False,
                "isCanceled": False,
                "attributes": [
                    {"id": 10, "code": "IMAX", "name": "IMAX"},
                    {"id": 20, "code": "DOLBY_CINEMA", "name": "Dolby Cinema"},
                ],
                "ticketPrices": [],
            },
            {
                "id": 100003,
                "movieId": 98766,
                "movieName": "Inside Out 2",
                "showDateTimeUtc": "2026-08-12T16:00:00Z",
                "showDateTimeLocal": "2026-08-12T09:00:00-07:00",
                "utcOffset": "-07:00",
                "theatreId": 8,
                "auditorium": 5,
                "layoutId": 99005,
                "performanceNumber": 550003,
                "runTime": 100,
                "mpaaRating": "PG",
                "genre": "Animation",
                "isAlmostSoldOut": False,
                "isSoldOut": False,
                "isCanceled": False,
                "attributes": [
                    {"id": 50, "code": "IMAX_LASER", "name": "IMAX with Laser"},
                ],
                "ticketPrices": [],
            },
            {
                "id": 100004,
                "movieId": 98765,
                "movieName": "Dune: Part Two",
                "showDateTimeUtc": "2026-08-13T03:00:00Z",
                "showDateTimeLocal": "2026-08-12T20:00:00-07:00",
                "utcOffset": "-07:00",
                "theatreId": 8,
                "auditorium": 15,
                "layoutId": 99015,
                "performanceNumber": 550004,
                "runTime": 166,
                "mpaaRating": "PG-13",
                "genre": "Sci-Fi",
                "isAlmostSoldOut": False,
                "isSoldOut": True,
                "isCanceled": False,
                "attributes": [
                    {"id": 30, "code": "PRIME", "name": "PRIME Cinema"},
                ],
                "ticketPrices": [],
            },
            # 4DX showing
            {
                "id": 100005,
                "movieId": 98765,
                "movieName": "Dune: Part Two",
                "showDateTimeUtc": "2026-08-13T04:30:00Z",
                "showDateTimeLocal": "2026-08-12T21:30:00-07:00",
                "utcOffset": "-07:00",
                "theatreId": 8,
                "auditorium": 3,
                "layoutId": 99003,
                "performanceNumber": 550005,
                "runTime": 166,
                "mpaaRating": "PG-13",
                "genre": "Sci-Fi",
                "isAlmostSoldOut": False,
                "isSoldOut": False,
                "isCanceled": True,
                "attributes": [
                    {"id": 60, "code": "4DX", "name": "4DX"},
                ],
                "ticketPrices": [],
            },
        ]
    },
    "_links": {},
    "hasNextPage": False,
    "totalElements": 5,
}

# Search theaters response
MOCK_THEATER_SEARCH = {
    "_embedded": {
        "theatres": [MOCK_THEATER_METREON, MOCK_THEATER_MERCADO],
    },
    "_links": {},
}


# ------------------------------------------------------------------ #
#  Fixtures                                                           #
# ------------------------------------------------------------------ #


@pytest.fixture
def mock_api_key_file(tmp_path: Path) -> Path:
    """Create a temporary API key file."""
    key_file = tmp_path / ".amc-api-key"
    key_file.write_text("A1CA15D3-2EB8-4D63-9C31-3B565278E5E9")
    return key_file


@pytest.fixture
def mock_client(mock_api_key_file: Path) -> MagicMock:
    """Create a mocked AMCClient with canned responses."""
    from amc_showtimes.client import AMCClient

    client = AMCClient(key_path=mock_api_key_file)

    def _get_side_effect(path, params=None, timeout=None):
        if path == "/v2/theatres/8":
            return MOCK_THEATER_METREON
        elif path == "/v2/theatres/17":
            return MOCK_THEATER_MERCADO
        elif "/v2/locations/name/" in path:
            return MOCK_THEATER_SEARCH
        elif path == "/v2/movies":
            return MOCK_MOVIE_LIST
        elif path == "/v2/movies/98765":
            return MOCK_MOVIE_DUNE
        elif "showtimes" in path:
            return MOCK_SHOWTIMES
        else:
            return {}

    client.get = MagicMock(side_effect=_get_side_effect)
    client._request = MagicMock(side_effect=_get_side_effect)
    return client


@pytest.fixture
def mock_showtimes_data() -> dict:
    """Return the mocked showtimes response data."""
    return MOCK_SHOWTIMES


@pytest.fixture
def mock_movie_data() -> dict:
    """Return the mocked movie data."""
    return MOCK_MOVIE_DUNE


@pytest.fixture
def mock_theater_data() -> dict:
    """Return the mocked theater data."""
    return MOCK_THEATER_METREON

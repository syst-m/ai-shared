"""Pydantic data models for AMC API responses."""

from __future__ import annotations

from datetime import datetime
from typing import Any

from pydantic import BaseModel, Field


class ShowtimeAttribute(BaseModel):
    """A format/attribute on a showtime (IMAX, Dolby, etc.)."""

    id: int
    code: str
    name: str


class TicketPrice(BaseModel):
    """Pricing for a specific ticket type."""

    priceTypeCode: str = ""
    retailPrice: float = 0.0
    promotionalDiscount: float = 0.0
    salePrice: float = 0.0


class Showtime(BaseModel):
    """A single showing of a movie at a theater."""

    id: int
    movieId: int
    movieName: str
    showDateTimeUtc: datetime
    showDateTimeLocal: datetime
    utcOffset: str
    theatreId: int
    auditorium: str
    layoutId: int
    performanceNumber: int
    runTime: int  # minutes
    mpaaRating: str
    genre: str
    isAlmostSoldOut: bool = False
    isSoldOut: bool = False
    isCanceled: bool = False
    attributes: list[ShowtimeAttribute] = Field(default_factory=list)
    ticketPrices: list[TicketPrice] = Field(default_factory=list)

    @property
    def format_codes(self) -> list[str]:
        """Return all format codes for this showtime."""
        return [a.code for a in self.attributes]

    @property
    def format_names(self) -> list[str]:
        """Return human-readable format names."""
        return [a.name for a in self.attributes]


class ShowtimeResponse(BaseModel):
    """Wraps the top-level AMC showtimes endpoint response.

    Note: Uses aliases for HAL-style underscore keys since Pydantic v2
    does not allow field names starting with underscores.
    """

    embedded: dict[str, Any] = Field(default_factory=dict, alias="_embedded")
    links: dict[str, Any] = Field(default_factory=dict, alias="_links")
    hasNextPage: bool = False
    hasPreviousPage: bool = False
    numberOfPages: int = 0
    pageSize: int = 0
    pageNumber: int = 0
    totalPages: int = 0
    totalElements: int = 0


class Movie(BaseModel):
    """A movie in the AMC catalog."""

    id: int
    title: str
    runtime: int
    mpaaRating: str
    genre: str
    synopsis: str = ""
    imageUrl: str = ""
    attributes: list[ShowtimeAttribute] = Field(default_factory=list)
    castList: list[str] = Field(default_factory=list)


class TheaterLocation(BaseModel):
    """Geographic location of a theater."""

    street: str = ""
    city: str = ""
    stateCode: str = ""
    zip: str = ""
    latitude: float = 0.0
    longitude: float = 0.0


class Theater(BaseModel):
    """An AMC theater."""

    id: int
    number: int
    name: str
    shortName: str = ""
    location: TheaterLocation = Field(default_factory=TheaterLocation)
    phone: str = ""
    attributes: list[ShowtimeAttribute] = Field(default_factory=list)


class EnrichmentRatings(BaseModel):
    """Enriched external ratings for a movie."""

    imdb_score: float | None = None
    imdb_votes: int | None = None
    rt_critics_score: int | None = None  # 0-100
    rt_audience_score: int | None = None  # 0-100
    imdb_id: str = ""
    rt_url: str = ""

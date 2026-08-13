"""Tests for movie details and catalog."""

import pytest

from amc_showtimes.movies import (
    get_movie,
    list_now_playing,
    search_movies,
    _parse_movie,
)


class TestParseMovie:
    """Test movie data parsing."""

    def test_parse_dune(self, mock_movie_data) -> None:
        result = _parse_movie(mock_movie_data)
        assert result is not None
        assert result.id == 98765
        assert result.title == "Dune: Part Two"
        assert result.runtime == 166
        assert result.mpaaRating == "PG-13"
        assert result.genre == "Sci-Fi"

    def test_parse_attributes(self, mock_movie_data) -> None:
        result = _parse_movie(mock_movie_data)
        assert result is not None
        codes = [a.code for a in result.attributes]
        assert "IMAX" in codes
        assert "DOLBY_CINEMA" in codes

    def test_parse_cast(self, mock_movie_data) -> None:
        result = _parse_movie(mock_movie_data)
        assert result is not None
        assert len(result.castList) >= 1
        assert "Timothée Chalamet" in result.castList

    def test_parse_empty_returns_none(self) -> None:
        assert _parse_movie({}) is None

    def test_parse_none_returns_none(self) -> None:
        assert _parse_movie(None) is None


class TestGetMovie:
    """Test fetching a single movie."""

    def test_get_dune(self, mock_client) -> None:
        movie = get_movie(mock_client, 98765)
        assert movie is not None
        assert movie.title == "Dune: Part Two"
        assert movie.runtime == 166


class TestSearchMovies:
    """Test movie search."""

    def test_search_returns_results(self, mock_client) -> None:
        results = search_movies(mock_client, "dune")
        assert len(results) >= 1

    def test_search_finds_dune(self, mock_client) -> None:
        results = search_movies(mock_client, "dune")
        titles = [m.title for m in results]
        assert any("Dune" in t for t in titles)


class TestListNowPlaying:
    """Test listing currently playing movies."""

    def test_list_returns_results(self, mock_client) -> None:
        results = list_now_playing(mock_client)
        assert len(results) >= 1

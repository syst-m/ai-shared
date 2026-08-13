"""Tests for theater lookup and discovery."""

import pytest

from amc_showtimes.theaters import (
    KNOWN_THEATERS,
    _parse_theater,
    get_theater,
    lookup_theater_number,
    search_theaters,
)


class TestLookupTheaterNumber:
    """Test theater name-to-number resolution."""

    def test_metreon_exact(self) -> None:
        assert lookup_theater_number("metreon") == 8

    def test_metreon_alias(self) -> None:
        for alias in ("amc-metreon", "metreon-15"):
            assert lookup_theater_number(alias) == 8

    def test_mercado_exact(self) -> None:
        assert lookup_theater_number("mercado") == 17

    def test_mercado_alias(self) -> None:
        for alias in ("amc-mercado", "mercado-6"):
            assert lookup_theater_number(alias) == 17

    def test_case_insensitive(self) -> None:
        assert lookup_theater_number("METREON") == 8
        assert lookup_theater_number("Mercado") == 17

    def test_unknown_returns_none(self) -> None:
        assert lookup_theater_number("nonexistent") is None


class TestKnownTheaters:
    """Test the known theaters registry."""

    def test_metreon_registered(self) -> None:
        assert "metreon" in KNOWN_THEATERS
        assert KNOWN_THEATERS["metreon"] == 8

    def test_mercado_registered(self) -> None:
        assert "mercado" in KNOWN_THEATERS
        assert KNOWN_THEATERS["mercado"] == 17

    def test_registry_not_empty(self) -> None:
        assert len(KNOWN_THEATERS) >= 6


class TestParseTheater:
    """Test theater data parsing."""

    def test_parse_metreon(self, mock_theater_data) -> None:
        result = _parse_theater(mock_theater_data)
        assert result is not None
        assert result.number == 8
        assert result.name == "AMC Metreon 15"
        assert result.location.city == "San Francisco"
        assert result.location.stateCode == "CA"

    def test_parse_empty(self) -> None:
        assert _parse_theater({}) is None
        assert _parse_theater(None) is None

    def test_parse_missing_location(self) -> None:
        data = {"id": 1, "number": 1, "name": "Test"}
        result = _parse_theater(data)
        assert result is not None
        assert result.location.city == ""


class TestGetTheater:
    """Test API theater fetch."""

    def test_get_metreon(self, mock_client) -> None:
        theater = get_theater(mock_client, 8)
        assert theater is not None
        assert theater.number == 8
        assert "Metreon" in theater.name

    def test_get_mercado(self, mock_client) -> None:
        theater = get_theater(mock_client, 17)
        assert theater is not None
        assert theater.number == 17
        assert "Mercado" in theater.name


class TestSearchTheaters:
    """Test theater search."""

    def test_search_returns_results(self, mock_client) -> None:
        results = search_theaters(mock_client, "amc")
        assert len(results) >= 2

    def test_search_finds_metreon(self, mock_client) -> None:
        results = search_theaters(mock_client, "amc")
        numbers = [t.number for t in results]
        assert 8 in numbers

    def test_search_finds_mercado(self, mock_client) -> None:
        results = search_theaters(mock_client, "amc")
        numbers = [t.number for t in results]
        assert 17 in numbers

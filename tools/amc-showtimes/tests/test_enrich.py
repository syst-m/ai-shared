"""Tests for IMDB/Rotten Tomatoes enrichment."""

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from amc_showtimes.enrich import (
    enrich_movie,
    _parse_omdb_ratings,
    _parse_rating_value,
    _to_integer_percent,
)


class TestParseRatingValue:
    """Test parsing rating value strings."""

    def test_slash_format(self) -> None:
        assert _parse_rating_value("8.1/10") == pytest.approx(8.1)

    def test_percent_format(self) -> None:
        assert _parse_rating_value("95%") == pytest.approx(95)

    def test_plain_number(self) -> None:
        assert _parse_rating_value("7.5") == pytest.approx(7.5)

    def test_empty(self) -> None:
        assert _parse_rating_value("") is None
        assert _parse_rating_value(None) is None

    def test_invalid(self) -> None:
        assert _parse_rating_value("abc") is None


class TestToIntegerPercent:
    """Test converting ratings to 0-100 integers."""

    def test_from_tenth_scale(self) -> None:
        assert _to_integer_percent(8.5) == 85

    def test_from_percent(self) -> None:
        assert _to_integer_percent(95) == 95

    def test_none(self) -> None:
        assert _to_integer_percent(None) is None

    def test_capped_at_100(self) -> None:
        assert _to_integer_percent(150) == 100


class TestParseOmdbRatings:
    """Test parsing OMDb response."""

    def test_parse_imdb_and_rt(self) -> None:
        data = {
            "imdbID": "tt1234567",
            "Ratings": [
                {"Source": "Internet Movie Database", "Value": "8.1/10"},
                {"Source": "Rotten Tomatoes", "Value": "92%"},
                {"Source": "Metacritic", "Value": "75/100"},
            ],
        }
        result = _parse_omdb_ratings(data)
        assert result.imdb_score == pytest.approx(8.1)
        assert result.rt_critics_score == 92
        assert result.imdb_id == "tt1234567"

    def test_parse_no_rt(self) -> None:
        data = {
            "Ratings": [
                {"Source": "IMDb", "Value": "7.0/10"},
            ],
        }
        result = _parse_omdb_ratings(data)
        assert result.imdb_score == pytest.approx(7.0)
        assert result.rt_critics_score is None

    def test_empty_ratings(self) -> None:
        data = {"Ratings": []}
        result = _parse_omdb_ratings(data)
        assert result.imdb_score is None


class TestEnrichMovie:
    """Test the high-level enrichment function."""

    def test_no_api_key_returns_empty(self, tmp_path: Path) -> None:
        """Should return empty ratings when no OMDb key exists."""
        # Make sure key file doesn't exist in test context
        result = enrich_movie("Dune", omdb_key=None)
        assert result.imdb_score is None

    def test_omdb_not_found(self) -> None:
        """Should handle movie not found gracefully."""
        with patch("amc_showtimes.enrich.requests.get") as mock_get:
            mock_get.return_value.status_code = 200
            mock_get.return_value.json.return_value = {
                "Response": "False",
                "Error": "Movie not found!",
            }
            result = enrich_movie("Nonexistent Movie 9999", omdb_key="fake-key")
            assert result.imdb_score is None

    def test_successful_enrichment(self) -> None:
        """Should parse and return ratings from OMDb."""
        with patch("amc_showtimes.enrich.requests.get") as mock_get:
            mock_get.return_value.status_code = 200
            mock_get.return_value.json.return_value = {
                "Response": "True",
                "imdbID": "tt15239678",
                "Ratings": [
                    {"Source": "Internet Movie Database", "Value": "8.5/10"},
                    {"Source": "Rotten Tomatoes", "Value": "92%"},
                ],
            }

            # Patch cache to not interfere
            with patch("amc_showtimes.enrich._save_cache"):
                result = enrich_movie("Dune: Part Two", omdb_key="fake-key")
                assert result.imdb_score == pytest.approx(8.5)
                assert result.rt_critics_score == 92
                assert result.imdb_id == "tt15239678"

    def test_api_error(self) -> None:
        """Should handle HTTP errors gracefully."""
        import requests as req
        with patch("amc_showtimes.enrich.requests.get") as mock_get:
            mock_get.side_effect = req.RequestException("Connection error")
            result = enrich_movie("Dune", omdb_key="fake-key")
            assert result.imdb_score is None

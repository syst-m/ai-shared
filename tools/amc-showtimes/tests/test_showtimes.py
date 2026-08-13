"""Tests for showtime retrieval and filtering."""

from datetime import datetime, timezone

import pytest

from amc_showtimes.showtimes import (
    filter_after_time,
    filter_by_formats,
    get_showtimes,
    _parse_showtime,
    _parse_showtimes,
)
from amc_showtimes.models import Showtime


class TestParseShowtimes:
    """Test parsing raw API response into Showtime objects."""

    def test_parse_full_response(self, mock_showtimes_data) -> None:
        showtimes = _parse_showtimes(mock_showtimes_data)
        assert len(showtimes) == 5
        assert all(isinstance(s, Showtime) for s in showtimes)

    def test_parse_dolby_showtime(self, mock_showtimes_data) -> None:
        showtimes = _parse_showtimes(mock_showtimes_data)
        dolby = [s for s in showtimes if "DOLBY_CINEMA" in s.format_codes]
        assert len(dolby) >= 1

    def test_parse_imax_showtime(self, mock_showtimes_data) -> None:
        showtimes = _parse_showtimes(mock_showtimes_data)
        imax = [s for s in showtimes if "IMAX" in s.format_codes]
        assert len(imax) >= 1

    def test_parse_sold_out(self, mock_showtimes_data) -> None:
        showtimes = _parse_showtimes(mock_showtimes_data)
        sold_out = [s for s in showtimes if s.isSoldOut]
        assert len(sold_out) == 1
        assert sold_out[0].auditorium == "15"

    def test_parse_canceled(self, mock_showtimes_data) -> None:
        showtimes = _parse_showtimes(mock_showtimes_data)
        canceled = [s for s in showtimes if s.isCanceled]
        assert len(canceled) == 1
        assert "4DX" in canceled[0].format_codes

    def test_parse_almost_sold_out(self, mock_showtimes_data) -> None:
        showtimes = _parse_showtimes(mock_showtimes_data)
        almost = [s for s in showtimes if s.isAlmostSoldOut]
        assert len(almost) == 1


class TestParseShowtime:
    """Test parsing a single showtime dict."""

    def test_parse_basic(self, mock_showtimes_data) -> None:
        raw = mock_showtimes_data["_embedded"]["showtimes"][0]
        result = _parse_showtime(raw)
        assert result.id == 100001
        assert result.movieName == "Dune: Part Two"
        assert result.mpaaRating == "PG-13"
        assert result.runTime == 166

    def test_format_codes_property(self, mock_showtimes_data) -> None:
        raw = mock_showtimes_data["_embedded"]["showtimes"][0]
        result = _parse_showtime(raw)
        assert "DOLBY_CINEMA" in result.format_codes
        assert "LASED" in result.format_codes

    def test_format_names_property(self, mock_showtimes_data) -> None:
        raw = mock_showtimes_data["_embedded"]["showtimes"][0]
        result = _parse_showtime(raw)
        assert "Dolby Cinema" in result.format_names

    def test_ticket_prices_parsed(self, mock_showtimes_data) -> None:
        raw = mock_showtimes_data["_embedded"]["showtimes"][0]
        result = _parse_showtime(raw)
        assert len(result.ticketPrices) == 1
        assert result.ticketPrices[0].salePrice == 18.0


class TestFilterByFormats:
    """Test format-based showtime filtering."""

    def _get_showtimes(self, mock_showtimes_data) -> list[Showtime]:
        return _parse_showtimes(mock_showtimes_data)

    def test_filter_dolby(self, mock_showtimes_data) -> None:
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_by_formats(showtimes, ["dolby"])
        for st in filtered:
            assert "DOLBY_CINEMA" in st.format_codes

    def test_filter_imax(self, mock_showtimes_data) -> None:
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_by_formats(showtimes, ["imax"])
        for st in filtered:
            assert "IMAX" in st.format_codes or "IMAX_LASER" in st.format_codes

    def test_filter_multiple_formats(self, mock_showtimes_data) -> None:
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_by_formats(showtimes, ["dolby", "imax"])
        # Should include showtimes with either IMAX, IMAX_LASER, or DOLBY_CINEMA
        for st in filtered:
            has_match = any(
                code in ("DOLBY_CINEMA", "IMAX", "IMAX_LASER")
                for code in st.format_codes
            )
            assert has_match

    def test_filter_4dx(self, mock_showtimes_data) -> None:
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_by_formats(showtimes, ["4dx"])
        assert len(filtered) == 1
        assert "4DX" in filtered[0].format_codes

    def test_filter_prime(self, mock_showtimes_data) -> None:
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_by_formats(showtimes, ["prime"])
        assert len(filtered) == 1
        assert "PRIME" in filtered[0].format_codes

    def test_filter_unknown_format_returns_all(self, mock_showtimes_data) -> None:
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_by_formats(showtimes, ["FOOBAR"])
        assert len(filtered) == len(showtimes)

    def test_filter_empty_returns_all(self, mock_showtimes_data) -> None:
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_by_formats(showtimes, [])
        assert len(filtered) == len(showtimes)


class TestFilterAfterTime:
    """Test time-based showtime filtering."""

    def _get_showtimes(self, mock_showtimes_data) -> list[Showtime]:
        return _parse_showtimes(mock_showtimes_data)

    def test_filter_after_19(self, mock_showtimes_data) -> None:
        """Filter after 7pm — should exclude the 9am showing."""
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_after_time(showtimes, 19, 0)
        for st in filtered:
            assert st.showDateTimeLocal.hour >= 19

    def test_filter_after_21(self, mock_showtimes_data) -> None:
        """Filter after 9pm — should only get the latest showings."""
        showtimes = self._get_showtimes(mock_showtimes_data)
        filtered = filter_after_time(showtimes, 21, 0)
        for st in filtered:
            assert st.showDateTimeLocal.hour >= 21

    def test_filter_no_results(self, mock_showtimes_data) -> None:
        showtimes = self._get_showtimes(mock_showtimes_data)
        # Nothing after midnight
        filtered = filter_after_time(showtimes, 23, 59)
        assert len(filtered) == 0


class TestGetShowtimes:
    """Test the high-level get_showtimes function."""

    def test_get_showtimes_success(self, mock_client) -> None:
        showtimes = get_showtimes(mock_client, 8, date="2026-08-12")
        assert len(showtimes) >= 1

    def test_get_showtimes_without_date(self, mock_client) -> None:
        showtimes = get_showtimes(mock_client, 8)
        assert len(showtimes) >= 1

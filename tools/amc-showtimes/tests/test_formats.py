"""Tests for format code resolution and filtering."""

import pytest

from amc_showtimes.formats import (
    FORMATS,
    FORMAT_BY_CODE,
    PREMIUM_CODES,
    filter_showtimes_by_format,
    is_premium_format,
    list_premium_codes,
    matches_formats,
    resolve_format,
    resolve_formats,
)


class TestResolveFormat:
    """Test format code and alias resolution."""

    def test_direct_code_imax(self) -> None:
        fmt = resolve_format("IMAX")
        assert fmt is not None
        assert fmt.code == "IMAX"
        assert fmt.is_premium is True

    def test_direct_code_dolby(self) -> None:
        fmt = resolve_format("DOLBY_CINEMA")
        assert fmt is not None
        assert fmt.name == "Dolby Cinema"

    def test_alias_dolby(self) -> None:
        fmt = resolve_format("dolby")
        assert fmt is not None
        assert fmt.code == "DOLBY_CINEMA"

    def test_alias_imax_laser(self) -> None:
        for alias in ("imax-laser", "imax_laser", "imaxlaser"):
            fmt = resolve_format(alias)
            assert fmt is not None, f"Alias '{alias}' should resolve"
            assert fmt.code == "IMAX_LASER"

    def test_alias_prime(self) -> None:
        fmt = resolve_format("prime")
        assert fmt is not None
        assert fmt.code == "PRIME"

    def test_alias_4dx(self) -> None:
        fmt = resolve_format("4dx")
        assert fmt is not None
        assert fmt.code == "4DX"

    def test_alias_screenx(self) -> None:
        for alias in ("screenx", "screen-x"):
            fmt = resolve_format(alias)
            assert fmt is not None, f"Alias '{alias}' should resolve"
            assert fmt.code == "SCREENX"

    def test_alias_laser(self) -> None:
        for alias in ("laser", "lased"):
            fmt = resolve_format(alias)
            assert fmt is not None
            assert fmt.code == "LASED"

    def test_unknown_code(self) -> None:
        assert resolve_format("FOOBAR") is None
        assert resolve_format("") is None

    def test_case_insensitive_alias(self) -> None:
        fmt = resolve_format("DOLBY")
        assert fmt is not None
        assert fmt.code == "DOLBY_CINEMA"


class TestResolveFormats:
    """Test batch format resolution."""

    def test_single(self) -> None:
        result = resolve_formats(["imax"])
        assert result == {"IMAX"}

    def test_multiple(self) -> None:
        result = resolve_formats(["dolby", "imax"])
        assert "DOLBY_CINEMA" in result
        assert "IMAX" in result

    def test_mixed_codes_and_aliases(self) -> None:
        result = resolve_formats(["PRIME", "4dx"])
        assert "PRIME" in result
        assert "4DX" in result

    def test_unknown_dropped(self) -> None:
        result = resolve_formats(["IMAX", "FOOBAR"])
        assert result == {"IMAX"}

    def test_empty_input(self) -> None:
        assert resolve_formats([]) == set()


class TestMatchesFormats:
    """Test format matching against showtime attributes."""

    def test_no_filter_matches_all(self) -> None:
        attrs = [{"code": "IMAX"}, {"code": "LASED"}]
        assert matches_formats(attrs, set()) is True

    def test_match_found(self) -> None:
        attrs = [
            {"code": "DOLBY_CINEMA"},
            {"code": "LASED"},
        ]
        assert matches_formats(attrs, {"DOLBY_CINEMA"}) is True

    def test_no_match(self) -> None:
        attrs = [{"code": "LASED"}]
        assert matches_formats(attrs, {"IMAX", "DOLBY_CINEMA"}) is False

    def test_empty_attributes(self) -> None:
        assert matches_formats([], {"IMAX"}) is False

    def test_any_match(self) -> None:
        """Only one attribute needs to match."""
        attrs = [
            {"code": "LASED"},
            {"code": "PRIME"},
        ]
        assert matches_formats(attrs, {"PRIME", "4DX"}) is True

    def test_empty_code_ignored(self) -> None:
        """Attributes with empty code shouldn't match."""
        attrs = [{"code": ""}, {"name": "Fake"}]
        assert matches_formats(attrs, {"IMAX"}) is False


class TestFilterShowtimesByFormat:
    """Test filtering raw showtime dicts by format."""

    def test_no_formats_pass_through(self) -> None:
        data = [
            {"id": 1, "attributes": [{"code": "IMAX"}]},
            {"id": 2, "attributes": []},
        ]
        result = filter_showtimes_by_format(data, [])
        assert len(result) == 2

    def test_filter_dolby(self) -> None:
        data = [
            {"id": 1, "attributes": [{"code": "DOLBY_CINEMA"}, {"code": "LASED"}]},
            {"id": 2, "attributes": [{"code": "IMAX"}]},
            {"id": 3, "attributes": []},
        ]
        result = filter_showtimes_by_format(data, ["dolby"])
        assert len(result) == 1
        assert result[0]["id"] == 1

    def test_filter_multiple(self) -> None:
        data = [
            {"id": 1, "attributes": [{"code": "DOLBY_CINEMA"}]},
            {"id": 2, "attributes": [{"code": "IMAX"}]},
            {"id": 3, "attributes": []},
        ]
        result = filter_showtimes_by_format(data, ["dolby", "imax"])
        assert len(result) == 2

    def test_unknown_format_passes_all(self) -> None:
        """Unknown format codes are dropped, resulting in no filter."""
        data = [
            {"id": 1, "attributes": [{"code": "IMAX"}]},
            {"id": 2, "attributes": []},
        ]
        result = filter_showtimes_by_format(data, ["FOOBAR"])
        assert len(result) == 2  # No valid filter = pass through


class TestPremiumFormats:
    """Test premium format classification."""

    def test_premium_codes(self) -> None:
        for code in ("IMAX", "IMAX_LASER", "DOLBY_CINEMA", "PRIME", "4DX", "SCREENX"):
            assert is_premium_format(code) is True

    def test_lased_not_premium(self) -> None:
        assert is_premium_format("LASED") is False

    def test_unknown_not_premium(self) -> None:
        assert is_premium_format("FOOBAR") is False

    def test_list_premium(self) -> None:
        codes = list_premium_codes()
        assert len(codes) >= 5
        assert "IMAX" in codes
        assert "DOLBY_CINEMA" in codes


class TestFormatConstants:
    """Test format catalog completeness."""

    def test_all_formats_registered(self) -> None:
        expected = {"IMAX", "IMAX_LASER", "DOLBY_CINEMA", "PRIME", "4DX", "SCREENX", "LASED"}
        assert set(FORMAT_BY_CODE.keys()) == expected

    def test_premium_set(self) -> None:
        assert PREMIUM_CODES == {"IMAX", "IMAX_LASER", "DOLBY_CINEMA", "PRIME", "4DX", "SCREENX"}
        assert "LASED" not in PREMIUM_CODES

    def test_format_count(self) -> None:
        assert len(FORMATS) == 7

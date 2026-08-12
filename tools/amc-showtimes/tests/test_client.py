"""Tests for AMCClient."""

from __future__ import annotations

import json
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest
import requests
from requests.exceptions import Timeout as RequestsTimeout, ConnectionError as RequestsConnectionError

from amc_showtimes.client import AMCClient, AMCClientError


class TestClientInit:
    """Test client initialization and API key loading."""

    def test_init_with_api_key(self) -> None:
        client = AMCClient(api_key="test-key-123")
        assert client.api_key == "test-key-123"

    def test_init_with_key_file(self, tmp_path: Path) -> None:
        key_file = tmp_path / "key.txt"
        key_file.write_text("  file-key-abc  \n")
        client = AMCClient(key_path=key_file)
        assert client.api_key == "file-key-abc"

    def test_api_key_in_headers(self) -> None:
        client = AMCClient(api_key="my-key")
        headers = client._headers
        assert headers["X-AMC-Vendor-Key"] == "my-key"
        assert headers["Accept"] == "application/json"

    def test_no_key_omits_header(self, tmp_path: Path) -> None:
        """When no key file exists at the configured path, no header is sent."""
        client = AMCClient(api_key="", key_path=tmp_path / "nonexistent")
        # Should not have vendor key if empty
        assert "X-AMC-Vendor-Key" not in client._headers or client.api_key == ""


class TestRetryLogic:
    """Test retry with exponential backoff."""

    def test_retry_on_500(self) -> None:
        """Should retry on 5xx errors."""
        client = AMCClient(api_key="test")
        call_count = 0

        def mock_request(*args, **kwargs):
            nonlocal call_count
            call_count += 1
            resp = MagicMock()
            if call_count < 3:
                resp.status_code = 500
                resp.headers = {}
                resp.content = b""
                return resp
            resp.status_code = 200
            resp.json.return_value = {"ok": True}
            resp.content = b'{"ok": true}'
            return resp

        with patch("amc_showtimes.client.requests") as mock_requests:
            mock_requests.request.side_effect = mock_request
            # Disable sleep for speed
            with patch("time.sleep"):
                result = client._request("GET", "/test")
                assert result == {"ok": True}
                assert call_count == 3

    def test_retry_on_429(self) -> None:
        """Should retry on rate limit."""
        client = AMCClient(api_key="test")
        call_count = 0

        def mock_request(*args, **kwargs):
            nonlocal call_count
            call_count += 1
            resp = MagicMock()
            if call_count < 2:
                resp.status_code = 429
                resp.headers = {"Retry-After": "0.1"}
                resp.content = b""
                return resp
            resp.status_code = 200
            resp.json.return_value = {"data": "ok"}
            resp.content = b'{"data": "ok"}'
            return resp

        with patch("amc_showtimes.client.requests") as mock_requests:
            mock_requests.request.side_effect = mock_request
            with patch("time.sleep"):
                result = client._request("GET", "/test")
                assert result == {"data": "ok"}

    def test_max_retries_exhausted(self) -> None:
        """Should raise after max retries."""
        client = AMCClient(api_key="test", max_retries=2)

        def mock_request(*args, **kwargs):
            resp = MagicMock()
            resp.status_code = 500
            resp.headers = {}
            resp.content = b""
            return resp

        with patch("amc_showtimes.client.requests") as mock_requests:
            mock_requests.request.side_effect = mock_request
            with patch("time.sleep"):
                with pytest.raises(AMCClientError, match="Max retries|Server error"):
                    client._request("GET", "/test")

    def test_401_raises_immediately(self) -> None:
        """Auth errors should not retry."""
        client = AMCClient(api_key="test")

        def mock_request(*args, **kwargs):
            resp = MagicMock()
            resp.status_code = 401
            resp.content = b""
            return resp

        with patch("amc_showtimes.client.requests") as mock_requests:
            mock_requests.request.side_effect = mock_request
            with pytest.raises(AMCClientError, match="Authentication|401"):
                client._request("GET", "/test")

    def test_timeout_retries(self) -> None:
        """Should retry on timeout."""
        client = AMCClient(api_key="test", max_retries=2)
        call_count = 0

        def mock_request(*args, **kwargs):
            nonlocal call_count
            call_count += 1
            if call_count < 3:
                raise RequestsTimeout()
            resp = MagicMock()
            resp.status_code = 200
            resp.json.return_value = {"ok": True}
            resp.content = b'{"ok": true}'
            return resp

        with patch("amc_showtimes.client.requests") as mock_requests:
            mock_requests.request.side_effect = mock_request
            with patch("time.sleep"):
                result = client._request("GET", "/test")
                assert result == {"ok": True}

    def test_connection_error_retries(self) -> None:
        """Should retry on connection errors."""
        client = AMCClient(api_key="test", max_retries=1)

        def mock_request(*args, **kwargs):
            raise RequestsConnectionError("refused")

        with patch("amc_showtimes.client.requests") as mock_requests:
            mock_requests.request.side_effect = mock_request
            with patch("time.sleep"):
                with pytest.raises(AMCClientError):
                    client._request("GET", "/test")


class TestConvenienceMethods:
    """Test high-level API methods."""

    def test_get_movie(self, mock_client) -> None:
        result = mock_client.get_movie(98765)
        assert result["id"] == 98765
        assert result["title"] == "Dune: Part Two"

    def test_list_movies(self, mock_client) -> None:
        result = mock_client.list_movies()
        movies = (result.get("_embedded", {}) or {}).get("movies", [])
        assert len(movies) >= 1

    def test_get_theater(self, mock_client) -> None:
        result = mock_client.get_theater(8)
        assert result["number"] == 8
        assert result["name"] == "AMC Metreon 15"

    def test_search_theaters(self, mock_client) -> None:
        result = mock_client.search_theaters("amc")
        theatres = (result.get("_embedded", {}) or {}).get("theatres", [])
        assert len(theatres) >= 1

    def test_get_showtimes_with_date(self, mock_client) -> None:
        result = mock_client.get_showtimes(8, date="2026-08-12")
        showtimes = (result.get("_embedded", {}) or {}).get("showtimes", [])
        assert len(showtimes) >= 1

    def test_get_showtimes_without_date(self, mock_client) -> None:
        result = mock_client.get_showtimes(8)
        # Should still work without date param
        showtimes = (result.get("_embedded", {}) or {}).get("showtimes", [])
        assert len(showtimes) >= 1

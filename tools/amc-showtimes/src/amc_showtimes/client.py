"""AMC API client with auth, retry, and rate limiting."""

from __future__ import annotations

import logging
import random
import time
from pathlib import Path
from typing import Any

import requests
from requests.exceptions import ConnectionError as RequestsConnectionError, Timeout as RequestsTimeout

logger = logging.getLogger(__name__)

BASE_URL = "https://api.amctheatres.com"
API_KEY_PATH = Path.home() / ".openclaw" / ".amc-api-key"


class AMCClient:
    """HTTP client for the AMC Theatres API.

    Handles authentication, retries with exponential backoff + jitter,
    and rate-limit awareness (429 responses).
    """

    def __init__(
        self,
        api_key: str | None = None,
        key_path: Path | None = None,
        base_url: str = BASE_URL,
        timeout: float = 8.0,
        max_retries: int = 3,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.max_retries = max_retries

        # Resolve API key
        if api_key is not None:
            self.api_key = api_key.strip()
        elif key_path and key_path.exists():
            self.api_key = key_path.read_text().strip()
        elif API_KEY_PATH.exists():
            self.api_key = API_KEY_PATH.read_text().strip()
        else:
            self.api_key = ""

    @property
    def _headers(self) -> dict[str, str]:
        headers: dict[str, str] = {
            "Accept": "application/json",
        }
        if self.api_key:
            headers["X-AMC-Vendor-Key"] = self.api_key
        return headers

    # ------------------------------------------------------------------ #
    #  Core HTTP helpers                                                  #
    # ------------------------------------------------------------------ #

    def _request(
        self,
        method: str,
        path: str,
        params: dict[str, Any] | None = None,
        timeout: float | None = None,
    ) -> dict[str, Any]:
        """Execute an HTTP request with retry + backoff.

        Args:
            method: HTTP method (GET, POST, etc.).
            path: URL path relative to base_url.
            params: Query parameters.
            timeout: Per-request timeout in seconds.

        Returns:
            Parsed JSON response as a dict.

        Raises:
            AMCClientError: On unrecoverable HTTP errors.
        """
        url = f"{self.base_url}{path}"
        effective_timeout = timeout or self.timeout
        last_error: Exception | None = None

        for attempt in range(self.max_retries + 1):
            try:
                response = requests.request(
                    method,
                    url,
                    headers=self._headers,
                    params=params,
                    timeout=effective_timeout,
                )

                # Rate limiting — back off
                if response.status_code == 429:
                    retry_after = float(response.headers.get("Retry-After", 1))
                    jitter = _jitter(retry_after)
                    wait = retry_after + jitter
                    logger.warning(
                        "Rate limited (429); backing off %.1fs [attempt %d/%d]",
                        wait,
                        attempt + 1,
                        self.max_retries + 1,
                    )
                    time.sleep(wait)
                    continue

                # Server errors — retryable with backoff
                if response.status_code >= 500:
                    last_error = AMCClientError(
                        f"Server error {response.status_code}",
                        status_code=response.status_code,
                    )
                    wait = _backoff(attempt)
                    logger.warning(
                        "Server error %d; retrying in %.1fs [attempt %d/%d]",
                        response.status_code,
                        wait,
                        attempt + 1,
                        self.max_retries + 1,
                    )
                    time.sleep(wait)
                    continue

                # Auth / inactive key
                if response.status_code == 401:
                    raise AMCClientError(
                        "Authentication failed. Check your API key.",
                        status_code=401,
                    )

                # Other 4xx — fail fast
                if response.status_code >= 400:
                    body = _safe_json(response)
                    raise AMCClientError(
                        f"API error {response.status_code}: {_error_detail(body)}",
                        status_code=response.status_code,
                        body=body,
                    )

                return response.json() if response.content else {}

            except RequestsTimeout:
                last_error = AMCClientError(
                    f"Request timed out after {effective_timeout}s",
                    timeout=True,
                )
                logger.warning(
                    "Timeout; retrying [attempt %d/%d]",
                    attempt + 1,
                    self.max_retries + 1,
                )
                time.sleep(_backoff(attempt))
            except RequestsConnectionError as exc:
                last_error = AMCClientError(
                    f"Connection error: {exc}",
                )
                logger.warning(
                    "Connection error; retrying [attempt %d/%d]",
                    attempt + 1,
                    self.max_retries + 1,
                )
                time.sleep(_backoff(attempt))

        # Exhausted retries
        raise last_error or AMCClientError("Max retries exceeded")

    def get(
        self,
        path: str,
        params: dict[str, Any] | None = None,
        timeout: float | None = None,
    ) -> dict[str, Any]:
        """GET request with retry logic."""
        return self._request("GET", path, params=params, timeout=timeout)

    # ------------------------------------------------------------------ #
    #  Convenience endpoints                                              #
    # ------------------------------------------------------------------ #

    def get_movie(self, movie_id: int, timeout: float = 5.0) -> dict[str, Any]:
        """Fetch a single movie by ID."""
        return self.get(f"/v2/movies/{movie_id}", timeout=timeout)

    def list_movies(
        self,
        page: int = 1,
        size: int = 50,
        search: str | None = None,
        timeout: float = 5.0,
    ) -> dict[str, Any]:
        """List/search movies."""
        params: dict[str, Any] = {
            "page-number": page,
            "page-size": size,
        }
        if search:
            params["search"] = search
        return self.get("/v2/movies", params=params, timeout=timeout)

    def get_theater(self, theater_number: int, timeout: float = 5.0) -> dict[str, Any]:
        """Fetch theater details by AMC theater number."""
        return self.get(f"/v2/theatres/{theater_number}", timeout=timeout)

    def search_theaters(
        self, name: str, timeout: float = 5.0
    ) -> dict[str, Any]:
        """Search theaters by partial name."""
        return self.get(
            f"/v2/locations/name/{_encode(name)}", timeout=timeout
        )

    def get_showtimes(
        self,
        theater_number: int,
        date: str | None = None,
        timeout: float = 8.0,
    ) -> dict[str, Any]:
        """Get showtimes for a theater, optionally filtered by date.

        Args:
            theater_number: AMC theater number (e.g. 8, 17).
            date: Date string in YYYY-MM-DD format. Omit for all showtimes.
            timeout: Request timeout in seconds.
        """
        if date:
            path = f"/v2/theatres/{theater_number}/showtimes/{date}"
        else:
            path = f"/v2/theatres/{theater_number}/showtimes"
        return self.get(path, timeout=timeout)


# ------------------------------------------------------------------ #
#  Exceptions                                                         #
# ------------------------------------------------------------------ #


class AMCClientError(Exception):
    """Raised on unrecoverable API errors."""

    def __init__(
        self,
        message: str,
        *,
        status_code: int | None = None,
        body: dict | None = None,
        timeout: bool = False,
    ) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.body = body or {}
        self.is_timeout = timeout


# ------------------------------------------------------------------ #
#  Helpers                                                            #
# ------------------------------------------------------------------ #


def _backoff(attempt: int, base: float = 0.5) -> float:
    """Exponential backoff with jitter."""
    delay = base * (2 ** attempt)
    return delay + _jitter(delay, factor=0.3)


def _jitter(value: float, factor: float = 0.1) -> float:
    """Add random jitter to a delay value."""
    return abs(random.gauss(0, value * factor))


def _safe_json(response: requests.Response) -> dict:
    """Safely parse response body as JSON."""
    try:
        return response.json()
    except Exception:
        return {}


def _error_detail(body: dict) -> str:
    """Extract error message from response body."""
    for key in ("message", "error", "detail"):
        if key in body:
            return str(body[key])
    return ""


def _encode(value: str) -> str:
    """URL-encode a theater name search value."""
    from urllib.parse import quote
    return quote(value, safe="-_.~")

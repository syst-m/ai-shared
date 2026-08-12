# AMC Showtimes

A lightweight Python CLI tool for querying AMC Theatres showtimes, filtering by premium formats (IMAX, Dolby Cinema, 4DX, PRIME), and enriching movies with IMDB/Rotten Tomatoes ratings.

## Features

- **Theater lookup** — Fast alias resolution for known theaters (AMC Metreon 15, AMC Mercado 6)
- **Showtime retrieval** — By theater, date, with format filtering
- **Premium format filtering** — Dolby Cinema, IMAX, IMAX Laser, 4DX, PRIME
- **Movie details** — Title, runtime, rating, genre, synopsis
- **Rating enrichment** — IMDB + Rotten Tomatoes scores via OMDb API (optional)
- **Natural queries** — "What's playing Dolby after 7pm tonight at Metreon?"
- **Resilient HTTP** — Retry with exponential backoff + jitter, rate-limit awareness

## Quick Start

```bash
# Install
cd amc-showtimes
pip install -e .

# Or as a user:
PYTHONPATH=src python3 -m amc_showtimes.cli --help
```

### API Key Setup

Place your AMC API key (UUID format) at `~/.openclaw/.amc-api-key`:

```bash
echo "A1CA15D3-2EB8-4D63-9C31-3B565278E5E9" > ~/.openclaw/.amc-api-key
chmod 600 ~/.openclaw/.amc-api-key
```

For enrichment (IMDB/RT ratings), get a free OMDb API key and place it at `~/.openclaw/.omdb-api-key`:

```bash
echo "your-omdb-key" > ~/.openclaw/.omdb-api-key
```

## CLI Commands

### List theaters

```bash
amc-showtimes theaters
# Known theaters:
#   #8 — AMC Metreon 15
#     135 4th St, San Francisco, CA 94103
#   #17 — AMC Mercado 6
#     555 El Camino Real, Mountain View, CA 94040
```

### Get showtimes with format filter

```bash
amc-showtimes showtimes --theater metreon --date today --formats dolby,imax
amc-showtimes showtimes -t mercado -d 2026-08-15 -f prime,4dx
```

### Search movies

```bash
amc-showtimes movies --search "dune"
#   Dune: Part Two
#     PG-13 | 2h46m | Sci-Fi | Formats: IMAX, Dolby Cinema, PRIME Cinema
```

### Quick query — premium format after time X

```bash
amc-showtimes query --theater metreon --after "19:00" --formats dolby,imax
# Showtimes for 2026-08-12:
#   Dune: Part Two (PG-13)
#     7:00 PM  Auditorium 1  Dolby Cinema | Digital
#     10:30 PM  Auditorium 2  IMAX | Dolby Cinema (Almost Sold Out)
```

### Enrich with ratings

```bash
amc-showtimes enrich --movie "Dune: Part Two"
#   Dune: Part Two
#     IMDB: 8.5/10
#     Rotten Tomatoes (Critics): 92%
```

### List supported formats

```bash
amc-showtimes formats
# Supported formats:
#   IMAX            — IMAX                 ⭐ PREMIUM
#   DOLBY_CINEMA    — Dolby Cinema         ⭐ PREMIUM
#   PRIME           — PRIME Cinema         ⭐ PREMIUM
#   4DX             — 4DX                  ⭐ PREMIUM
#   SCREENX         — ScreenX              ⭐ PREMIUM
#   IMAX_LASER      — IMAX with Laser      ⭐ PREMIUM
#   LASED           — Laser Digital
```

## Python API

```python
from amc_showtimes.client import AMCClient
from amc_showtimes.theaters import lookup_theater_number, get_theater
from amc_showtimes.showtimes import get_showtimes, filter_by_formats

# Initialize client (reads API key from ~/.openclaw/.amc-api-key)
client = AMCClient()

# Look up theater by alias
theater_num = lookup_theater_number("metreon")  # → 8

# Fetch showtimes for today
from datetime import datetime
today = datetime.now().strftime("%Y-%m-%d")
showtimes = get_showtimes(client, theater_num, date=today)

# Filter by premium formats
dolby_imax = filter_by_formats(showtimes, ["dolby", "imax"])

for st in dolby_imax:
    print(f"{st.movieName} @ {st.showDateTimeLocal}")
    print(f"  Formats: {', '.join(st.format_names)}")
```

## Theater Aliases

| Alias | Theater Number | Location |
|-------|---------------|----------|
| `metreon`, `amc-metreon`, `metreon-15` | 8 | San Francisco, CA |
| `mercado`, `amc-mercado`, `mercado-6` | 17 | Mountain View, CA |

## Format Codes

| Code | Name | Premium? |
|------|------|----------|
| `IMAX` | IMAX | ✅ |
| `IMAX_LASER` | IMAX with Laser | ✅ |
| `DOLBY_CINEMA` | Dolby Cinema | ✅ |
| `PRIME` | PRIME Cinema | ✅ |
| `4DX` | 4DX | ✅ |
| `SCREENX` | ScreenX | ✅ |
| `LASED` | Laser Digital | Standard |

## Project Structure

```
amc-showtimes/
├── README.md
├── pyproject.toml
├── src/
│   └── amc_showtimes/
│       ├── __init__.py
│       ├── client.py          # AMC API client with auth, retry, rate limiting
│       ├── models.py          # Pydantic data models
│       ├── theaters.py        # Theater lookup and discovery
│       ├── showtimes.py       # Showtime retrieval and filtering
│       ├── movies.py          # Movie details and catalog
│       ├── enrich.py          # IMDB + Rotten Tomatoes enrichment
│       ├── formats.py         # Format codes and filtering logic
│       └── cli.py             # Click CLI interface
├── tests/
│   ├── conftest.py            # Fixtures, mocked responses
│   ├── test_client.py
│   ├── test_theaters.py
│   ├── test_showtimes.py
│   ├── test_movies.py
│   ├── test_formats.py
│   └── test_enrich.py
└── examples/
    └── quickstart.py
```

## Testing

```bash
# Install test dependencies
pip install -e ".[test]"

# Run tests (all external calls are mocked)
pytest tests/ -v

# 112 tests, 0 failures
```

## Error Handling

- **Retry logic**: Automatic retry on 429 (rate limit) and 5xx (server errors) with exponential backoff + jitter, max 3 retries
- **Timeouts**: 8s for showtime endpoints, 5s for catalog
- **API key errors**: Clear message when authentication fails
- **Enrichment failures**: Graceful degradation — tool works without OMDb key

## License

MIT

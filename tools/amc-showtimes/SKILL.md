---
name: "amc-showtimes"
description: "Query AMC Theatres for showtimes, movies, theaters, and premium formats using the local amc-showtimes CLI (ai-shared repo)."
version: 2
---

# AMC Showtimes Integration

Query AMC Theatres for showtimes, movies, theaters, and formats.

## When to Use

- User asks for AMC showtimes, movies playing, theater info, or premium format screenings
- User mentions AMC Metreon, AMC Mercado, or other AMC theaters
- User asks "what's playing" or "showtimes for [movie]"

## Prerequisites

- `amc-showtimes` CLI is on PATH — editable-installed from `~/ai-shared/tools/amc-showtimes` (repo `syst-m/ai-shared`, branch `main`). No separate install step needed in this environment.
- If the command is missing: `cd ~/ai-shared/tools/amc-showtimes && uv pip install -e . --system` (or `python3 -m pip install -e . --break-system-packages`)
- API key at `~/.openclaw/.amc-api-key` (chmod 600, 36-char UUID, read-only). Read by default; override with `--api-key-file PATH`.
- OMDb key at `~/.openclaw/.omdb-api-key` (optional — only for `enrich` ratings).
- Global flags: `-v/--verbose`, `--api-key-file PATH`.

## CLI Commands

### Quick Query (Most Common)
```
amc-showtimes query -t <theater> [-a "HH:MM"] [-f <formats>]
```
- `-t` theater alias: `metreon`, `mercado` (aliases: `amc-metreon`, `metreon-15`, `amc-mercado`, `mercado-6`; matching is normalized exact, case/hyphen-insensitive)
- `-a` filter showtimes at or after time (24h)
- `-f` comma-separated formats: `dolby`, `imax`, `imax_laser`, `lased`, `prime`, `4dx`, `screenx`

### Showtimes by Date
```
amc-showtimes showtimes -t <theater> [-d YYYY-MM-DD] [-f <formats>]
```
- Defaults to today; follows `_links.next` pagination (absolute URLs) so ALL showtimes are returned, sorted by local time (naive/aware-safe).

### Search or List Movies
```
amc-showtimes movies [-s "<search term>"] [-p <page>]
```
- Without `-s`: lists currently playing (API requires the `name` param — tool sends wildcard `*` with `movie-status=currently-playing` internally, so the bare command works).
- With `-s`: name search via the `name` query param; `-p` pages the catalog.

### List Theaters
```
amc-showtimes theaters [-s "<search term>"]
```
- Without `-s`: prints the known-theater alias table. With `-s`: live fuzzy search via `GET /v2/locations/name/{name}`.

### List Formats
```
amc-showtimes formats
```

### Enrich with Ratings (OMDb)
```
amc-showtimes enrich --movie "<movie title>"
```
- Requires `~/.openclaw/.omdb-api-key` (optional). 24h file cache at `~/.cache/amc-showtimes/enrich/`.

## Known Theaters (per `KNOWN_THEATERS` in theaters.py)

| Alias | Number | Notes |
|-------|--------|-------|
| metreon / amc-metreon / metreon-15 | 8 | SF |
| mercado / amc-mercado / mercado-6 | 17 | LA |

> ⚠️ **Open issue (2026-08-23):** live probe of `GET /v2/theatres/8` returned a Kansas-City-area record and `/v2/theatres/17` a "Closed Unit", with `name` fields empty — the number mapping may be stale and should be re-verified against the live catalog before relying on it.

## Common Patterns

**"What's playing at [theater]?"**
```
amc-showtimes query -t <theater>
```

**"Showtimes for [movie] at [theater]?"**
```
amc-showtimes query -t <theater> | grep -A 2 "<movie>"
```
Or `showtimes -t <theater> -d <date>` then grep.

**"What's playing [format] after [time]?"**
```
amc-showtimes query -t <theater> -a "HH:MM" -f <format>
```

## Troubleshooting

- `command not found` → editable install is stale; `uv pip install -e ~/ai-shared/tools/amc-showtimes --system`
- API key must be readable (chmod 600); missing key → clear client error at startup
- `WARNING: Failed to parse theater data: 'number'` on `theaters` → known cosmetic bug: the live single-theater object has `id` but no `number` key; showtimes/query flows are unaffected
- Output is plain text; parse with grep/awk for specific movies
- Rate limits are credit-based (429 on excess); client is token-bucket throttled — keep batch queries small

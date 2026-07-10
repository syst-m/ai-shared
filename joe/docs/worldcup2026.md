# World Cup 2026 Tournament Page

Single-page tournament tracker built with HTML + CSS + vanilla JS (no frameworks).

## Architecture

Three files, cleanly separated:

- `worldcup2026.html` — Presentation layer: CSS styles, HTML skeleton, module loader
- `wc-data.json` — Data layer: pure JSON (no code) with tournament data
- `wc-render.js` — Logic layer: ES module that reads `window.WC_DATA` and renders DOM

### Update workflow

To update the page for a new match day, **only edit `wc-data.json`**. The HTML shell and render engine remain untouched.

## Running

```bash
# Simple HTTP server (Python 3)
cd joe
python3 -m http.server 8080

# Then open: http://localhost:8080/worldcup2026.html
```

The page won't work via `file://` because `fetch()` requires an HTTP server.

## Data Structure

See `wc-data.json` for the full schema. Key sections:

- `knockout.qf[]` — Quarter-final matches (update scores/winners as played)
- `knockout.sf[]` — Semi-finals (add results when available)
- `knockout.final` — Final match details
- `scorers[]` — Top scorers list
- `groups[]` — Group standings

## Style Guide

When editing the page, follow these patterns:

1. **Colors**: Use team colors from `wc-data.json` → `teams` map (via `teamColor()` in wc-render.js)
2. **Match cards**: Use `matchCard(m, headerCls, headerText)` — outputs a consistent card with circles, scores, status badges
3. **Status badges**: `.status.win`, `.status.upcoming`, `.status.aet`, `.status.pens`
4. **Team names**: Use `teamInitials()` for circle text
5. **Dates**: Keep format like `"Jul 9"` (no leading zero)
6. **Scores**: Use `hs` (home score) / `as` (away score); set to `null` for upcoming

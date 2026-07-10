# World Cup 2026 — Data Update Instructions

When updating `worldcup2026.html`, only modify **`wc-data.json`**. The HTML shell (`worldcup2026.html`) and render engine (`wc-render.js`) are immutable from your perspective.

## How It Works

The HTML page fetches `wc-data.json` at load time, then the render engine reads `window.WC_DATA` to build all DOM content. There are zero separate code paths — one JSON file drives everything.

**Always use an HTTP server to test** (the page won't work via `file://` because `fetch()` requires HTTP):
```
cd joe && python3 -m http.server 8080
# then open: http://localhost:8080/worldcup2026.html
```

## File Inventory

| File | Role | Change? |
|---|---|---|
| `joe/wc-data.json` | All tournament data | **Yes — this is the only file you edit** |
| `joe/worldcup2026.html` | CSS + HTML skeleton + module loader | Never |
| `joe/wc-render.js` | ES module with render functions | Never |

## JSON Schema

### `meta` — Page metadata
```json
"meta": {
  "title": "FIFA World Cup 2026",        // browser title bar
  "subtitle": "Canada · Mexico · USA — Semi-finals Underway",  // hero subtitle
  "badge": "Live",                         // hero badge text (e.g. "Live", "Final Played")
  "currentDate": "July 14, 2026",          // displayed in hero badge
  "finalDate": "2026-07-19T23:00:00-04:00",// countdown target (ISO 8601)
  "finalVenue": "MetLife Stadium, East Rutherford, NJ"
}
```

### `stats` — Hero bar numbers
```json
"stats": {
  "teams": 48,              // total teams (constant)
  "matchesPlayed": 60,      // how many matches have been played
  "totalMatches": 104,      // total tournament matches
  "progressPct": 58,        // calculated: Math.round(matchesPlayed / totalMatches * 100)
  "totalGoals": "~247",     // string with tilde approximation
  "matchesLeft": 44,        // = totalMatches - matchesPlayed
  "teamsLeft": 32           // teams still in tournament
}
```

### `tournamentStats` — Stats bar (6 items)
```json
"tournamentStats": [
  { "value": "~2.40", "label": "Goals / Match" },
  { "value": "20",     "label": "Clean Sheets" },
  // ... exactly 6 items
]
```

### `teams` — Color map (KEY for all team name lookups)
```json
"teams": {
  "France":"#002395",
  "Morocco":"#C1272D",
  // ... every team that appears in matches MUST have a hex color here
  // Format: "#RRGGBB" — do NOT use named colors or rgb()
}
```
**Critical:** If a match references `"France"` as home/away/winner, `teams["France"]` MUST exist. Missing entries cause gray fallback circles (`#a0aec0`).

### `knockout` — All knockout stage data

#### `r32[]` — Round of 32 (16 matches)
```json
{
  "date": "Jun 28",                    // string, no leading zero on day
  "home": "South Africa",              // team name MUST match teams map key
  "away": "Canada",
  "hs": 0,                             // home score (number) or null if upcoming
  "as": 1,                             // away score (number) or null if upcoming
  "winner": "Canada",                  // team name of winner — omit if not yet decided
  "venue": "SoFi Stadium",             // short venue name (city shown in HTML separately)
  "aet": true,                         // optional — true if match went to extra time
  "pens": "4-3",                       // optional — penalty shootout score string
  "scorers": [                         // optional — list of goalscorers for this match
    { "n": "Mbappe", "m": "34'" },     // n = player name, m = minute with apostrophe
    { "n": "Dembélé", "m": "67'" }
  ]
}
```

#### `r16[]` — Round of 16 (8 matches) — same schema as r32 above

#### `qf[]` — Quarter-finals (4 matches) — same schema

#### `sf[]` — Semi-finals
```json
{
  "id": "SF1",              // "SF1" or "SF2"
  "date": "Jul 14",         // match date
  "winnerQF": [0, 1],       // indices into qf[] array whose winners play — update to actual teams once known
  "venue": "AT&T Stadium, Dallas",
  "hs": null,               // null if upcoming; number if played
  "as": null,
  "winner": null,           // set when match is played
  "aet": false,
  "pens": null,
  "scorers": []
}
```

#### `thirdPlace` — Third-place match
```json
"thirdPlace": {
  "date": "Jul 18",
  "venue": "Hard Rock Stadium, Miami Gardens",
  "hs": null, "as": null, "winner": null, "aet": false, "pens": null, "scorers": []
}
```

#### `final` — Final match (same shape as thirdPlace)

**When a knockout stage is complete:** Set `hs`, `as`, and `winner`. Remove `null` scores. Add scorers if known.

### `groups[]` — Group standings (12 groups, A–L)
```json
{
  "letter": "A",                         // group letter
  "headerClass": "g-a",                  // CSS class for header coloring
  "teams": [                             // exactly 4 teams per group
    {
      "name": "Mexico",                  // MUST match teams map key
      "pts": 9,                          // total points (W*3 + D)
      "gp": 3,                           // games played
      "w": 3,                            // wins
      "d": 0,                            // draws
      "l": 2,                            // losses — should equal gp - w - d
      "gd": 6,                           // goal difference (gf - ga)
      "gf": 8,                           // goals for
      "ga": 2,                           // goals against
      "qa": true                         // qualified — set true for top 2 teams
    }
  ]
}
```
**Rules:**
- Each group has exactly 4 teams
- `qa: true` on the top 2 finishers (or both if both qualify from group)
- `elim: true` on teams that were eliminated (bottom 2)
- `w + d + l == gp` always

### `groupMatches[]` — All 72 group-stage match results
```json
{ "date": "Jun 11", "group": "A", "home": "Mexico", "away": "South Korea", "hs": 1, "as": 0 }
```
- No `winner` field (implied by hs/as)
- Groups A through L
- Each group has exactly 6 matches (round-robin)
- Dates like `"Jun 11"`, `"Jul 9"` (no leading zero on day)

### `scorers[]` — Top scorers leaderboard (descending by goals)
```json
{ "name": "Kylian Mbappe", "team": "France", "goals": 8, "club": "Real Madrid" }
```
- Sort order: highest goals first
- Ties broken alphabetically by name within same goal count
- Keep the list at ~20–25 entries

### `venues[]` — Stadiums used in the tournament
```json
{
  "name": "MetLife Stadium",
  "city": "East Rutherford, NJ",
  "country": "USA",
  "capacity": "82,500",            // string with comma formatting
  "flag": "usa",                   // CSS class suffix for accent color (usa, mex, can)
  "matches": "Final, Semi-final"   // summary of matches hosted
}
```

### `storylines[]` — Tournament narratives
```json
{
  "title": "The Moroccan Miracle",
  "tag": "magic-moment",           // narrative | upset | golden-boot | magic-moment
  "cssClass": "",                  // "" | "magic" | "upset" | "golden" (maps to CSS color themes)
  "body": "Long-form narrative text here…"
}
```

### `timeline[]` — Tournament milestones
```json
{ "date": "Jul 12", "title": "Quarter-final 4", "desc": "Argentina vs Switzerland", "status": "past" }
```
- `status`: `"past"` | `"current"` | `"upcoming"`

### `eliminated` — Teams eliminated by round
```json
"eliminated": {
  "r32": ["Team1", "Team2", ...],    // 15 teams
  "r16": ["Team1", "Team2", ...],    // 8 teams
  "group": ["Team1", "Team2", ...]   // 15 teams that didn't advance past group stage
}
```
- Each array contains plain team name strings (must exist in `teams` map)

## Common Update Scenarios

### After each match day — update knockout results
1. For every finished match, set `hs`, `as`, and `winner`. Remove null scores. Add scorers if available.
2. Check off the corresponding timeline entry (`status: "past"`).
3. Move eliminated teams from tournament consideration into the appropriate `eliminated` bucket (if applicable at that stage).
4. Recalculate `stats.matchesPlayed`, `stats.totalGoals`, `stats.progressPct`, and `stats.matchesLeft`.
5. Update `stats.teamsLeft` based on how many teams remain in knockout.
6. If a QF is complete, set the SF's `winnerQF` to actual team names (the JSON uses indices into qf[] which are correct; if you reorder, update these).

### After group stage — update standings
1. For each group, compute points: W×3 + D×1.
2. Sort teams by: Pts → GD → GF → head-to-head.
3. Set `qa: true` on top 2, `elim: true` on bottom 2.
4. Update all 72 `groupMatches[]` entries for completed matches.

### Golden Boot / Scorers list
1. Recount all goals across all match types (group + knockout).
2. Sort descending by goals, then alphabetically.
3. Keep top ~25 players.

### A stage is complete (e.g., all QFs played)
1. Update `meta.badge` and `meta.subtitle`.
2. Set `meta.currentDate` to today's date.
3. Mark timeline entries as `"past"` where appropriate, one as `"current"`.
4. For completed matches in the finished stage: set scores + winners.
5. For the next stage: keep `hs`/`as` as `null`, add `winnerQF` references for SF slots.

## Validation Checklist Before Saving

Run this to verify consistency:
```bash
node -e "
const d = JSON.parse(require('fs').readFileSync('./wc-data.json','utf8'));
// 1. Every team referenced in matches must exist in teams map
function checkTeams(list, key) {
  list.forEach(m => {
    [m.home, m.away, m.winner].forEach(t => {
      if (t && !d.teams[t]) console.log('MISSING COLOR for ' + t + ' in ' + key);
    });
  });
}
['r32','r16','qf','sf'].forEach(k => checkTeams(d.knockout[k], k));
// 2. Group standings: teams must exist
d.groups.forEach(g => g.teams.forEach(t => {
  if (!d.teams[t.name]) console.log('MISSING COLOR for ' + t.name + ' in group ' + g.letter);
}));
// 3. Eliminated teams
Object.values(d.eliminated).flat().forEach(t => {
  if (!d.teams[t]) console.log('MISSING COLOR for eliminated team ' + t);
});
// 4. Stats consistency
const mp = d.stats.matchesPlayed;
const tm = d.stats.totalMatches;
console.log('Progress: ' + Math.round(mp/tm*100) + '% | Goals avg: ' + (d.stats.totalGoals));
console.log('OK — no missing colors');
"
```

## Naming Conventions

- **Team names:** Use exact FIFA-style names. These MUST match keys in the `teams` color map and all references throughout the file.
  - Examples: `"Ivory Coast"` (not "Cote d'Ivoire"), `"United States"` (not "USA"), `"South Africa"`, `"Bosnia and Herzegovina"`, `"DR Congo"` (not "Congo")
- **Dates:** No leading zero on day number: `"Jul 9"` not `"Jul 09"`. Month names capitalized. Format: `"Mon DD"` for group stage, `"Jul 14"` for knockout.
- **Scores:** Numbers only (`2`, `0`), not strings (`"2"`, `"0"`). Use `null` for upcoming matches.
- **Minute notation:** With apostrophe: `"34'"`. For combined notation like `"67' (pen)"`, keep the apostrophe.
- **Penalty shootout:** String format: `"4-3"`. Do NOT parse this in code — it's display-only.

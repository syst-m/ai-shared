# ai-shared

Shared AI projects, analysis documents, and tools worth publishing or reusing.

## Contents

### 📊 Solar + Battery Installation Analysis (`solar/`)

In-depth solar + battery planning for residential installations, including adversarial review and cost modeling.

| Document | Description |
|----------|-------------|
| [`solar-installation-dc.md`](solar/solar-installation-dc.md) | Daly City, CA installation plan (978 Martin Trail). Full cost breakdown, incentive stack, payback analysis. Reviewed by three independent analysts. |
| [`solar-installation-nj.md`](solar/solar-installation-nj.md) | West Caldwell, NJ installation plan. PSE&G territory, SREC-II eligibility, NEM 3.0 comparison. |
| [`review.md`](solar/review.md) | Adversarial review of the Daly City install doc — flags arithmetic errors, overstated incentives, and unrealistic system sizing assumptions. |
| [`solar-update-prompt.md`](solar/solar-update-prompt.md) | System prompt for AI agents to refresh/validate solar installation plans with current data. |

### ⚽ World Cup 2026 Bracket Tracker (`joe/worldcup2026.html`)

Interactive FIFA World Cup 2026 bracket tracker — self-contained HTML with tabs for knockout results, group standings, upcoming fixtures, top scorers, venues, and tournament timeline. Built for GitHub Pages.

- [Live Demo](https://syst-m.github.io/ai-shared/joe/worldcup2026.html) *(after enabling GitHub Pages)*

## Setup & Publishing

### GitHub Pages

This repo is configured for **GitHub Pages** via the `gh` CLI:

```bash
gh pages enable --source joe/
```

No build step required — all content is static HTML/markdown served as-is.

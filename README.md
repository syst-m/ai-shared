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

### 📦 MCP Inventory API (`examples/mcp-inventory-api/`)

Example database-backed REST API for inventorying MCP metadata — audiences, claims, and scopes. FastAPI + SQLAlchemy 2.0 + Alembic + PostgreSQL, with a test suite that runs against a real PostgreSQL via testcontainers (HTTP layer → DB, no mocks), Dockerfile, and docker-compose for a deployable stack.

| File | Description |
|------|-------------|
| [`README.md`](examples/mcp-inventory-api/README.md) | Data model, API reference, quick start, migration & test guide |
| `src/mcp_inventory/` | App: FastAPI factory, CRUD routers, SQLAlchemy models, Pydantic schemas |
| `alembic/` | Migration chain (applied by the Docker entrypoint and the test suite) |
| `tests/` | testcontainers-based integration tests |

### 🐍 Python by Example (`learning/python-by-example/`)

A zero-to-100 Python refresher in the [Go by Example](https://gobyexample.com/) style, for experienced programmers getting back into Python — backend/web-API weighted. Single self-contained HTML file with **57 editable playgrounds** running real CPython 3.14 in the browser via Pyodide (CDN, ~10 MB one-time). Double-click [`python-by-example.html`](learning/python-by-example/python-by-example.html) to open — no server needed.

### 🛠 Tools (`tools/`)

| Tool | Description |
|------|-------------|
| [`amc-showtimes/`](tools/amc-showtimes/README.md) | AMC showtimes client — Python package, CLI, [API reference](tools/amc-showtimes/API.md), a [Claude skill](tools/amc-showtimes/SKILL.md), a [quickstart](tools/amc-showtimes/examples/quickstart.py), and a test suite |
| [`pv-sentinel/`](tools/pv-sentinel/README.md) | Go DaemonSet that probes node volumes and remediates them. Two detectors: read-only ext4 mounts (whose `/proc/mounts` lies, so the only honest probe is creating a file) and stuck PVC expansions (a `spec`/`status` size gap whose usual cause is an iSCSI session never rescanned). Ships in `report` mode; [an example](tools/pv-sentinel/examples/daemonset-remediate.yaml) shows the remediate posture |

## Setup & Publishing

### GitHub Pages

This repo is configured for **GitHub Pages** via the `gh` CLI:

```bash
gh pages enable --source joe/
```

No build step required — all content is static HTML/markdown served as-is.

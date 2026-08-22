# MCP Inventory API

An example **database-backed REST API** for inventorying MCP metadata — **audiences**, **claims**, and **scopes**. Built to demonstrate a deployment-ready Python backend: FastAPI, SQLAlchemy 2.0, Alembic migrations, PostgreSQL, and a test suite that runs against a **real** PostgreSQL via testcontainers (no mocks of the database anywhere).

## Data Model

| Entity | Description | Key constraints |
|--------|-------------|-----------------|
| `audiences` | A registered MCP audience (e.g. a server namespace) | unique `name` |
| `scopes` | Access scopes granted within an audience (e.g. `mcp:tools:read`) | unique `(audience_id, name)`, FK → audience (CASCADE) |
| `claims` | Protocol claims required by an audience (e.g. `sub`, `mcp:server_id`) | unique `(audience_id, name)`, FK → audience (CASCADE) |

## API

All endpoints live under `/api/v1`; the probe is at `/healthz`.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Readiness; 200 if `SELECT 1` round-trips, 503 when the DB is down |
| `GET` | `/api/v1/audiences` | List audiences (`limit`/`offset`) |
| `POST` | `/api/v1/audiences` | Create (201; 409 on duplicate name; 422 on bad input) |
| `GET` | `/api/v1/audiences/{id}` | Fetch one (404 if absent, 422 if malformed) |
| `PATCH` | `/api/v1/audiences/{id}` | Update name/description |
| `DELETE` | `/api/v1/audiences/{id}` | Delete; **cascades** to scopes & claims |
| `GET` | `/api/v1/scopes` | List scopes (optional `audience_id` filter) |
| `POST` | `/api/v1/scopes` | Create (404 unknown audience, 409 duplicate, 422 bad name) |
| `GET`/`PATCH`/`DELETE` | `/api/v1/scopes/{id}` | Standard |
| `GET` | `/api/v1/claims` | List claims (optional `audience_id` filter) |
| `POST` | `/api/v1/claims` | Create (type ∈ `string` `integer` `boolean` `json`) |
| `GET`/`PATCH`/`DELETE` | `/api/v1/claims/{id}` | Standard |

Interactive docs at `/docs` (Swagger) and `/openapi.json` when running locally.

## Quick Start

Prereqs: Python ≥ 3.10 and Docker (for tests / compose).

```bash
make install        # venv + deps (incl. test extras)
make migrate        # alembic upgrade head (needs a reachable Postgres)
make run            # uvicorn on :8000
```

Point it at your own database:

```bash
export MCPINVENTORY_DATABASE_URL=postgresql+psycopg://user:pass@host:5432/db
make migrate && make run
```

### Or with Docker Compose (Postgres + API, migrations run on boot)

```bash
docker compose up --build
curl localhost:8000/healthz
```

The Postgres container is **not** published to the host (the API reaches it over
the compose network); inspect it with
`docker compose exec db psql -U mcp -d mcp_inventory`. Credentials and the API's
database URL can be overridden via env vars (`POSTGRES_USER`, `POSTGRES_PASSWORD`,
`POSTGRES_DB`, `MCPINVENTORY_DATABASE_URL`) — the shipped values are example-only.

### Try it

```bash
# create an audience
curl -s localhost:8000/api/v1/audiences -H 'content-type: application/json' \
  -d '{"name": "github-mcp", "description": "GitHub MCP server"}'

# add a scope and a claim under it (use the returned audience id)
curl -s localhost:8000/api/v1/scopes -H 'content-type: application/json' \
  -d '{"name": "mcp:tools:read", "audience_id": "<audience-id>"}'
curl -s localhost:8000/api/v1/claims -H 'content-type: application/json' \
  -d '{"name": "mcp:server_id", "type": "string", "audience_id": "<audience-id>"}'
```

## Migrations

Schema is managed by [Alembic](https://alembic.sqlalchemy.org/). The env var
`MCPINVENTORY_DATABASE_URL` always takes precedence over `alembic.ini`.

```bash
make migrate                                  # upgrade head
.venv/bin/alembic revision --autogenerate -m "add x"   # new revision from models
.venv/bin/alembic downgrade -1                # step back
```

## Testing

```bash
make test        # full suite
make coverage    # suite + coverage report
```

The suite boots a real `postgres:16-alpine` container via testcontainers, applies the
Alembic chain to it (`test_migrations` asserts the head revision is recorded), then
exercises every endpoint through the HTTP layer — with assertions on **both** the JSON
response and the rows actually stored in PostgreSQL (including cascade deletes,
unique violations, FK 404s, and tz-aware timestamps). Tables are truncated between
tests for isolation.

## Project Layout

```
├── alembic/                 # migration env + versions/
├── alembic.ini
├── src/mcp_inventory/
│   ├── app.py               # FastAPI factory (inject a Database for tests)
│   ├── api/                 # health + audiences/scopes/claims routers
│   ├── config.py            # pydantic-settings (MCPINVENTORY_* env)
│   ├── crud.py              # session-scoped data access
│   ├── database.py          # engine/Session plumbing
│   ├── models.py            # SQLAlchemy 2.0 ORM
│   └── schemas.py           # Pydantic v2 request/response
└── tests/                   # testcontainers + FastAPI TestClient
```

## Deployment

`Dockerfile` builds a slim, non-root image whose entrypoint runs
`alembic upgrade head` before starting uvicorn — so a container is deployable as soon
as it has a valid `MCPINVENTORY_DATABASE_URL`. A `HEALTHCHECK` polls `/healthz`.

## Verification

Everything below was executed (not just linted) against **PostgreSQL 16**
(`postgres:16-alpine`) — the only engine this project supports, since the models,
migrations, and tests use Postgres-specific types (e.g. `postgresql.UUID`).

- **Test suite** — `make test`: boots a real PostgreSQL via testcontainers, applies
  the Alembic chain to it, then runs **57 tests, all passing** (Python 3.12) through
  the HTTP layer, asserting both the JSON responses *and* the rows stored in the
  container database (CRUD, unique violations, FK 404s, cascade deletes, pagination
  bounds, tz-aware timestamps).
- **Coverage** — `make coverage`: **97%** total; 100% on every router, model, schema,
  and CRUD module (misses are only the `__main__` entry point and the cached-settings
  loaders).
- **Migrations** — revision `0001` applies cleanly to a fresh container
  (`alembic upgrade head`); `test_migrations` asserts the head revision is recorded.
- **Docker deploy** — `docker compose up --build`: image builds, both containers
  reach `(healthy)`, the entrypoint migration runs on boot, and a full curl smoke
  pass succeeds (`/healthz` 200 → audience/scope/claim 201 → audience-filter 200 →
  duplicate 409 → invalid name 422 → `DELETE` 204 → child counts `0/0/0` after
  cascade, checked via `psql`). Image runs as unprivileged `appuser` (uid 1000).
- **DB outage** — `/healthz` returns **503** (covered by
  `test_healthz_is_503_when_database_unreachable`) when the database cannot be
  reached, instead of surfacing an unhandled 500.

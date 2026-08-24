"""Test fixtures.

Every test runs against a **real PostgreSQL** spun up by testcontainers:
the session boots one container, applies the Alembic migrations to it, and
each test gets an isolated (truncated) database plus an HTTP client that
exercises the full app. Assertions go HTTP -> response AND HTTP -> row.
"""

import os
from pathlib import Path
from collections.abc import Iterator

import pytest
from fastapi.testclient import TestClient
from sqlalchemy import text
from sqlalchemy.orm import Session
from testcontainers.community.postgres import PostgresContainer

from mcp_inventory.app import create_app
from mcp_inventory.database import Database, make_engine

PROJECT_ROOT = Path(__file__).resolve().parents[1]

POSTGRES_IMAGE = "postgres:16-alpine"
DB_NAME = "mcp_inventory_test"


@pytest.fixture(scope="session")
def db_url() -> Iterator[str]:
    """Yield the PostgreSQL URL for the whole test session.

    By default a real `postgres:16-alpine` container is booted via
    testcontainers. If `MCPINVENTORY_TEST_DATABASE_URL` is set, that database
    is used instead (e.g. a locally installed PostgreSQL) — for environments
    without Docker. The suite still applies the Alembic chain to it.

    Warning: point this at a **disposable** database — the suite truncates the
    inventory tables before every test.
    """
    external = os.environ.get("MCPINVENTORY_TEST_DATABASE_URL")
    if external:
        yield external
        return

    with PostgresContainer(
        image=POSTGRES_IMAGE,
        username="mcp",
        password="mcp",
        dbname=DB_NAME,
        driver="psycopg",  # we depend on psycopg v3, not psycopg2
    ) as container:
        yield container.get_connection_url()


@pytest.fixture(scope="session")
def migrated_engine(db_url: str):
    """Apply the Alembic migration chain to the container DB, once per session."""
    from alembic import command
    from alembic.config import Config

    cfg = Config(str(PROJECT_ROOT / "alembic.ini"))
    cfg.set_main_option("script_location", str(PROJECT_ROOT / "alembic"))
    cfg.set_main_option("sqlalchemy.url", db_url)
    # env.py gives the MCPINVENTORY_DATABASE_URL env var precedence over the
    # ini url; hide it while upgrading so migrations always hit db_url.
    saved = os.environ.pop("MCPINVENTORY_DATABASE_URL", None)
    try:
        command.upgrade(cfg, "head")
    finally:
        if saved is not None:
            os.environ["MCPINVENTORY_DATABASE_URL"] = saved

    engine = make_engine(db_url)
    yield engine
    engine.dispose()


@pytest.fixture()
def session(migrated_engine) -> Iterator[Session]:
    """Raw DB session for asserting rows directly (the 'DB side' of round-trips)."""
    db = Database(migrated_engine)
    with db.session_scope() as s:
        yield s


@pytest.fixture(autouse=True)
def clean_tables(migrated_engine) -> None:
    """Truncate all inventory tables before every test for isolation."""
    with migrated_engine.begin() as conn:
        conn.execute(text("TRUNCATE claims, scopes, audiences RESTART IDENTITY CASCADE"))


@pytest.fixture()
def client(migrated_engine) -> Iterator[TestClient]:
    """HTTP client bound to the app wired against the container database."""
    app = create_app(Database(migrated_engine))
    with TestClient(app) as test_client:
        yield test_client

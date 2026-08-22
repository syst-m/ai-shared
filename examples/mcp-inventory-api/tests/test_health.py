"""Health probe + migration-state tests (proof the real DB is live)."""

import uuid
from datetime import datetime

from fastapi.testclient import TestClient
from sqlalchemy import select, text

from mcp_inventory import models
from mcp_inventory.app import create_app
from mcp_inventory.database import Database, make_engine


class TestHealth:
    def test_healthz_reports_ok_against_real_db(self, client):
        resp = client.get("/healthz")
        assert resp.status_code == 200
        assert resp.json() == {"status": "ok", "database": "ok"}

    def test_healthz_is_503_when_database_unreachable(self):
        """The probe degrades to 503 (not an unhandled 500) when the DB is down."""
        dead = Database(make_engine("postgresql+psycopg://mcp:mcp@127.0.0.1:59999/unreachable"))
        with TestClient(create_app(dead)) as offline_client:
            resp = offline_client.get("/healthz")
        assert resp.status_code == 503
        assert resp.json()["detail"] == "database unavailable"


class TestMigrations:
    def test_alembic_head_is_applied(self, migrated_engine):
        with migrated_engine.connect() as conn:
            row = conn.execute(text("SELECT version_num FROM alembic_version")).fetchone()
        assert row is not None
        assert row[0] == "0001"

    def test_timestamps_are_tz_aware(self, client, session):
        resp = client.post("/api/v1/audiences", json={"name": "ts-check", "description": "d"})
        body = resp.json()
        created = datetime.fromisoformat(body["created_at"])
        assert created.tzinfo is not None
        # and the stored row matches what the API returned
        row = session.scalar(select(models.Audience).where(models.Audience.name == "ts-check"))
        assert row.created_at == created

    def test_unknown_route_is_404(self, client):
        assert client.get("/nope").status_code == 404

    def test_malformed_uuid_is_422(self, client):
        assert client.get(f"/api/v1/audiences/{uuid.uuid4()}x").status_code == 422

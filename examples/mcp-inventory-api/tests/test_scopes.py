"""Scope CRUD — HTTP in, assertions on both the response and the DB row."""

import uuid

from sqlalchemy import func, select

from mcp_inventory import models
from .helpers import make_audience


class TestCreateScope:
    def test_create_returns_201_and_persists(self, client, session):
        aud = make_audience(client)
        resp = client.post(
            "/api/v1/scopes",
            json={"name": "mcp:tools:read", "description": "read tools", "audience_id": aud["id"]},
        )
        assert resp.status_code == 201, resp.text
        body = resp.json()
        assert set(body) == {"id", "name", "description", "audience_id", "created_at"}
        assert body["audience_id"] == aud["id"]

        row = session.get(models.Scope, uuid.UUID(body["id"]))
        assert row is not None
        assert row.name == "mcp:tools:read"
        assert row.audience_id == uuid.UUID(aud["id"])

    def test_create_for_missing_audience_is_404(self, client):
        resp = client.post("/api/v1/scopes", json={"name": "s1", "audience_id": str(uuid.uuid4())})
        assert resp.status_code == 404

    def test_create_duplicate_in_same_audience_is_409(self, client):
        aud = make_audience(client)
        payload = {"name": "mcp:tools:read", "audience_id": aud["id"]}
        assert client.post("/api/v1/scopes", json=payload).status_code == 201
        resp = client.post("/api/v1/scopes", json=payload)
        assert resp.status_code == 409

    def test_same_name_different_audience_is_allowed(self, client):
        a1 = make_audience(client, name="aud-one")
        a2 = make_audience(client, name="aud-two")
        assert client.post("/api/v1/scopes", json={"name": "read", "audience_id": a1["id"]}).status_code == 201
        assert client.post("/api/v1/scopes", json={"name": "read", "audience_id": a2["id"]}).status_code == 201

    def test_create_is_404_when_audience_vanishes_before_flush(self, client, monkeypatch):
        """Race window: audience exists at the pre-check but is gone by INSERT time.

        The FK violation from Postgres (SQLSTATE 23503) must surface as a 404,
        not be misreported as a name conflict.
        """
        from mcp_inventory import crud

        aud = make_audience(client)
        client.delete(f"/api/v1/audiences/{aud['id']}")  # gone by the time the INSERT flushes
        monkeypatch.setattr(crud, "get_audience_by_id", lambda db, audience_id: object())
        resp = client.post("/api/v1/scopes", json={"name": "s1", "audience_id": aud["id"]})
        assert resp.status_code == 404
        assert "not found" in resp.json()["detail"]

    def test_create_invalid_name_is_422(self, client):
        aud = make_audience(client)
        for bad in ["", "UPPER", "mcp :x", ":leading-colon", "s" * 201]:
            resp = client.post("/api/v1/scopes", json={"name": bad, "audience_id": aud["id"]})
            assert resp.status_code == 422, f"name {bad!r} should be rejected"


class TestReadScope:
    def test_list_all_and_filter_by_audience(self, client):
        a1 = make_audience(client, name="aud-one")
        a2 = make_audience(client, name="aud-two")
        client.post("/api/v1/scopes", json={"name": "s-a1", "audience_id": a1["id"]})
        client.post("/api/v1/scopes", json={"name": "s-a2", "audience_id": a2["id"]})

        all_names = {s["name"] for s in client.get("/api/v1/scopes").json()}
        assert all_names == {"s-a1", "s-a2"}

        filtered = client.get("/api/v1/scopes", params={"audience_id": a1["id"]}).json()
        assert [s["name"] for s in filtered] == ["s-a1"]

    def test_list_filter_missing_audience_is_404(self, client):
        resp = client.get("/api/v1/scopes", params={"audience_id": str(uuid.uuid4())})
        assert resp.status_code == 404

    def test_list_limit_bounds_are_422(self, client):
        assert client.get("/api/v1/scopes", params={"limit": 0}).status_code == 422
        assert client.get("/api/v1/scopes", params={"limit": 201}).status_code == 422

    def test_get_by_id_and_missing(self, client):
        aud = make_audience(client)
        created = client.post("/api/v1/scopes", json={"name": "s1", "audience_id": aud["id"]}).json()
        assert client.get(f"/api/v1/scopes/{created['id']}").status_code == 200
        assert client.get(f"/api/v1/scopes/{uuid.uuid4()}").status_code == 404


class TestUpdateScope:
    def test_patch(self, client, session):
        aud = make_audience(client)
        created = client.post("/api/v1/scopes", json={"name": "s1", "audience_id": aud["id"]}).json()
        resp = client.patch(f"/api/v1/scopes/{created['id']}", json={"name": "s1:renamed", "description": "d"})
        assert resp.status_code == 200
        assert resp.json()["name"] == "s1:renamed"
        assert resp.json()["description"] == "d"
        assert session.get(models.Scope, uuid.UUID(created["id"])).name == "s1:renamed"

    def test_patch_empty_body_is_noop(self, client):
        aud = make_audience(client)
        created = client.post("/api/v1/scopes", json={"name": "s1", "audience_id": aud["id"]}).json()
        resp = client.patch(f"/api/v1/scopes/{created['id']}", json={})
        assert resp.status_code == 200
        assert resp.json()["name"] == "s1"

    def test_patch_missing_is_404(self, client):
        assert client.patch(f"/api/v1/scopes/{uuid.uuid4()}", json={"name": "x"}).status_code == 404

    def test_patch_to_duplicate_name_is_409(self, client):
        aud = make_audience(client)
        client.post("/api/v1/scopes", json={"name": "dup", "audience_id": aud["id"]})
        other = client.post("/api/v1/scopes", json={"name": "other", "audience_id": aud["id"]}).json()
        resp = client.patch(f"/api/v1/scopes/{other['id']}", json={"name": "dup"})
        assert resp.status_code == 409


class TestDeleteScope:
    def test_delete_then_get_is_404(self, client, session):
        aud = make_audience(client)
        created = client.post("/api/v1/scopes", json={"name": "s1", "audience_id": aud["id"]}).json()
        assert client.delete(f"/api/v1/scopes/{created['id']}").status_code == 204
        assert client.get(f"/api/v1/scopes/{created['id']}").status_code == 404
        assert session.scalar(select(func.count()).select_from(models.Scope)) == 0

    def test_delete_missing_is_404(self, client):
        assert client.delete(f"/api/v1/scopes/{uuid.uuid4()}").status_code == 404

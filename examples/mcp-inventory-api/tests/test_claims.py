"""Claim CRUD — HTTP in, assertions on both the response and the DB row."""

import uuid

from sqlalchemy import func, select

from mcp_inventory import models
from .helpers import make_audience


class TestCreateClaim:
    def test_create_returns_201_and_persists(self, client, session):
        aud = make_audience(client)
        resp = client.post(
            "/api/v1/claims",
            json={"name": "mcp:server_id", "type": "string", "description": "server id", "audience_id": aud["id"]},
        )
        assert resp.status_code == 201, resp.text
        body = resp.json()
        assert set(body) == {"id", "name", "type", "description", "audience_id", "created_at"}
        assert body["type"] == "string"

        row = session.get(models.Claim, uuid.UUID(body["id"]))
        assert row is not None
        assert row.name == "mcp:server_id"
        assert row.type == "string"

    def test_type_defaults_to_string(self, client):
        aud = make_audience(client)
        body = client.post("/api/v1/claims", json={"name": "sub", "audience_id": aud["id"]}).json()
        assert body["type"] == "string"

    def test_each_claim_type_is_accepted(self, client):
        aud = make_audience(client)
        for i, t in enumerate(["string", "integer", "boolean", "json"]):
            body = client.post(
                "/api/v1/claims", json={"name": f"c{i}", "type": t, "audience_id": aud["id"]}
            ).json()
            assert body["type"] == t

    def test_create_for_missing_audience_is_404(self, client):
        resp = client.post("/api/v1/claims", json={"name": "c", "audience_id": str(uuid.uuid4())})
        assert resp.status_code == 404

    def test_create_duplicate_in_same_audience_is_409(self, client):
        aud = make_audience(client)
        payload = {"name": "sub", "audience_id": aud["id"]}
        assert client.post("/api/v1/claims", json=payload).status_code == 201
        assert client.post("/api/v1/claims", json=payload).status_code == 409

    def test_same_name_different_audience_is_allowed(self, client):
        a1 = make_audience(client, name="aud-one")
        a2 = make_audience(client, name="aud-two")
        assert client.post("/api/v1/claims", json={"name": "sub", "audience_id": a1["id"]}).status_code == 201
        assert client.post("/api/v1/claims", json={"name": "sub", "audience_id": a2["id"]}).status_code == 201

    def test_create_is_404_when_audience_vanishes_before_flush(self, client, monkeypatch):
        """Same FK race as for scopes: a 23503 at flush time must map to 404, not 409."""
        from mcp_inventory import crud

        aud = make_audience(client)
        client.delete(f"/api/v1/audiences/{aud['id']}")
        monkeypatch.setattr(crud, "get_audience_by_id", lambda db, audience_id: object())
        resp = client.post("/api/v1/claims", json={"name": "c1", "audience_id": aud["id"]})
        assert resp.status_code == 404
        assert "not found" in resp.json()["detail"]

    def test_create_invalid_name_or_type_is_422(self, client):
        aud = make_audience(client)
        assert client.post("/api/v1/claims", json={"name": "", "audience_id": aud["id"]}).status_code == 422
        assert client.post("/api/v1/claims", json={"name": "c", "type": "float", "audience_id": aud["id"]}).status_code == 422
        assert client.post("/api/v1/claims", json={"name": "9leading", "audience_id": aud["id"]}).status_code == 422


class TestReadClaim:
    def test_list_all_and_filter_by_audience(self, client):
        a1 = make_audience(client, name="aud-one")
        a2 = make_audience(client, name="aud-two")
        client.post("/api/v1/claims", json={"name": "c-a1", "audience_id": a1["id"]})
        client.post("/api/v1/claims", json={"name": "c-a2", "audience_id": a2["id"]})

        all_names = {c["name"] for c in client.get("/api/v1/claims").json()}
        assert all_names == {"c-a1", "c-a2"}

        filtered = client.get("/api/v1/claims", params={"audience_id": a1["id"]}).json()
        assert [c["name"] for c in filtered] == ["c-a1"]

    def test_list_filter_missing_audience_is_404(self, client):
        resp = client.get("/api/v1/claims", params={"audience_id": str(uuid.uuid4())})
        assert resp.status_code == 404

    def test_list_limit_bounds_are_422(self, client):
        assert client.get("/api/v1/claims", params={"limit": 0}).status_code == 422
        assert client.get("/api/v1/claims", params={"limit": 201}).status_code == 422

    def test_get_by_id_and_missing(self, client):
        aud = make_audience(client)
        created = client.post("/api/v1/claims", json={"name": "c1", "audience_id": aud["id"]}).json()
        assert client.get(f"/api/v1/claims/{created['id']}").status_code == 200
        assert client.get(f"/api/v1/claims/{uuid.uuid4()}").status_code == 404


class TestUpdateClaim:
    def test_patch_name_type_and_description(self, client, session):
        aud = make_audience(client)
        created = client.post("/api/v1/claims", json={"name": "c1", "audience_id": aud["id"]}).json()
        resp = client.patch(
            f"/api/v1/claims/{created['id']}", json={"name": "c2", "type": "boolean", "description": "flag"}
        )
        assert resp.status_code == 200
        assert resp.json() == {**created, "name": "c2", "type": "boolean", "description": "flag"}

        row = session.get(models.Claim, uuid.UUID(created["id"]))
        assert row.type == "boolean"

    def test_patch_empty_body_is_noop(self, client):
        aud = make_audience(client)
        created = client.post("/api/v1/claims", json={"name": "c1", "audience_id": aud["id"]}).json()
        resp = client.patch(f"/api/v1/claims/{created['id']}", json={})
        assert resp.status_code == 200
        assert resp.json()["name"] == "c1"

    def test_patch_missing_is_404(self, client):
        assert client.patch(f"/api/v1/claims/{uuid.uuid4()}", json={"name": "x"}).status_code == 404

    def test_patch_to_duplicate_name_is_409(self, client):
        aud = make_audience(client)
        client.post("/api/v1/claims", json={"name": "dup", "audience_id": aud["id"]})
        other = client.post("/api/v1/claims", json={"name": "other", "audience_id": aud["id"]}).json()
        resp = client.patch(f"/api/v1/claims/{other['id']}", json={"name": "dup"})
        assert resp.status_code == 409


class TestDeleteClaim:
    def test_delete_then_get_is_404(self, client, session):
        aud = make_audience(client)
        created = client.post("/api/v1/claims", json={"name": "c1", "audience_id": aud["id"]}).json()
        assert client.delete(f"/api/v1/claims/{created['id']}").status_code == 204
        assert client.get(f"/api/v1/claims/{created['id']}").status_code == 404
        assert session.scalar(select(func.count()).select_from(models.Claim)) == 0

    def test_delete_missing_is_404(self, client):
        assert client.delete(f"/api/v1/claims/{uuid.uuid4()}").status_code == 404

"""Audience CRUD — HTTP in, assertions on both the response and the DB row."""

import uuid

from sqlalchemy import func, select

from mcp_inventory import models
from .helpers import make_audience


class TestCreateAudience:
    def test_create_returns_201_and_persists(self, client, session):
        body = make_audience(client)
        assert set(body) == {"id", "name", "description", "created_at", "updated_at"}
        assert body["name"] == "github-mcp"
        uuid.UUID(body["id"])  # parses

        row = session.get(models.Audience, uuid.UUID(body["id"]))
        assert row is not None
        assert row.name == "github-mcp"
        assert row.description == "GitHub MCP server"

    def test_create_duplicate_name_is_409(self, client):
        make_audience(client)
        resp = client.post("/api/v1/audiences", json={"name": "github-mcp"})
        assert resp.status_code == 409
        assert "already exists" in resp.json()["detail"]

    def test_create_invalid_names_are_422(self, client):
        for bad in ["", "UPPER", "has space", "-leading", "x" * 201, "a/b"]:
            resp = client.post("/api/v1/audiences", json={"name": bad})
            assert resp.status_code == 422, f"name {bad!r} should be rejected"

    def test_create_description_defaults_to_empty(self, client):
        resp = client.post("/api/v1/audiences", json={"name": "no-desc"})
        assert resp.status_code == 201
        assert resp.json()["description"] == ""


class TestReadAudience:
    def test_list_empty(self, client):
        assert client.get("/api/v1/audiences").json() == []

    def test_list_returns_created(self, client):
        make_audience(client, name="a-one")
        make_audience(client, name="a-two")
        names = [a["name"] for a in client.get("/api/v1/audiences").json()]
        assert names == ["a-one", "a-two"]

    def test_list_pagination(self, client):
        for i in range(5):
            make_audience(client, name=f"page-{i}")
        first = client.get("/api/v1/audiences", params={"limit": 2}).json()
        rest = client.get("/api/v1/audiences", params={"limit": 2, "offset": 4}).json()
        assert [a["name"] for a in first] == ["page-0", "page-1"]
        assert [a["name"] for a in rest] == ["page-4"]

    def test_list_limit_bounds_are_422(self, client):
        assert client.get("/api/v1/audiences", params={"limit": 0}).status_code == 422
        assert client.get("/api/v1/audiences", params={"limit": 201}).status_code == 422
        assert client.get("/api/v1/audiences", params={"offset": -1}).status_code == 422

    def test_get_by_id(self, client):
        body = make_audience(client)
        resp = client.get(f"/api/v1/audiences/{body['id']}")
        assert resp.status_code == 200
        assert resp.json()["id"] == body["id"]

    def test_get_missing_is_404(self, client):
        assert client.get(f"/api/v1/audiences/{uuid.uuid4()}").status_code == 404


class TestUpdateAudience:
    def test_patch_name_and_description(self, client, session):
        body = make_audience(client)
        resp = client.patch(f"/api/v1/audiences/{body['id']}", json={"name": "new-name", "description": "d2"})
        assert resp.status_code == 200
        assert resp.json()["name"] == "new-name"
        assert resp.json()["description"] == "d2"

        row = session.get(models.Audience, uuid.UUID(body["id"]))
        assert row.name == "new-name"
        assert row.updated_at >= row.created_at

    def test_patch_empty_body_is_noop(self, client):
        body = make_audience(client)
        resp = client.patch(f"/api/v1/audiences/{body['id']}", json={})
        assert resp.status_code == 200
        assert resp.json()["name"] == "github-mcp"
        assert resp.json()["description"] == "GitHub MCP server"

    def test_patch_missing_is_404(self, client):
        resp = client.patch(f"/api/v1/audiences/{uuid.uuid4()}", json={"name": "x"})
        assert resp.status_code == 404

    def test_patch_to_duplicate_name_is_409(self, client):
        make_audience(client, name="keep-me")
        other = make_audience(client, name="rename-me")
        resp = client.patch(f"/api/v1/audiences/{other['id']}", json={"name": "keep-me"})
        assert resp.status_code == 409


class TestDeleteAudience:
    def test_delete_then_get_is_404(self, client, session):
        body = make_audience(client)
        assert client.delete(f"/api/v1/audiences/{body['id']}").status_code == 204
        assert client.get(f"/api/v1/audiences/{body['id']}").status_code == 404
        assert session.scalar(select(func.count()).select_from(models.Audience)) == 0

    def test_delete_missing_is_404(self, client):
        assert client.delete(f"/api/v1/audiences/{uuid.uuid4()}").status_code == 404

    def test_delete_cascades_to_scopes_and_claims(self, client, session):
        body = make_audience(client)
        aud_id = body["id"]
        assert client.post("/api/v1/scopes", json={"name": "mcp:tools:read", "audience_id": aud_id}).status_code == 201
        assert client.post("/api/v1/claims", json={"name": "sub", "audience_id": aud_id}).status_code == 201

        assert client.delete(f"/api/v1/audiences/{aud_id}").status_code == 204

        assert session.execute(select(func.count()).select_from(models.Scope)).scalar() == 0
        assert session.execute(select(func.count()).select_from(models.Claim)).scalar() == 0

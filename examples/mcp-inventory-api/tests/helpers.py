"""Shared test helpers."""


def make_audience(client, name: str = "github-mcp", description: str = "GitHub MCP server") -> dict:
    """Create an audience through the HTTP layer; return the response body."""
    resp = client.post("/api/v1/audiences", json={"name": name, "description": description})
    assert resp.status_code == 201, resp.text
    return resp.json()

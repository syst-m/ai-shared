"""FastAPI application factory."""

from fastapi import FastAPI

from . import __version__
from .api import audiences, claims, health, scopes
from .database import Database, get_database


def create_app(database: Database | None = None) -> FastAPI:
    """Build the app. Pass a custom `Database` in tests; production uses settings."""
    db = database or get_database()

    app = FastAPI(
        title="MCP Inventory API",
        version=__version__,
        description="Database-backed inventory of MCP metadata: audiences, claims, and scopes.",
    )
    app.state.db = db

    app.include_router(health.router)
    app.include_router(audiences.router, prefix="/api/v1")
    app.include_router(scopes.router, prefix="/api/v1")
    app.include_router(claims.router, prefix="/api/v1")

    return app

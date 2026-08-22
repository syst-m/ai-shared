"""Shared FastAPI dependencies and error helpers."""

import uuid
from collections.abc import Iterator

from fastapi import HTTPException, Request, status
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from .. import crud
from ..database import Database


def get_db(request: Request) -> Iterator[Session]:
    """Yield a session bound to the app's Database (set by `create_app`)."""
    db: Database = request.app.state.db
    with db.session_scope() as session:
        yield session


def require_audience(db: Session, audience_id: uuid.UUID) -> None:
    """Raise 404 if the referenced audience does not exist (avoids FK violations)."""
    if crud.get_audience_by_id(db, audience_id) is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"audience {audience_id} not found")


def is_fk_violation(exc: IntegrityError) -> bool:
    """True if the IntegrityError is a PostgreSQL foreign-key violation (SQLSTATE 23503)."""
    orig = getattr(exc, "orig", None)
    code = getattr(orig, "sqlstate", None) or getattr(orig, "pgcode", None)
    return code == "23503"

"""Shared FastAPI dependencies and error helpers."""

import uuid
from collections.abc import Iterator

from fastapi import HTTPException, Request, status
from pydantic import BaseModel
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


def reject_null_update_fields(payload: BaseModel) -> None:
    """Raise 422 if an explicitly provided field is ``null``.

    All updatable columns are NOT NULL, so a ``null`` value can never be stored;
    without this check it would surface later as a NOT NULL ``IntegrityError``
    misreported by the router as a 409 duplicate.
    """
    null_fields = sorted(f for f in payload.model_fields_set if getattr(payload, f) is None)
    if null_fields:
        raise HTTPException(
            status_code=422,  # literal: HTTP_422_* constant names differ across FastAPI versions
            detail=f"cannot be null: {', '.join(null_fields)}",
        )

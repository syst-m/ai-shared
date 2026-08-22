"""Scope CRUD endpoints."""

import uuid

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from .. import crud, models, schemas
from .deps import get_db, is_fk_violation, require_audience

router = APIRouter(prefix="/scopes", tags=["scopes"])


@router.get("", response_model=list[schemas.ScopeRead])
def list_scopes(
    audience_id: uuid.UUID | None = Query(default=None),
    limit: int = Query(default=50, ge=1, le=200),
    offset: int = Query(default=0, ge=0),
    db: Session = Depends(get_db),
) -> list[models.Scope]:
    if audience_id is not None:
        require_audience(db, audience_id)
    return crud.list_scopes(db, audience_id=audience_id, limit=limit, offset=offset)


@router.post("", response_model=schemas.ScopeRead, status_code=status.HTTP_201_CREATED)
def create_scope(payload: schemas.ScopeCreate, db: Session = Depends(get_db)) -> models.Scope:
    require_audience(db, payload.audience_id)
    try:
        return crud.create_scope(db, payload)
    except IntegrityError as exc:
        # FK violations only happen if the audience vanished between the check and the flush.
        if is_fk_violation(exc):
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND, detail=f"audience {payload.audience_id} not found"
            ) from exc
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail=f"scope '{payload.name}' already exists for audience {payload.audience_id}",
        ) from exc


@router.get("/{scope_id}", response_model=schemas.ScopeRead)
def get_scope(scope_id: uuid.UUID, db: Session = Depends(get_db)) -> models.Scope:
    scope = crud.get_scope(db, scope_id)
    if scope is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"scope {scope_id} not found")
    return scope


@router.patch("/{scope_id}", response_model=schemas.ScopeRead)
def update_scope(scope_id: uuid.UUID, payload: schemas.ScopeUpdate, db: Session = Depends(get_db)) -> models.Scope:
    scope = crud.get_scope(db, scope_id)
    if scope is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"scope {scope_id} not found")
    try:
        return crud.update_scope(db, scope, payload)
    except IntegrityError as exc:
        raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail=f"scope '{payload.name}' already exists") from exc


@router.delete("/{scope_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_scope(scope_id: uuid.UUID, db: Session = Depends(get_db)) -> None:
    scope = crud.get_scope(db, scope_id)
    if scope is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"scope {scope_id} not found")
    crud.delete_scope(db, scope)

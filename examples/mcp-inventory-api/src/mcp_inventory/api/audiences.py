"""Audience CRUD endpoints."""

import uuid

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from .. import crud, models, schemas
from .deps import get_db

router = APIRouter(prefix="/audiences", tags=["audiences"])


def _not_found(audience_id: uuid.UUID) -> HTTPException:
    return HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"audience {audience_id} not found")


@router.get("", response_model=list[schemas.AudienceRead])
def list_audiences(
    limit: int = Query(default=50, ge=1, le=200),
    offset: int = Query(default=0, ge=0),
    db: Session = Depends(get_db),
) -> list[models.Audience]:
    return crud.list_audiences(db, limit=limit, offset=offset)


@router.post("", response_model=schemas.AudienceRead, status_code=status.HTTP_201_CREATED)
def create_audience(payload: schemas.AudienceCreate, db: Session = Depends(get_db)) -> models.Audience:
    try:
        return crud.create_audience(db, payload)
    except IntegrityError as exc:
        raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail=f"audience '{payload.name}' already exists") from exc


@router.get("/{audience_id}", response_model=schemas.AudienceRead)
def get_audience(audience_id: uuid.UUID, db: Session = Depends(get_db)) -> models.Audience:
    audience = crud.get_audience_by_id(db, audience_id)
    if audience is None:
        raise _not_found(audience_id)
    return audience


@router.patch("/{audience_id}", response_model=schemas.AudienceRead)
def update_audience(audience_id: uuid.UUID, payload: schemas.AudienceUpdate, db: Session = Depends(get_db)) -> models.Audience:
    audience = crud.get_audience_by_id(db, audience_id)
    if audience is None:
        raise _not_found(audience_id)
    try:
        return crud.update_audience(db, audience, payload)
    except IntegrityError as exc:
        conflicting = payload.name if payload.name is not None else "an existing name"
        raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail=f"audience '{conflicting}' already exists") from exc


@router.delete("/{audience_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_audience(audience_id: uuid.UUID, db: Session = Depends(get_db)) -> None:
    """Deletes the audience and cascades to its scopes and claims."""
    audience = crud.get_audience_by_id(db, audience_id)
    if audience is None:
        raise _not_found(audience_id)
    crud.delete_audience(db, audience)

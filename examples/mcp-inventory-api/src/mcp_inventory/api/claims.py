"""Claim CRUD endpoints."""

import uuid

from fastapi import APIRouter, Depends, HTTPException, Query, status
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from .. import crud, models, schemas
from .deps import get_db, is_fk_violation, require_audience

router = APIRouter(prefix="/claims", tags=["claims"])


@router.get("", response_model=list[schemas.ClaimRead])
def list_claims(
    audience_id: uuid.UUID | None = Query(default=None),
    limit: int = Query(default=50, ge=1, le=200),
    offset: int = Query(default=0, ge=0),
    db: Session = Depends(get_db),
) -> list[models.Claim]:
    if audience_id is not None:
        require_audience(db, audience_id)
    return crud.list_claims(db, audience_id=audience_id, limit=limit, offset=offset)


@router.post("", response_model=schemas.ClaimRead, status_code=status.HTTP_201_CREATED)
def create_claim(payload: schemas.ClaimCreate, db: Session = Depends(get_db)) -> models.Claim:
    require_audience(db, payload.audience_id)
    try:
        return crud.create_claim(db, payload)
    except IntegrityError as exc:
        # FK violations only happen if the audience vanished between the check and the flush.
        if is_fk_violation(exc):
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND, detail=f"audience {payload.audience_id} not found"
            ) from exc
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail=f"claim '{payload.name}' already exists for audience {payload.audience_id}",
        ) from exc


@router.get("/{claim_id}", response_model=schemas.ClaimRead)
def get_claim(claim_id: uuid.UUID, db: Session = Depends(get_db)) -> models.Claim:
    claim = crud.get_claim(db, claim_id)
    if claim is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"claim {claim_id} not found")
    return claim


@router.patch("/{claim_id}", response_model=schemas.ClaimRead)
def update_claim(claim_id: uuid.UUID, payload: schemas.ClaimUpdate, db: Session = Depends(get_db)) -> models.Claim:
    claim = crud.get_claim(db, claim_id)
    if claim is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"claim {claim_id} not found")
    try:
        return crud.update_claim(db, claim, payload)
    except IntegrityError as exc:
        raise HTTPException(status_code=status.HTTP_409_CONFLICT, detail=f"claim '{payload.name}' already exists") from exc


@router.delete("/{claim_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_claim(claim_id: uuid.UUID, db: Session = Depends(get_db)) -> None:
    claim = crud.get_claim(db, claim_id)
    if claim is None:
        raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"claim {claim_id} not found")
    crud.delete_claim(db, claim)

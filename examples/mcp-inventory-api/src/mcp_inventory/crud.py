"""Data-access functions.

Routers look up by id and raise 404 when a row is missing; on create/patch they
catch the `IntegrityError` raised by `flush` and map a unique-violation to 409
(and, for children, a stale FK to 404). All writes are flushed here and committed
by the `session_scope` context manager in `deps.get_db`.
"""

import uuid

from sqlalchemy import select
from sqlalchemy.orm import Session

from . import models, schemas


# --- audiences ----------------------------------------------------------------


def get_audience_by_id(session: Session, audience_id: uuid.UUID) -> models.Audience | None:
    return session.get(models.Audience, audience_id)


def list_audiences(session: Session, *, limit: int = 50, offset: int = 0) -> list[models.Audience]:
    stmt = select(models.Audience).order_by(models.Audience.created_at, models.Audience.name).limit(limit).offset(offset)
    return list(session.scalars(stmt))


def create_audience(session: Session, payload: schemas.AudienceCreate) -> models.Audience:
    audience = models.Audience(name=payload.name, description=payload.description)
    session.add(audience)
    session.flush()
    return audience


def update_audience(session: Session, audience: models.Audience, payload: schemas.AudienceUpdate) -> models.Audience:
    data = payload.model_dump(exclude_unset=True)
    for field, value in data.items():
        setattr(audience, field, value)
    session.flush()
    return audience


def delete_audience(session: Session, audience: models.Audience) -> None:
    session.delete(audience)
    session.flush()


# --- scopes -------------------------------------------------------------------


def get_scope(session: Session, scope_id: uuid.UUID) -> models.Scope | None:
    return session.get(models.Scope, scope_id)


def list_scopes(
    session: Session, *, audience_id: uuid.UUID | None = None, limit: int = 50, offset: int = 0
) -> list[models.Scope]:
    stmt = select(models.Scope).order_by(models.Scope.created_at, models.Scope.name).limit(limit).offset(offset)
    if audience_id is not None:
        stmt = stmt.where(models.Scope.audience_id == audience_id)
    return list(session.scalars(stmt))


def create_scope(session: Session, payload: schemas.ScopeCreate) -> models.Scope:
    scope = models.Scope(name=payload.name, description=payload.description, audience_id=payload.audience_id)
    session.add(scope)
    session.flush()
    return scope


def update_scope(session: Session, scope: models.Scope, payload: schemas.ScopeUpdate) -> models.Scope:
    data = payload.model_dump(exclude_unset=True)
    for field, value in data.items():
        setattr(scope, field, value)
    session.flush()
    return scope


def delete_scope(session: Session, scope: models.Scope) -> None:
    session.delete(scope)
    session.flush()


# --- claims -------------------------------------------------------------------


def get_claim(session: Session, claim_id: uuid.UUID) -> models.Claim | None:
    return session.get(models.Claim, claim_id)


def list_claims(
    session: Session, *, audience_id: uuid.UUID | None = None, limit: int = 50, offset: int = 0
) -> list[models.Claim]:
    stmt = select(models.Claim).order_by(models.Claim.created_at, models.Claim.name).limit(limit).offset(offset)
    if audience_id is not None:
        stmt = stmt.where(models.Claim.audience_id == audience_id)
    return list(session.scalars(stmt))


def create_claim(session: Session, payload: schemas.ClaimCreate) -> models.Claim:
    claim = models.Claim(
        name=payload.name, type=payload.type.value, description=payload.description, audience_id=payload.audience_id
    )
    session.add(claim)
    session.flush()
    return claim


def update_claim(session: Session, claim: models.Claim, payload: schemas.ClaimUpdate) -> models.Claim:
    data = payload.model_dump(exclude_unset=True)
    if "type" in data and data["type"] is not None:
        data["type"] = data["type"].value  # store the plain string form
    for field, value in data.items():
        setattr(claim, field, value)
    session.flush()
    return claim


def delete_claim(session: Session, claim: models.Claim) -> None:
    session.delete(claim)
    session.flush()

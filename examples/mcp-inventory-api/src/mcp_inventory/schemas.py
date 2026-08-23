"""Pydantic request/response schemas."""

import uuid
from datetime import datetime
from enum import Enum

from pydantic import BaseModel, ConfigDict, Field

# --- shared validation rules -------------------------------------------------

AUDIENCE_NAME_PATTERN = r"^[a-z0-9][a-z0-9._-]{0,199}$"
SCOPE_NAME_PATTERN = r"^[a-z][a-z0-9._:-]{0,199}$"
CLAIM_NAME_PATTERN = r"^[a-zA-Z][a-zA-Z0-9._:-]{0,199}$"


class ClaimType(str, Enum):
    STRING = "string"
    INTEGER = "integer"
    BOOLEAN = "boolean"
    JSON = "json"


# --- audiences ----------------------------------------------------------------


class AudienceCreate(BaseModel):
    name: str = Field(min_length=1, max_length=200, pattern=AUDIENCE_NAME_PATTERN)
    description: str = Field(default="", max_length=2000)


class AudienceUpdate(BaseModel):
    name: str | None = Field(default=None, min_length=1, max_length=200, pattern=AUDIENCE_NAME_PATTERN)
    description: str | None = Field(default=None, max_length=2000)


class AudienceRead(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: uuid.UUID
    name: str
    description: str
    created_at: datetime
    updated_at: datetime


# --- scopes -------------------------------------------------------------------


class ScopeCreate(BaseModel):
    name: str = Field(min_length=1, max_length=200, pattern=SCOPE_NAME_PATTERN)
    description: str = Field(default="", max_length=2000)
    audience_id: uuid.UUID


class ScopeUpdate(BaseModel):
    name: str | None = Field(default=None, min_length=1, max_length=200, pattern=SCOPE_NAME_PATTERN)
    description: str | None = Field(default=None, max_length=2000)


class ScopeRead(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: uuid.UUID
    name: str
    description: str
    audience_id: uuid.UUID
    created_at: datetime


# --- claims -------------------------------------------------------------------


class ClaimCreate(BaseModel):
    name: str = Field(min_length=1, max_length=200, pattern=CLAIM_NAME_PATTERN)
    type: ClaimType = ClaimType.STRING
    description: str = Field(default="", max_length=2000)
    audience_id: uuid.UUID


class ClaimUpdate(BaseModel):
    name: str | None = Field(default=None, min_length=1, max_length=200, pattern=CLAIM_NAME_PATTERN)
    type: ClaimType | None = None
    description: str | None = Field(default=None, max_length=2000)


class ClaimRead(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: uuid.UUID
    name: str
    type: ClaimType
    description: str
    audience_id: uuid.UUID
    created_at: datetime

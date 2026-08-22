"""ORM models for the MCP metadata inventory."""

import uuid
from datetime import datetime

from sqlalchemy import DateTime, ForeignKey, String, Text, UniqueConstraint, func
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import Mapped, mapped_column, relationship

from .database import Base


def _uuid() -> uuid.UUID:
    return uuid.uuid4()


class Audience(Base):
    """A registered MCP audience (e.g. a server or tool namespace)."""

    __tablename__ = "audiences"

    id: Mapped[uuid.UUID] = mapped_column(UUID(as_uuid=True), primary_key=True, default=_uuid)
    name: Mapped[str] = mapped_column(String(200), unique=True, index=True)
    description: Mapped[str] = mapped_column(Text, default="")
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    scopes: Mapped[list["Scope"]] = relationship(back_populates="audience", cascade="all, delete-orphan")
    claims: Mapped[list["Claim"]] = relationship(back_populates="audience", cascade="all, delete-orphan")


class Scope(Base):
    """An access scope granted within an audience (e.g. ``mcp:tools:read``)."""

    __tablename__ = "scopes"
    __table_args__ = (UniqueConstraint("name", "audience_id", name="uq_scopes_audience_name"),)

    id: Mapped[uuid.UUID] = mapped_column(UUID(as_uuid=True), primary_key=True, default=_uuid)
    name: Mapped[str] = mapped_column(String(200), index=True)
    description: Mapped[str] = mapped_column(Text, default="")
    audience_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("audiences.id", ondelete="CASCADE"), index=True
    )
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())

    audience: Mapped[Audience] = relationship(back_populates="scopes")


class Claim(Base):
    """A protocol claim required by an audience (e.g. ``sub`` or ``mcp:server_id``)."""

    __tablename__ = "claims"
    __table_args__ = (UniqueConstraint("name", "audience_id", name="uq_claims_audience_name"),)

    id: Mapped[uuid.UUID] = mapped_column(UUID(as_uuid=True), primary_key=True, default=_uuid)
    name: Mapped[str] = mapped_column(String(200), index=True)
    type: Mapped[str] = mapped_column(String(32), default="string")
    description: Mapped[str] = mapped_column(Text, default="")
    audience_id: Mapped[uuid.UUID] = mapped_column(
        UUID(as_uuid=True), ForeignKey("audiences.id", ondelete="CASCADE"), index=True
    )
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), server_default=func.now())

    audience: Mapped[Audience] = relationship(back_populates="claims")

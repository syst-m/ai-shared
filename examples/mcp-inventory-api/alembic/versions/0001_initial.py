"""Initial schema: audiences, scopes, claims.

Revision ID: 0001
Revises:
Create Date: 2026-08-21

"""
import sqlalchemy as sa
from alembic import op
from sqlalchemy.dialects import postgresql

# revision identifiers, used by Alembic.
revision = "0001"
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "audiences",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("name", sa.String(length=200), nullable=False),
        sa.Column("description", sa.Text(), nullable=False, server_default=""),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.Column("updated_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
    )
    op.create_index("ix_audiences_name", "audiences", ["name"], unique=True)

    op.create_table(
        "scopes",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("name", sa.String(length=200), nullable=False),
        sa.Column("description", sa.Text(), nullable=False, server_default=""),
        sa.Column("audience_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.ForeignKeyConstraint(["audience_id"], ["audiences.id"], ondelete="CASCADE"),
        sa.UniqueConstraint("name", "audience_id", name="uq_scopes_audience_name"),
    )
    op.create_index("ix_scopes_name", "scopes", ["name"])
    op.create_index("ix_scopes_audience_id", "scopes", ["audience_id"])

    op.create_table(
        "claims",
        sa.Column("id", postgresql.UUID(as_uuid=True), primary_key=True),
        sa.Column("name", sa.String(length=200), nullable=False),
        sa.Column("type", sa.String(length=32), nullable=False, server_default="string"),
        sa.Column("description", sa.Text(), nullable=False, server_default=""),
        sa.Column("audience_id", postgresql.UUID(as_uuid=True), nullable=False),
        sa.Column("created_at", sa.DateTime(timezone=True), server_default=sa.text("now()"), nullable=False),
        sa.ForeignKeyConstraint(["audience_id"], ["audiences.id"], ondelete="CASCADE"),
        sa.UniqueConstraint("name", "audience_id", name="uq_claims_audience_name"),
    )
    op.create_index("ix_claims_name", "claims", ["name"])
    op.create_index("ix_claims_audience_id", "claims", ["audience_id"])


def downgrade() -> None:
    op.drop_table("claims")
    op.drop_table("scopes")
    op.drop_table("audiences")

"""Alembic migration environment (sync engine; psycopg driver)."""

import os
from logging.config import fileConfig

from alembic import context
from sqlalchemy import engine_from_config, pool

from mcp_inventory import models  # noqa: F401  (register models on Base.metadata)
from mcp_inventory.database import Base

config = context.config

if config.config_file_name is not None:
    fileConfig(config.config_file_name)

# Env var wins over alembic.ini so tests and containers can point anywhere.
url = os.environ.get("MCPINVENTORY_DATABASE_URL") or config.get_main_option("sqlalchemy.url")
if url:
    config.set_main_option("sqlalchemy.url", url)

target_metadata = Base.metadata


def run_migrations_offline() -> None:
    """Run migrations in 'offline' mode (emit SQL to stdout)."""
    context.configure(
        url=url,
        target_metadata=target_metadata,
        literal_binds=True,
        dialect_opts={"paramstyle": "named"},
    )
    with context.begin_transaction():
        context.run_migrations()


def run_migrations_online() -> None:
    """Run migrations in 'online' mode (live DB connection)."""
    connectable = engine_from_config(
        config.get_section(config.config_ini_section, {}),
        prefix="sqlalchemy.",
        poolclass=pool.NullPool,
    )
    with connectable.connect() as connection:
        context.configure(
            connection=connection,
            target_metadata=target_metadata,
        )
        with context.begin_transaction():
            context.run_migrations()


if context.is_offline_mode():
    run_migrations_offline()
else:
    run_migrations_online()

"""SQLAlchemy engine/session plumbing shared by the app and the tests."""

from collections.abc import Iterator
from contextlib import contextmanager
from functools import lru_cache

from sqlalchemy import create_engine
from sqlalchemy.engine import Engine
from sqlalchemy.orm import DeclarativeBase, Session, sessionmaker

from .config import get_settings


class Base(DeclarativeBase):
    pass


def make_engine(database_url: str) -> Engine:
    return create_engine(database_url, pool_pre_ping=True)


class Database:
    """Bundles an engine with a session factory so the app and tests share one connection path."""

    def __init__(self, engine: Engine) -> None:
        self.engine = engine
        self.Session = sessionmaker(bind=engine, expire_on_commit=False)

    @contextmanager
    def session_scope(self) -> Iterator[Session]:
        """Yield a session; commit on success, roll back on error, always close."""
        session = self.Session()
        try:
            yield session
            session.commit()
        except Exception:
            session.rollback()
            raise
        finally:
            session.close()


@lru_cache
def get_database() -> Database:
    settings = get_settings()
    return Database(make_engine(settings.database_url))

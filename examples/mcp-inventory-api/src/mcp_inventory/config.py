"""Runtime configuration, loaded from environment variables."""

from functools import lru_cache

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Environment-driven settings (prefix ``MCPINVENTORY_``)."""

    model_config = SettingsConfigDict(env_prefix="MCPINVENTORY_", env_file=".env", extra="ignore")

    database_url: str = "postgresql+psycopg://mcp:mcp@localhost:5432/mcp_inventory"
    log_level: str = "INFO"


@lru_cache
def get_settings() -> Settings:
    return Settings()

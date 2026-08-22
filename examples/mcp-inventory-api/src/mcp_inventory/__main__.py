"""CLI entry point: `python -m mcp_inventory` or `mcp-inventory-api`."""

import logging

import uvicorn

from .config import get_settings


def main() -> None:
    logging.basicConfig(level=get_settings().log_level.upper())
    uvicorn.run("mcp_inventory.app:create_app", factory=True, host="0.0.0.0", port=8000)


if __name__ == "__main__":
    main()

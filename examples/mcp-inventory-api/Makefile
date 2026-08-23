PY ?= python3
VENV := .venv
BIN := $(VENV)/bin

.PHONY: install migrate run test coverage docker-build clean

## Create the venv and install the app + test extras.
install:
	$(PY) -m venv $(VENV)
	$(BIN)/pip install -q -e ".[test]"

## Apply Alembic migrations to MCPINVENTORY_DATABASE_URL (or the default).
migrate: install
	$(BIN)/alembic upgrade head

## Run the API on :8000.
run: install
	$(BIN)/uvicorn mcp_inventory.app:create_app --factory --reload --port 8000

## Run the full test suite against a testcontainers PostgreSQL.
test: install
	$(BIN)/pytest

## Run tests with a coverage report.
coverage: install
	$(BIN)/pytest --cov=mcp_inventory --cov-report=term-missing

## Build the deployable image.
docker-build:
	docker build -t mcp-inventory-api:0.1.0 .

## Remove venv and build artifacts.
clean:
	rm -rf $(VENV) .pytest_cache htmlcov .coverage
	find . -name __pycache__ -type d -exec rm -rf {} +

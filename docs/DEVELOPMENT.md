# Development Guide

This guide covers setup, testing, and contribution workflows.

## 🛠 Prerequisites
- **Go**: 1.25+
- **Docker**: For running Postgres/Redis dependencies
- **Make**: For automation

## 🚀 Quick Start
```bash
# 1. Download dependencies
make deps

# 2. Start Infrastructure (Postgres, Redis)
make docker-up

# 3. Run the Service
make run
```

## 🏃 Running Locally (Manual)
If you want to run services individually without Docker Compose for the API:

1.  **Start Dependencies (DB & Redis)**:
    ```bash
    docker-compose up -d postgres redis
    ```
2.  **Start Mock Providers**:
    ```bash
    make mock
    ```
3.  **Run API**:
    ```bash
    make run
    ```

## 🧪 Testing

We use a comprehensive testing strategy:

| Type | Command | Description |
|------|---------|-------------|
| **Unit** | `make test-unit` | Fast, isolated tests (Mocks) |
| **Integration** | `make test-integration` | Real DB/Redis tests (Testcontainers) |
| **Coverage** | `make coverage` | Generate HTML report |
| **Lint** | `make lint` | Run `golangci-lint` |

### Key Test Implementation
- **Ranking**: `TestScoring` in `internal/infra/postgres` verifies the hybrid algorithm against a real DB.
- **Validation**: Table-driven tests for request validation.

## 💻 Contribution Workflow

1.  **Crawl/Update**: Modifying `internal/domain` requires updating `internal/infra/postgres` and `mocks`.
2.  **Helpers**: Use `make mock` to start local mock servers for Provider A/B.
3.  **Conventions**:
    - **Linting**: Must pass `golangci-lint` with zero warnings.
    - **Commits**: Use Conventional Commits (e.g., `feat: add distributed lock`).

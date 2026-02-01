# Search Engine Service - Documentation Index

## Overview
The **Search Engine Service** is a high-performance microservice designed to aggregate, rank, and serve content. It solves the problem of "viral but irrelevant" content by implementing a **Hybrid Ranking Algorithm** that balances semantic text relevance (FTS) with engagement metrics.

## 🌟 Key Features
- **Hybrid Ranking**: Balances `ts_rank` (Title: A, Tags: B) with logarithmic popularity.
- **Distributed Consistency**: Uses `redsync` distributed locks for safe background synchronization.
- **Resilience**: Per-provider **Circuit Breakers** (`sony/gobreaker`) to handle external failures.
- **Performance**: Redis caching for hot searches and optimized GIN indexes for Postgres FTS.
- **Observability**: Structured logs (Zap) and Health Checks (`/livez`, `/readyz`).

---

## 📚 Documentation Map

| Document | Description |
|----------|-------------|
| [**Architecture & Design**](ARCHITECTURE.md) | Deep dive into the Hybrid Ranking Logic, Circuit Breakers, and Data Flow. |
| [**API Reference**](API.md) | Comprehensive list of endpoints, request parameters, and response examples. |
| [**Configuration**](CONFIGURATION.md) | Env vars, Viper settings, and Secrets. |
| [**Development**](DEVELOPMENT.md) | Setup, testing strategies, and contribution guide. |
| [**Deployment**](DEPLOYMENT.md) | Docker, Kubernetes specs, and Observability. |

---

## 🛠 Technology Stack

| Component | Technology | Role |
|-----------|------------|------|
| **Core** | Go 1.25 | Business Logic & Concurrency |
| **Web** | Fiber v2 | High-performance HTTP transport |
| **Database** | PostgreSQL | Persistence & Full-Text Search (tsvector) |
| **Cache/Lock** | Redis | Caching & Distributed Locking (RedSync) |
| **Resilience** | sony/gobreaker | Circuit Breaker Pattern |
| **Config** | Viper |

---

## 🚀 Quick Start

### 1. Prerequisites
- Docker & Docker Compose
- Go 1.25+

### 2. Quick Start (Docker)
```bash
# Start the full stack (API, DB, Redis, Mock Providers)
make docker-up

# Verify
curl http://localhost:8080/livez
```

### 3. Local Development
If you prefer running the app locally (`go run`):

```bash
# 1. Start Database & Redis only
docker-compose up -d postgres redis

# 2. Start Mock Providers
make mock

# 3. Run the API
make run
```

---

## 📁 Project Structure

```
search-engine-service/
├── cmd/api/            # Entry point (Main)
├── internal/
│   ├── app/            # Application Services (Search, Sync)
│   ├── config/         # Configuration logic
│   ├── domain/         # Core Domain Entities & Interfaces
│   ├── infra/          # Infrastructure Adapters (Postgres, Redis)
│   ├── job/            # Background Workers
│   ├── logger/         # Logging setup
│   ├── transport/      # HTTP Handlers & Middlewares
│   └── validator/      # Request Validation
├── pkg/
│   └── locker/         # Reusable Distributed Lock pkg
├── api/                # OpenAPI specs
├── mock/               # Mock Providers for testing
├── web/                # Dashboard assets
└── docs/               # This documentation folder
```

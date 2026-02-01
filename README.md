# Search Engine Service

![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8)
![Fiber](https://img.shields.io/badge/Fiber-v2.52.10-00ADD8)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-Database-336791)
![Redis](https://img.shields.io/badge/Redis-Cache%2FLock-DC382D)

<p align="center">
  <img src="./gopher.png" width="400" alt="Search Engine Service">
</p>

A high-performance Go microservice that aggregates, ranks, and serves content from multiple providers using a hybrid
ranking algorithm combining full-text search with engagement metrics.

## 🚀 Quick Start

```bash
# Clone the repository
git clone git@github.com:yourusername/search-engine-service.git
cd search-engine-service

# Start the full stack (API, PostgreSQL, Redis, Mock Providers)
make docker-up

# Verify health
curl http://localhost:8080/livez
```

📖 For detailed setup instructions, see [Development Guide](docs/DEVELOPMENT.md)

## 📚 Documentation

Comprehensive documentation is available in the [`docs/`](docs/) directory:

| Document                                          | Description                                                     |
|---------------------------------------------------|-----------------------------------------------------------------|
| [**Architecture & Design**](docs/ARCHITECTURE.md) | Hybrid ranking algorithm, circuit breakers, distributed locking |
| [**API Reference**](docs/API.md)                  | Complete endpoint documentation with examples                   |
| [**Configuration**](docs/CONFIGURATION.md)        | Environment variables and Viper settings                        |
| [**Development**](docs/DEVELOPMENT.md)            | Setup, testing, and contribution workflow                       |
| [**Deployment**](docs/DEPLOYMENT.md)              | Docker, Kubernetes manifests, and observability                 |

## 🏗️ Architecture Overview

**Tech Stack:** Go 1.25 • Fiber v2.52.10 • PostgreSQL • Redis • GORM v1.31.1

**Layered Architecture (Ports & Adapters):**

- **Transport Layer**: HTTP handlers, middleware, DTOs
- **Application Layer**: Search and Sync services (use cases)
- **Domain Layer**: Core entities, scoring logic, repository interfaces
- **Infrastructure Layer**: PostgreSQL, Redis, external provider adapters

**Key Features:**

- **Hybrid Ranking**: Balances `ts_rank` (Title: A, Tags: B) with logarithmic popularity
- **Distributed Locking**: Redis-based Redlock for safe background synchronization
- **Circuit Breakers**: Per-provider resilience with `sony/gobreaker`
- **Retry with Backoff**: Exponential backoff + jitter via Resty
- **Full-Text Search**: PostgreSQL GIN indexes with weighted `tsvector`
- **Observability**: Structured logs (Zap) and health checks (`/livez`, `/readyz`)

For detailed architecture documentation, see [ARCHITECTURE.md](docs/ARCHITECTURE.md)

## 🔧 Configuration

Key environment variables:

| Variable            | Description              | Default     |
|---------------------|--------------------------|-------------|
| `APP_APP_PORT`      | Service port             | `8080`      |
| `APP_DATABASE_HOST` | PostgreSQL host          | `localhost` |
| `APP_DATABASE_PORT` | PostgreSQL port          | `5432`      |
| `APP_REDIS_HOST`    | Redis host               | `localhost` |
| `APP_REDIS_PORT`    | Redis port               | `6379`      |
| `APP_CACHE_ENABLED` | Enable search caching    | `false`     |
| `APP_SYNC_INTERVAL` | Background sync interval | `5m`        |

📋 See [Configuration Guide](docs/CONFIGURATION.md) for complete environment variables
and [config/config.example.yaml](config/config.example.yaml) for YAML structure.

## 🧪 Development

```bash
# Run all tests
make test

# Run unit tests only (fast)
make test-unit

# Run integration tests (real DB/Redis)
make test-integration

# Generate coverage report
make coverage

# Run linter
make lint

# Start mock providers for local testing
make mock
```

📘 See [Development Guide](docs/DEVELOPMENT.md) for comprehensive development workflows and testing strategies.

## 📦 Project Structure

```
search-engine-service/
├── cmd/api/            # Application entry point, DI wiring
├── internal/
│   ├── app/            # Application services (Search, Sync)
│   ├── config/         # Configuration management (Viper)
│   ├── domain/         # Core entities, scoring, interfaces
│   ├── infra/          # Infrastructure adapters
│   │   ├── postgres/   # PostgreSQL repository, migrations
│   │   ├── redis/      # Redis cache implementation
│   │   └── provider/   # External provider clients
│   ├── job/            # Background workers (sync scheduler)
│   ├── logger/         # Structured logging setup (Zap)
│   ├── transport/      # HTTP handlers, middleware, DTOs
│   └── validator/      # Request validation
├── pkg/locker/         # Reusable distributed lock package
├── api/                # OpenAPI specifications
├── config/             # Configuration templates
├── mock/               # Mock provider servers
├── web/                # Dashboard assets (Vue.js)
└── docs/               # Comprehensive documentation
```

🗂️ See [Architecture](docs/ARCHITECTURE.md) for detailed structure documentation.

## 🔌 Key Endpoints

| Endpoint                       | Method | Purpose                        |
|--------------------------------|--------|--------------------------------|
| `/livez`                       | GET    | Kubernetes liveness probe      |
| `/readyz`                      | GET    | Kubernetes readiness probe     |
| `/dashboard`                   | GET    | Web dashboard (Vue.js)         |
| `/api/v1/contents`             | GET    | Search content with pagination |
| `/api/v1/contents/:id`         | GET    | Get single content by ID       |
| `/api/v1/admin/sync`           | POST   | Trigger sync for all providers |
| `/api/v1/admin/sync/:provider` | POST   | Sync specific provider         |
| `/api/v1/admin/providers`      | GET    | List provider status           |

📖 See [API Reference](docs/API.md) for complete endpoint documentation.

## 🔒 Distributed System Patterns

### Circuit Breakers

Per-provider circuit breakers (`sony/gobreaker`) prevent cascading failures:

- **Closed**: Normal operation
- **Open**: Fast-failing when provider is down
- **Half-Open**: Testing recovery before full restoration

### Distributed Locking

Redis-based distributed locks (`redsync`) ensure only one replica runs background sync:

- **Algorithm**: Redlock for safety across multiple Redis nodes
- **Lock Keys**: `sync:{provider_name}`
- **TTL**: Configured via `sync.timeout`

### Retry with Backoff

Resty HTTP client with exponential backoff + jitter:

- **Max Attempts**: 3 (configurable)
- **Wait Time**: Starts at 1s, exponentially increases
- **Jitter**: Random delay to prevent thundering herd

## 🧮 Hybrid Search & Ranking Strategy

When searching, results are sorted by a hybrid rank that combines text relevance with content score:

$$
FinalRank = \text{ts_rank}(Vector, Query) \times \log_{10}(\text{CalculatedScore} + 10)
$$

**Content Score** (pre-calculated during sync):

- **Base**: Views/likes for videos, reading time for articles
- **Type Coefficient**: Video (1.5x) vs Article (1.0x)
- **Recency Bonus**: +5 for fresh content (<1 week), decays to 0
- **Interaction**: Engagement quality (likes/views or reactions/reading_time)

**Search Relevance**:

- PostgreSQL full-text search with weighted `tsvector` (Title: A, Tags: B)
- Logarithmic normalization prevents viral content from dominating relevant results
- The `+ 10` smoothing handles cold-start for new content

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for the complete scoring formula.

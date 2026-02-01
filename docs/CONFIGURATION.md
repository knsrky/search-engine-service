# Configuration Guide

The Search Engine Service uses **Viper** for configuration, supporting YAML files and Environment Variable overrides.

## 🔑 Priority

1. **Environment Variables** (Highest)
2. **Config File** (`config/config.yaml`)
3. **Defaults**

## 🌍 Environment Variables

All variables use the `APP_` prefix. The configuration structure is flattened for environment variables.

### Server Configuration

| Variable        | Default                 | Description                                   |
|-----------------|-------------------------|-----------------------------------------------|
| `APP_APP_NAME`  | `search-engine-service` | Application name                              |
| `APP_APP_ENV`   | `development`           | Environment: development, staging, production |
| `APP_APP_PORT`  | `8080`                  | HTTP service port                             |
| `APP_APP_DEBUG` | `true`                  | Enable debug mode                             |

### Database Configuration

| Variable                      | Default         | Description                                        |
|-------------------------------|-----------------|----------------------------------------------------|
| `APP_DATABASE_HOST`           | `localhost`     | PostgreSQL host                                    |
| `APP_DATABASE_PORT`           | `5432`          | PostgreSQL port                                    |
| `APP_DATABASE_NAME`           | `search_engine` | Database name                                      |
| `APP_DATABASE_USER`           | `app`           | Database user                                      |
| `APP_DATABASE_PASSWORD`       | `secret`        | Database password                                  |
| `APP_DATABASE_SSL_MODE`       | `disable`       | SSL mode: disable, require, verify-ca, verify-full |
| `APP_DATABASE_MAX_OPEN_CONNS` | `25`            | Maximum open connections                           |
| `APP_DATABASE_MAX_IDLE_CONNS` | `5`             | Maximum idle connections                           |
| `APP_DATABASE_MAX_LIFETIME`   | `5m`            | Connection max lifetime                            |

### Redis Configuration

| Variable             | Default     | Description           |
|----------------------|-------------|-----------------------|
| `APP_REDIS_HOST`     | `localhost` | Redis host            |
| `APP_REDIS_PORT`     | `6379`      | Redis port            |
| `APP_REDIS_PASSWORD` | `""`        | Redis password        |
| `APP_REDIS_DB`       | `0`         | Redis database number |

### Cache Configuration

| Variable               | Default         | Description                   |
|------------------------|-----------------|-------------------------------|
| `APP_CACHE_ENABLED`    | `false`         | Enable search result caching  |
| `APP_CACHE_SEARCH_TTL` | `15m`           | TTL for cached search results |
| `APP_CACHE_KEY_PREFIX` | `search-engine` | Cache key prefix              |

### Provider Configuration

The endpoint path is hardcoded in the provider client code (not configurable via env vars).

#### Provider A

| Variable                                       | Default                 | Description                     |
|------------------------------------------------|-------------------------|---------------------------------|
| `APP_PROVIDER_A_BASE_URL`                      | `http://localhost:8081` | Provider A base URL             |
| `APP_PROVIDER_A_TIMEOUT`                       | `10s`                   | HTTP request timeout            |
| `APP_PROVIDER_A_RETRY_MAX_ATTEMPTS`            | `3`                     | Maximum retry attempts          |
| `APP_PROVIDER_A_RETRY_WAIT_TIME`               | `1s`                    | Initial retry wait time         |
| `APP_PROVIDER_A_RETRY_MAX_WAIT_TIME`           | `5s`                    | Maximum retry wait time         |
| `APP_PROVIDER_A_CIRCUIT_BREAKER_MAX_REQUESTS`  | `3`                     | Max requests in half-open state |
| `APP_PROVIDER_A_CIRCUIT_BREAKER_INTERVAL`      | `60s`                   | CB statistical interval         |
| `APP_PROVIDER_A_CIRCUIT_BREAKER_TIMEOUT`       | `30s`                   | CB open state timeout           |
| `APP_PROVIDER_A_CIRCUIT_BREAKER_FAILURE_RATIO` | `0.5`                   | Failure ratio to trip CB        |

### Provider B Configuration is identical to Provider A

### Sync Configuration

| Variable              | Default | Description                |
|-----------------------|---------|----------------------------|
| `APP_SYNC_INTERVAL`   | `5m`    | Background sync interval   |
| `APP_SYNC_ON_STARTUP` | `true`  | Run sync on startup        |
| `APP_SYNC_TIMEOUT`    | `30s`   | Sync operation timeout     |
| `APP_SYNC_BATCH_SIZE` | `100`   | Batch size for bulk upsert |

### Logger Configuration

| Variable            | Default   | Description                         |
|---------------------|-----------|-------------------------------------|
| `APP_LOGGER_LEVEL`  | `info`    | Log level: debug, info, warn, error |
| `APP_LOGGER_FORMAT` | `console` | Log format: console, json           |
| `APP_LOGGER_OUTPUT` | `stdout`  | Log output destination              |

### Sentry Configuration

| Variable                 | Default       | Description                   |
|--------------------------|---------------|-------------------------------|
| `APP_SENTRY_ENABLED`     | `false`       | Enable Sentry error tracking  |
| `APP_SENTRY_DSN`         | `""`          | Sentry DSN                    |
| `APP_SENTRY_ENVIRONMENT` | `development` | Sentry environment            |
| `APP_SENTRY_SAMPLE_RATE` | `1.0`         | Error sampling rate (0.0-1.0) |

## ⚙️ Config File Example

(`config/config.yaml`)

```yaml
app:
  name: search-engine-service
  env: development
  port: 8080
  debug: true

database:
  host: localhost
  port: 5432
  name: search_engine
  user: app
  password: secret
  ssl_mode: disable
  max_open_conns: 25
  max_idle_conns: 5
  max_lifetime: 5m

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

cache:
  enabled: false
  search_ttl: 15m
  key_prefix: search-engine

# Provider Settings
provider:
  a:
    base_url: http://localhost:8081
    timeout: 10s
    retry:
      max_attempts: 3
      wait_time: 1s
      max_wait_time: 5s
    circuit_breaker:
      max_requests: 3
      interval: 60s
      timeout: 30s
      failure_ratio: 0.5
  b:
    base_url: http://localhost:8082
    timeout: 10s
    retry:
      max_attempts: 3
      wait_time: 1s
      max_wait_time: 5s
    circuit_breaker:
      max_requests: 3
      interval: 60s
      timeout: 30s
      failure_ratio: 0.5

sync:
  interval: 5m
  on_startup: true
  timeout: 30s
  batch_size: 100

logger:
  level: info
  format: console
  output: stdout

sentry:
  enabled: false
  dsn: ""
  environment: development
  sample_rate: 1.0
```

## 🔁 Circuit Breaker Settings

The circuit breaker uses the [sony/gobreaker](https://github.com/sony/gobreaker) implementation with three states:

- **Closed**: Normal operation, requests pass through
- **Open**: Circuit tripped, requests fail immediately
- **Half-Open**: Testing if service has recovered

| Setting         | Description                                 | Default   |
|-----------------|---------------------------------------------|-----------|
| `max_requests`  | Max requests allowed in half-open state     | 3         |
| `interval`      | Time window for statistical calculations    | 60s       |
| `timeout`       | Duration to wait before attempting recovery | 30s       |
| `failure_ratio` | Failure ratio threshold to trip the breaker | 0.5 (50%) |

## 🔄 Retry Mechanism

The retry mechanism uses [Resty](https://github.com/go-resty/resty) with exponential backoff:

| Setting         | Description                          | Default |
|-----------------|--------------------------------------|---------|
| `max_attempts`  | Maximum number of retry attempts     | 3       |
| `wait_time`     | Initial wait time before first retry | 1s      |
| `max_wait_time` | Maximum wait time between retries    | 5s      |

The retry uses exponential backoff with jitter to prevent thundering herd problems.

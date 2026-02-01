# Configuration Guide

The Search Engine Service uses **Viper** for configuration, supporting YAML files and Environment Variable overrides.

## 🔑 Priority
1. **Environment Variables** (Highest)
2. **Config File** (`config/config.yaml`)
3. **Defaults**

## 🌍 Environment Variables

All variables use the `APP_` prefix.

| Category | Variable | Default | Description |
|----------|----------|---------|-------------|
| **Server** | `APP_SERVER_PORT` | `8080` | HTTP service port |
| **Log** | `APP_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| **DB** | `APP_DATABASE_HOST` | `localhost` | PostgreSQL Host |
| **DB** | `APP_DATABASE_PORT` | `5432` | PostgreSQL Port |
| **Redis** | `APP_REDIS_HOST` | `localhost` | Redis Host |
| **Redis** | `APP_REDIS_PORT` | `6379` | Redis Port |
| **Cache** | `APP_CACHE_ENABLED` | `false` | Enable/Disable search caching |
| **Cache** | `APP_CACHE_SEARCH_TTL`| `15m` | TTL for search results |
| **Sync** | `APP_SYNC_INTERVAL` | `15m` | Background sync frequency |
| **Provider**| `APP_PROVIDERS_PROVIDER_A_BASE_URL` | `http://...` | Provider A URL |
| **Provider**| `APP_PROVIDERS_PROVIDER_B_BASE_URL` | `http://...` | Provider B URL |



## ⚙️ Config File Example
(`config/config.yaml`)

```yaml
server:
  port: 9092

database:
  host: "postgres"
  name: "search_engine"

redis:
  host: "redis"

# Provider Settings
providers:
  provider_a:
    base_url: "http://provider-a:8081"
    timeout: "10s"
```

## 🔒 Secrets
**Never commit secrets.** In production (Kubernetes), map Secrets to environment variables:
- `APP_DATABASE_PASSWORD` -> `Secret("db-pass")`
- `APP_REDIS_PASSWORD` -> `Secret("redis-pass")`

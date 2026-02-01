# Deployment Guide

## 🐳 Docker

### Build Image
```bash
docker build -t search-engine-service:latest .
```

### Run Container
```bash
docker run -p 8080:8080 \
  -e APP_DATABASE_HOST=host.docker.internal \
  -e APP_REDIS_HOST=host.docker.internal \
  search-engine-service:latest
```

## ☸️ Kubernetes

The service is stateless and ready for horizontal scaling.

| Resource | Purpose | Settings |
|----------|---------|----------|
| **Deployment** | Manages Pods | Replicas: 3 (Auto-scale 2-10) |
| **Service** | Load Balancer | Port 80 -> 8080 |
| **ConfigMap** | Non-sensitive | Log Level, Sync Interval |
| **Secret** | Sensitive | DB Password, Redis Password |

### Probes
- **Liveness** (`/livez`): Checks if process is running. Restart if fails.
- **Readiness** (`/readyz`): Checks DB/Redis connection. traffic off if fails.

## 📊 Observability

- **Metrics**: Prometheus-compatible metrics exposed at `/metrics` (if enabled in router).
- **Logs**: Structured JSON logging via **Zap**. Ideal for ELK/Loki.

### Resource Limits (Recommended)
- **CPU**: Request `100m`, Limit `500m`
- **Memory**: Request `128Mi`, Limit `512Mi`

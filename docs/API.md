# API Reference

The Search Engine Service exposes a RESTful API for searching content and managing the system.

## Base URL

Default: `http://localhost:8080`

## Endpoints

### 1. Health Checks

Kubernetes probes for container health monitoring.

| Endpoint  | Method | Purpose                                        |
|-----------|--------|------------------------------------------------|
| `/livez`  | GET    | Liveness probe - checks if process is running  |
| `/readyz` | GET    | Readiness probe - checks DB/Redis connectivity |

**Example Request**:

```bash
curl http://localhost:8080/livez
# Response: OK
```

---

### 2. Dashboard

Web interface for content search and management.

| Endpoint     | Method | Purpose                             |
|--------------|--------|-------------------------------------|
| `/`          | GET    | Redirects to `/dashboard`           |
| `/dashboard` | GET    | HTML dashboard with Vue.js frontend |

**Example Request**:

```bash
curl http://localhost:8080/dashboard
# Returns HTML page
```

---

### 3. Search Contents

Search for content with support for full-text query, filtering, sorting, and pagination.

**Endpoint**: `GET /api/v1/contents`

**Query Parameters**:

| Parameter    | Type    | Default      | Constraints                              | Description             |
|--------------|---------|--------------|------------------------------------------|-------------------------|
| `q`          | string  | -            | max 200 chars                            | Search query            |
| `type`       | string  | -            | `video` \| `article`                     | Filter by content type  |
| `sort_by`    | string  | `relevance`* | `relevance` \| `score` \| `published_at` | Field to sort by        |
| `sort_order` | string  | `desc`       | `asc` \| `desc`                          | Sort direction          |
| `page`       | integer | `1`          | min 1                                    | Page number (1-indexed) |
| `page_size`  | integer | `5`          | min 1, max 100                           | Items per page          |

*When `q` is provided and `sort_by` is not specified, defaults to `relevance`. Otherwise defaults to `score`.

**Example Request**:

```bash
curl -G "http://localhost:8080/api/v1/contents" \
    --data-urlencode "q=distributed systems" \
    --data-urlencode "type=article" \
    --data-urlencode "sort_by=relevance" \
    --data-urlencode "sort_order=desc" \
    --data-urlencode "page=1" \
    --data-urlencode "page_size=20"
```

**Example Response**:

```json
{
  "contents": [
    {
      "id": "809743ba-5825-4e56-ae11-7fc524eac3f3",
      "provider_id": "provider_b",
      "external_id": "a1",
      "title": "Clean Architecture in Go",
      "type": "article",
      "tags": [
        "programming",
        "architecture"
      ],
      "reading_time": 8,
      "reactions": 450,
      "comments": 25,
      "score": 298.25,
      "published_at": "2024-03-14T00:00:00Z",
      "created_at": "2026-01-31T20:40:38Z",
      "updated_at": "2026-02-01T19:17:32Z"
    },
    {
      "id": "edd76794-557a-4b7f-bdce-b4866b5356e3",
      "provider_id": "provider_a",
      "external_id": "v3",
      "title": "Building RESTful APIs with Go",
      "type": "video",
      "tags": [
        "programming",
        "api",
        "rest"
      ],
      "views": 18500,
      "likes": 1500,
      "duration": "19:15",
      "score": 51.06,
      "published_at": "2024-03-13T09:15:00Z",
      "created_at": "2026-01-31T20:40:38Z",
      "updated_at": "2026-02-01T19:17:32Z"
    }
  ],
  "pagination": {
    "total": 8,
    "page": 1,
    "page_size": 20,
    "total_pages": 1
  }
}
```

---

### 4. Get Single Content

Retrieve details of a specific content item by its ID.

**Endpoint**: `GET /api/v1/contents/:id`

**Example Request**:

```bash
curl "http://localhost:8080/api/v1/contents/809743ba-5825-4e56-ae11-7fc524eac3f3"
```

**Example Response**:

```json
{
  "id": "809743ba-5825-4e56-ae11-7fc524eac3f3",
  "provider_id": "provider_b",
  "external_id": "a1",
  "title": "Clean Architecture in Go",
  "type": "article",
  "tags": [
    "programming",
    "architecture"
  ],
  "reading_time": 8,
  "reactions": 450,
  "comments": 25,
  "score": 298.25,
  "published_at": "2024-03-14T00:00:00Z",
  "created_at": "2026-01-31T20:40:38Z",
  "updated_at": "2026-02-01T19:17:32Z"
}
```

---

### 5. Admin: Trigger Sync All

Manually trigger synchronization for all providers.

**Endpoint**: `POST /api/v1/admin/sync`

**Example Request**:

```bash
curl -X POST "http://localhost:8080/api/v1/admin/sync"
```

**Example Response**:

```json
{
  "results": [
    {
      "provider": "provider_a",
      "count": 150,
      "duration": "1.2s"
    },
    {
      "provider": "provider_b",
      "count": 45,
      "duration": "0.8s"
    }
  ],
  "summary": {
    "total_synced": 195,
    "providers_ok": 2,
    "providers_fail": 0
  }
}
```

---

### 6. Admin: Sync Specific Provider

Trigger synchronization for a single provider.

**Endpoint**: `POST /api/v1/admin/sync/:provider`

**Path Parameters**:

- `provider`: Provider name (`provider_a` or `provider_b`)

**Example Request**:

```bash
curl -X POST "http://localhost:8080/api/v1/admin/sync/provider_a"
```

**Example Response**:

```json
{
  "provider": "provider_a",
  "count": 150,
  "duration": "1.2s"
}
```

---

### 7. Admin: List Providers

Retrieve providers list

**Endpoint**: `GET /api/v1/admin/providers`

**Example Request**:

```bash
curl "http://localhost:8080/api/v1/admin/providers"
```

**Example Response**:

```json
{
  "providers": [
    "provider_a",
    "provider_b"
  ]
}
```

---

## Error Handling

Errors are returned in a standard format:

```json
{
  "error": "human readable message",
  "code": "ERROR_CODE",
  "details": {}
}
```

**Common Error Codes**:

| Code                  | Description                   |
|-----------------------|-------------------------------|
| `VALIDATION_ERROR`    | Request validation failed     |
| `NOT_FOUND`           | Resource not found            |
| `INTERNAL_ERROR`      | Server-side error             |
| `SERVICE_UNAVAILABLE` | Provider circuit breaker open |

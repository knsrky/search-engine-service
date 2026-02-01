# API Reference

The Search Engine Service exposes a RESTful API for searching content and managing the system.

## Base URL

Default: `http://localhost:8080`

## Endpoints

### 1. Search Contents

Search for content with support for full-text query, filtering, sorting, and pagination.

**Endpoint**: `GET /api/v1/contents`

**Parameters**:

- `q` (optional): Search query (e.g., "golang tutorial"). Supports boolean operators.
- `type` (optional): Filter by type (`video` | `article`).
- `sort_by` (optional): Field to sort by (`relevance` | `score` | `published_at`). Default: `relevance` if `q` is
  specified,else `score`.
- `page` (optional): Page number. Default: `1`.
- `page_size` (optional): Items per page. Default: `5`.

**Example Request**:

```bash
curl -G "http://localhost:9092/api/v1/contents" \
    --data-urlencode "q=distributed systems" \
    --data-urlencode "type=article" \
    --data-urlencode "sort_by=relevance"
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
    "page_size": 5,
    "total_pages": 2
  }
}
```

---

### 2. Get Single Content

Retrieve details of a specific content item by its ID.

**Endpoint**: `GET /api/v1/contents/:id`

**Example Request**:

```bash
curl "http://localhost:9092/api/v1/contents/809743ba-5825-4e56-ae11-7fc524eac3f3"
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

### 3. Admin: Trigger Sync

Manually trigger the background synchronization process.

**Endpoint**: `POST /api/v1/admin/sync`

**Example Request**:

```bash
curl -X POST "http://localhost:9092/api/v1/admin/sync"
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

## Error Handling

Errors are returned in a standard format:

```json
{
  "error": "human readable message",
  "code": "ERROR_CODE",
  "details": {}
}
```

# API Versioning Strategy

## Overview

DeployPilot uses **URL-based API versioning**. All API endpoints are prefixed with `/api/v{version}/`, for example:

```
/api/v1/apps
/api/v1/servers
/api/v1/credentials
```

This document describes the versioning lifecycle, response headers, and migration guidelines for API consumers.

---

## Version Lifecycle

Each API version progresses through the following stages:

| Stage         | Description                                                                 |
|---------------|-----------------------------------------------------------------------------|
| **Experimental** | Newly introduced version. May have breaking changes without notice.        |
| **Stable**       | Fully supported. Backward-compatible changes only. Recommended for production. |
| **Deprecated**   | Still available but scheduled for removal. Consumers should migrate.        |
| **Sunset**       | No longer available. All requests return `410 Gone`.                        |

### Current Status

| Version | Status       | Notes                        |
|---------|-------------|------------------------------|
| v1      | **Stable**  | Current production version   |

---

## Response Headers

Every API response includes the following version-related headers:

| Header            | Example Value                | Description                                              |
|-------------------|------------------------------|----------------------------------------------------------|
| `X-API-Version`   | `v1`                         | The API version that served the request.                 |
| `Deprecation`     | `true`                       | Present when the requested version is deprecated.        |
| `Sunset`          | `2027-01-01`                 | Date after which the deprecated version will be removed.  |
| `Accept-Version`  | `v1, v2`                     | Present when an unsupported version is requested. Lists valid versions. |
| `Link`            | `</api/v2>; rel="successor-version"` | Points to the successor version when deprecating. |

### Header Examples

**Stable version request:**
```http
GET /api/v1/apps HTTP/1.1

HTTP/1.1 200 OK
X-API-Version: v1
```

**Deprecated version request (future):**
```http
GET /api/v1/apps HTTP/1.1

HTTP/1.1 200 OK
X-API-Version: v1
Deprecation: true
Sunset: 2027-01-01
Link: </api/v2>; rel="successor-version"
```

**Unsupported version request:**
```http
GET /api/v99/apps HTTP/1.1

HTTP/1.1 404 Not Found
X-API-Version: v99
Accept-Version: v1
```

---

## Version Discovery Endpoint

```
GET /api/v1/version
```

Returns the current API version information without authentication:

```json
{
  "status": "success",
  "data": {
    "version": "v1",
    "supported": ["v1"],
    "status": "stable",
    "documentation": "/api/v1/docs"
  }
}
```

---

## Backward Compatibility Guarantees

When a version is marked **Stable**, the following guarantees apply:

1. **No breaking changes**: Existing fields will not be removed or renamed.
2. **Additive changes only**: New fields may be added to response objects.
3. **New endpoints**: New endpoints may be added under the same version prefix.
4. **Deprecated fields**: Fields scheduled for removal will be documented with a migration path and remain functional for at least two minor releases.

---

## Migration Guide for API Consumers

When a new API version is released:

1. **Monitor headers**: Watch for `Deprecation` and `Sunset` headers in responses.
2. **Check the version endpoint**: Periodically query `GET /api/v1/version` for the latest status.
3. **Update your base URL**: Switch from `/api/v1/` to `/api/v2/` (or the latest stable version).
4. **Review changelog**: Consult the CHANGELOG.md for a detailed list of changes between versions.
5. **Test in staging**: Validate your integration against the new version before the sunset date.

### Example Migration

```bash
# Before (v1)
curl -H "Authorization: Bearer $TOKEN" https://your-deploypilot/api/v1/apps

# After (v2)
curl -H "Authorization: Bearer $TOKEN" https://your-deploypilot/api/v2/apps
```

---

## Configuration

API versioning can be configured via environment variables or the configuration file:

| Config Key                          | Environment Variable                    | Default   |
|-------------------------------------|-----------------------------------------|-----------|
| `api_version.current_version`       | `DEPLOYPILOT_API_VERSION_CURRENT_VERSION` | `v1`    |
| `api_version.supported_versions`    | `DEPLOYPILOT_API_VERSION_SUPPORTED_VERSIONS` | `["v1"]` |
| `api_version.deprecated_versions`   | `DEPLOYPILOT_API_VERSION_DEPRECATED_VERSIONS` | `{}`    |

Example in `config.yaml`:

```yaml
api_version:
  current_version: "v1"
  supported_versions:
    - "v1"
  deprecated_versions: {}
```

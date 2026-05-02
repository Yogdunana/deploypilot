# Phase 7.3: Outbound Webhook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an outbound webhook system that delivers events to external URLs with HMAC signatures, multi-platform format adapters, event filtering, retry with exponential backoff, and delivery logging.

**Architecture:** New `OutboundWebhookService` integrates with existing `EventRouter` as a forwarding target. When events match a webhook's filter rules, the service formats the payload (JSON/Slack/Discord/Teams), signs it with HMAC-SHA256, and delivers via HTTP POST with exponential backoff retry. All deliveries are logged to `webhook_deliveries` table.

**Tech Stack:** Go 1.23, Gin, GORM, crypto/hmac, Vue 3 + TypeScript + Tailwind CSS

**Branch:** `feat/phase-7.3-outbound-webhook`
**PR Title:** `[v1.7] Phase 7.3: Outbound Webhook with HMAC, format adapters, retry, and delivery log`

---

### Task 1: Create OutboundWebhook and WebhookDelivery models

**Files:**
- Create: `internal/model/outbound_webhook.go`

- [ ] **Step 1: Create the model file**

```go
package model

import "time"

// OutboundWebhook represents an outbound webhook configuration.
type OutboundWebhook struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	TenantID       string    `gorm:"index" json:"tenant_id,omitempty"`
	Name           string    `gorm:"not null;size:200" json:"name"`
	URL            string    `gorm:"not null;size:500" json:"url"`
	Secret         string    `gorm:"size:500" json:"-"` // HMAC secret, never exposed
	Format         string    `gorm:"size:20;default:json" json:"format"` // json, slack, discord, teams
	EventTypes     string    `gorm:"type:text" json:"event_types"`       // JSON array
	SeverityFilter string    `gorm:"type:text" json:"severity_filter"`   // JSON array, empty = all
	AppFilter      string    `gorm:"type:text" json:"app_filter"`        // JSON array, empty = all
	ServerFilter   string    `gorm:"type:text" json:"server_filter"`     // JSON array, empty = all
	Enabled        bool      `gorm:"default:true" json:"enabled"`
	MaxRetries     int       `gorm:"default:5" json:"max_retries"`
	Timeout        int       `gorm:"default:10" json:"timeout"`         // seconds
	Description    string    `json:"description,omitempty"`
	LastDeliveryAt *time.Time `json:"last_delivery_at,omitempty"`
	LastStatus     string    `gorm:"size:20" json:"last_status"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (OutboundWebhook) TableName() string { return "outbound_webhooks" }

// WebhookDelivery records each delivery attempt.
type WebhookDelivery struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	WebhookID    string    `gorm:"index" json:"webhook_id"`
	TenantID     string    `gorm:"index" json:"tenant_id,omitempty"`
	EventID      string    `json:"event_id"`
	EventType    string    `gorm:"size:20;index" json:"event_type"`
	StatusCode   int       `json:"status_code"`
	LatencyMs    int       `json:"latency_ms"`
	Attempt      int       `json:"attempt"`
	Success      bool      `json:"success"`
	ErrorResponse string   `gorm:"type:text" json:"error_response,omitempty"`
	RequestBody  string    `gorm:"type:text" json:"request_body,omitempty"`
	CreatedAt    time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (WebhookDelivery) TableName() string { return "webhook_deliveries" }
```

- [ ] **Step 2: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./internal/model/`

- [ ] **Step 3: Commit**

```bash
git add internal/model/outbound_webhook.go
git commit -m "feat(model): add OutboundWebhook and WebhookDelivery models"
```

---

### Task 2: Create HMAC signature utility and format adapters

**Files:**
- Create: `internal/service/webhook_formatter.go`

- [ ] **Step 1: Create formatter file with HMAC + 4 format adapters**

```go
package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// SignWebhook computes HMAC-SHA256 signature: sha256=<hex(timestamp.body)>
func SignWebhook(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d.%s", timestamp, body)))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// WebhookPayload is the standard JSON payload structure.
type WebhookPayload struct {
	EventID    string      `json:"event_id"`
	EventType  string      `json:"event_type"`
	Topic      string      `json:"topic,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
	Payload    interface{} `json:"payload"`
}

// FormatAdapter converts a WebhookPayload to platform-specific format.
type FormatAdapter interface {
	Format(payload WebhookPayload) ([]byte, string) // returns (body, contentType)
	Name() string
}

// JSONFormatter outputs standard JSON.
type JSONFormatter struct{}

func (JSONFormatter) Name() string { return "json" }

func (JSONFormatter) Format(payload WebhookPayload) ([]byte, string) {
	body, _ := json.Marshal(payload)
	return body, "application/json"
}

// SlackFormatter outputs Slack Incoming Webhook format.
type SlackFormatter struct{}

func (SlackFormatter) Name() string { return "slack" }

func (SlackFormatter) Format(payload WebhookPayload) ([]byte, string) {
	// Extract fields from payload
	pMap, _ := payloadToMap(payload.Payload)
	color := "#36a64f" // green
	status := "info"
	if s, ok := pMap["status"].(string); ok {
		switch s {
		case "failed", "error", "critical":
			color = "#e01e5a"
			status = s
		case "warning":
			color = "#ffaa00"
			status = s
		default:
			color = "#36a64f"
		}
	}

	msg := fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic)
	if m, ok := pMap["message"].(string); ok && m != "" {
		msg = m
	}

	attachment := map[string]interface{}{
		"color": color,
		"title": fmt.Sprintf("DeployPilot Event: %s", payload.EventType),
		"text":  msg,
		"fields": []map[string]interface{}{
			{"title": "Event", "value": payload.EventType, "short": true},
			{"title": "Time", "value": payload.Timestamp.Format(time.RFC3339), "short": true},
		},
		"ts": payload.Timestamp.Unix(),
	}

	// Add app/server fields if present
	if app, ok := pMap["app_name"].(string); ok && app != "" {
		attachment["fields"] = append(attachment["fields"].([]map[string]interface{}),
			map[string]interface{}{"title": "App", "value": app, "short": true})
	}
	if srv, ok := pMap["server_name"].(string); ok && srv != "" {
		attachment["fields"] = append(attachment["fields"].([]map[string]interface{}),
			map[string]interface{}{"title": "Server", "value": srv, "short": true})
	}

	body, _ := json.Marshal(map[string]interface{}{
		"attachments": []interface{}{attachment},
	})
	return body, "application/json"
}

// DiscordFormatter outputs Discord Webhook format with embeds.
type DiscordFormatter struct{}

func (DiscordFormatter) Name() string { return "discord" }

func (DiscordFormatter) Format(payload WebhookPayload) ([]byte, string) {
	pMap, _ := payloadToMap(payload.Payload)
	color := 3066993 // green
	if s, ok := pMap["status"].(string); ok {
		switch s {
		case "failed", "error", "critical":
			color = 15158332
		case "warning":
			color = 16776960
		}
	}

	msg := fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic)
	if m, ok := pMap["message"].(string); ok && m != "" {
		msg = m
	}

	embed := map[string]interface{}{
		"title":       fmt.Sprintf("DeployPilot: %s", payload.EventType),
		"description": msg,
		"color":       color,
		"timestamp":   payload.Timestamp.Format(time.RFC3339),
		"fields": []map[string]interface{}{
			{"name": "Event ID", "value": payload.EventID, "inline": true},
		},
	}

	if app, ok := pMap["app_name"].(string); ok && app != "" {
		embed["fields"] = append(embed["fields"].([]map[string]interface{}),
			map[string]interface{}{"name": "App", "value": app, "inline": true})
	}
	if srv, ok := pMap["server_name"].(string); ok && srv != "" {
		embed["fields"] = append(embed["fields"].([]map[string]interface{}),
			map[string]interface{}{"name": "Server", "value": srv, "inline": true})
	}

	body, _ := json.Marshal(map[string]interface{}{"embeds": []interface{}{embed}})
	return body, "application/json"
}

// TeamsFormatter outputs Microsoft Adaptive Card format.
type TeamsFormatter struct{}

func (TeamsFormatter) Name() string { return "teams" }

func (TeamsFormatter) Format(payload WebhookPayload) ([]byte, string) {
	pMap, _ := payloadToMap(payload.Payload)

	msg := fmt.Sprintf("[%s] %s", payload.EventType, payload.Topic)
	if m, ok := pMap["message"].(string); ok && m != "" {
		msg = m
	}

	facts := []map[string]string{
		{"title": "Event", "value": payload.EventType},
		{"title": "Time", "value": payload.Timestamp.Format(time.RFC3339)},
		{"title": "Event ID", "value": payload.EventID},
	}
	if app, ok := pMap["app_name"].(string); ok && app != "" {
		facts = append(facts, map[string]string{"title": "App", "value": app})
	}

	body, _ := json.Marshal(map[string]interface{}{
		"type": "message",
		"attachments": []map[string]interface{}{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]interface{}{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.4",
					"body": []interface{}{
						map[string]interface{}{"type": "TextBlock", "text": msg, "weight": "bolder", "size": "medium"},
						map[string]interface{}{"type": "FactSet", "facts": facts},
					},
				},
			},
		},
	})
	return body, "application/json"
}

// GetFormatter returns the appropriate FormatAdapter for the given format string.
func GetFormatter(format string) FormatAdapter {
	switch format {
	case "slack":
		return SlackFormatter{}
	case "discord":
		return DiscordFormatter{}
	case "teams":
		return TeamsFormatter{}
	default:
		return JSONFormatter{}
	}
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./internal/service/`

- [ ] **Step 3: Commit**

```bash
git add internal/service/webhook_formatter.go
git commit -m "feat(webhook): add HMAC signature and format adapters (JSON/Slack/Discord/Teams)"
```

---

### Task 3: Create OutboundWebhookService with CRUD + delivery + retry

**Files:**
- Create: `internal/service/outbound_webhook_service.go`

- [ ] **Step 1: Create the service file**

This file should contain:
- `OutboundWebhookService` struct with `db *gorm.DB` and `bus TypedEventBus`
- `NewOutboundWebhookService(db, bus)` constructor
- CRUD methods: `Create`, `GetByID`, `List`, `Update`, `Delete`
- `Deliver(webhook, event)` method that:
  1. Gets the format adapter via `GetFormatter(webhook.Format)`
  2. Builds `WebhookPayload` from `BusEvent`
  3. Formats payload via adapter
  4. Signs with HMAC via `SignWebhook(webhook.Secret, timestamp, body)`
  5. Sends HTTP POST with `X-Webhook-Signature` and `X-Webhook-Timestamp` headers
  6. Records delivery to `webhook_deliveries` table
  7. On failure, schedules retry via goroutine with exponential backoff
- `RetryDelivery(deliveryID)` method for retry logic
- `TestDelivery(webhookID)` method for manual test
- `CleanupOldDeliveries()` method (delete records older than 7 days)
- `Start()` method that subscribes to all event types via TypedEventBus and dispatches to matching webhooks

Key implementation details:
- Use `github.com/google/uuid` for IDs (already in go.mod)
- Retry delay: `min(2^attempt * time.Second, 30*time.Second)`
- HTTP client timeout from webhook config
- Event filtering: check `webhook.EventTypes` JSON array contains `event.Type`, check `webhook.SeverityFilter` against payload severity, check `webhook.AppFilter`/`ServerFilter` against payload fields

- [ ] **Step 2: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./internal/service/`

- [ ] **Step 3: Commit**

```bash
git add internal/service/outbound_webhook_service.go
git commit -m "feat(webhook): add OutboundWebhookService with delivery, retry, and event filtering"
```

---

### Task 4: Create API endpoints for outbound webhooks

**Files:**
- Create: `internal/api/outbound_webhook_api.go`
- Modify: `internal/api/router.go`

- [ ] **Step 1: Create the API handler file**

Follow the existing pattern (see `globalMonitorAPI` pattern in router.go):

```go
package api

// Global outbound webhook API instance
var globalWebhookAPI *OutboundWebhookAPI

// GetGlobalWebhookAPI returns the global instance.
func GetGlobalWebhookAPI() *OutboundWebhookAPI { return globalWebhookAPI }
```

Handlers to implement:
- `CreateWebhook(db)` — POST /webhooks
- `ListWebhooks(db)` — GET /webhooks (paginated)
- `GetWebhook(db)` — GET /webhooks/:id
- `UpdateWebhook(db)` — PUT /webhooks/:id
- `DeleteWebhook(db)` — DELETE /webhooks/:id
- `TestWebhook(svc)` — POST /webhooks/:id/test
- `ListDeliveries(db)` — GET /webhooks/:id/deliveries (paginated)
- `GetDelivery(db)` — GET /webhooks/:id/deliveries/:did

Each handler should:
- Extract `db` from gin context via `c.MustGet("db").(*gorm.DB)`
- Parse request body / path params
- Call the corresponding service method
- Return JSON response

- [ ] **Step 2: Register routes in router.go**

In `RegisterRoutes()`, after creating `globalMonitorAPI`:
```go
// Outbound Webhook API
globalWebhookAPI = NewOutboundWebhookAPI(db)
```

In the `protected` group, add:
```go
// Outbound Webhooks
whGroup := protected.Group("/webhooks")
{
    whGroup.GET("", globalWebhookAPI.ListWebhooks)
    whGroup.POST("", globalWebhookAPI.CreateWebhook)
    whGroup.GET("/:id", globalWebhookAPI.GetWebhook)
    whGroup.PUT("/:id", globalWebhookAPI.UpdateWebhook)
    whGroup.DELETE("/:id", globalWebhookAPI.DeleteWebhook)
    whGroup.POST("/:id/test", globalWebhookAPI.TestWebhook)
    whGroup.GET("/:id/deliveries", globalWebhookAPI.ListDeliveries)
    whGroup.GET("/:id/deliveries/:did", globalWebhookAPI.GetDelivery)
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./...`

- [ ] **Step 4: Commit**

```bash
git add internal/api/outbound_webhook_api.go internal/api/router.go
git commit -m "feat(api): add outbound webhook CRUD and delivery API endpoints"
```

---

### Task 5: Integrate OutboundWebhookService with EventRouter and main.go

**Files:**
- Modify: `internal/service/event_router.go`
- Modify: `cmd/api-server/main.go`

- [ ] **Step 1: Add webhook service to EventRouter**

In `event_router.go`, add `webhookSvc *OutboundWebhookService` field to `EventRouter` struct. Update `NewEventRouter` to accept it as parameter. In `forwardToChannels()`, after sending to notification channels, also check if any outbound webhooks match the event and deliver.

Alternative simpler approach: Don't modify EventRouter at all. Instead, have `OutboundWebhookService.Start()` subscribe to TypedEventBus independently (same pattern as EventRouter.listenType). This avoids coupling.

**Recommended: Independent subscription approach.** The `OutboundWebhookService.Start()` method subscribes to all event types and handles filtering/delivery internally. No changes to EventRouter needed.

- [ ] **Step 2: Initialize OutboundWebhookService in main.go**

In `cmd/api-server/main.go`, after creating the EventRouter:
```go
webhookSvc := service.NewOutboundWebhookService(db, typedBus)
webhookSvc.Start()
```

Also start the delivery cleanup goroutine:
```go
go webhookSvc.StartCleanupLoop(context.Background())
```

- [ ] **Step 3: Verify compilation**

Run: `cd /data/user/work/deploypilot-dev && go build ./cmd/api-server/`

- [ ] **Step 4: Commit**

```bash
git add cmd/api-server/main.go
git commit -m "feat(webhook): initialize OutboundWebhookService in main.go startup"
```

---

### Task 6: Create frontend API client

**Files:**
- Create: `web/src/api/modules/outbound_webhook.ts`

- [ ] **Step 1: Create the API module**

```typescript
import api from '@/api'
import type { ApiResponse } from '@/types/api'

export interface OutboundWebhook {
  id: string
  name: string
  url: string
  format: 'json' | 'slack' | 'discord' | 'teams'
  event_types: string[]
  severity_filter: string[]
  app_filter: string[]
  server_filter: string[]
  enabled: boolean
  max_retries: number
  timeout: number
  description: string
  last_delivery_at: string | null
  last_status: string
  created_at: string
  updated_at: string
}

export interface WebhookDelivery {
  id: string
  webhook_id: string
  event_id: string
  event_type: string
  status_code: number
  latency_ms: number
  attempt: number
  success: boolean
  error_response: string
  request_body: string
  created_at: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}

export function listWebhooks(page = 1, pageSize = 20) {
  return api.get<ApiResponse<PaginatedResponse<OutboundWebhook>>>('/api/v1/webhooks', {
    params: { page, page_size: pageSize },
  })
}

export function getWebhook(id: string) {
  return api.get<ApiResponse<OutboundWebhook>>(`/api/v1/webhooks/${id}`)
}

export function createWebhook(data: Partial<OutboundWebhook>) {
  return api.post<ApiResponse<OutboundWebhook>>('/api/v1/webhooks', data)
}

export function updateWebhook(id: string, data: Partial<OutboundWebhook>) {
  return api.put<ApiResponse<OutboundWebhook>>(`/api/v1/webhooks/${id}`, data)
}

export function deleteWebhook(id: string) {
  return api.delete(`/api/v1/webhooks/${id}`)
}

export function testWebhook(id: string) {
  return api.post<ApiResponse<WebhookDelivery>>(`/api/v1/webhooks/${id}/test`)
}

export function listDeliveries(webhookId: string, page = 1, pageSize = 20) {
  return api.get<ApiResponse<PaginatedResponse<WebhookDelivery>>>(`/api/v1/webhooks/${webhookId}/deliveries`, {
    params: { page, page_size: pageSize },
  })
}

export function getDelivery(webhookId: string, deliveryId: string) {
  return api.get<ApiResponse<WebhookDelivery>>(`/api/v1/webhooks/${webhookId}/deliveries/${deliveryId}`)
}
```

- [ ] **Step 2: Verify frontend builds**

Run: `cd /data/user/work/deploypilot-dev/web && npm run build`

- [ ] **Step 3: Commit**

```bash
git add web/src/api/modules/outbound_webhook.ts
git commit -m "feat(frontend): add outbound webhook API client"
```

---

### Task 7: Create WebhookList.vue page

**Files:**
- Create: `web/src/views/WebhookList.vue`

- [ ] **Step 1: Create the list page**

A page showing all outbound webhooks as cards with:
- Name, URL (truncated/masked), format badge, enabled/disabled toggle
- Last delivery status indicator (green dot = success, red dot = failure)
- Last delivery time (relative)
- Edit/Delete actions
- "Create Webhook" button in header
- Click card → navigate to `/settings/webhooks/:id/edit`
- Click "Deliveries" → navigate to `/settings/webhooks/:id/deliveries`

Follow the existing page patterns (e.g., `UptimeMonitors.vue`, `Plugins.vue`). Use Tailwind CSS classes. Import `PageHeader` from `@/components/common/PageHeader.vue`.

- [ ] **Step 2: Verify frontend builds**

Run: `cd /data/user/work/deploypilot-dev/web && npm run build`

- [ ] **Step 3: Commit**

```bash
git add web/src/views/WebhookList.vue
git commit -m "feat(frontend): add outbound webhook list page"
```

---

### Task 8: Create WebhookForm.vue (create/edit)

**Files:**
- Create: `web/src/views/WebhookForm.vue`

- [ ] **Step 1: Create the form page**

A form page for creating/editing outbound webhooks with:
- Name input
- URL input
- Secret input (password type, show/hide toggle)
- Format dropdown (JSON / Slack / Discord / Teams)
- Event types multi-select (deploy, alert, notify, system, user, server, security, audit, backup)
- Severity filter multi-select (critical, warning, info) — optional
- App filter text input (comma-separated) — optional
- Server filter text input (comma-separated) — optional
- Max retries number input (default 5)
- Timeout number input (default 10)
- Description textarea
- Save / Test Delivery buttons

If route has `:id` param, load existing webhook data. Otherwise, create new.

- [ ] **Step 2: Verify frontend builds**

Run: `cd /data/user/work/deploypilot-dev/web && npm run build`

- [ ] **Step 3: Commit**

```bash
git add web/src/views/WebhookForm.vue
git commit -m "feat(frontend): add outbound webhook create/edit form"
```

---

### Task 9: Create WebhookDeliveries.vue page

**Files:**
- Create: `web/src/views/WebhookDeliveries.vue`

- [ ] **Step 1: Create the deliveries page**

A page showing delivery logs for a specific webhook:
- Back link to webhook list
- Table with columns: Time, Event Type, Status Code, Latency, Attempt, Status (success/fail badge)
- Click row → expand to show request body and error response
- Pagination

- [ ] **Step 2: Verify frontend builds**

Run: `cd /data/user/work/deploypilot-dev/web && npm run build`

- [ ] **Step 3: Commit**

```bash
git add web/src/views/WebhookDeliveries.vue
git commit -m "feat(frontend): add webhook delivery log page"
```

---

### Task 10: Register frontend routes and add sidebar navigation

**Files:**
- Modify: `web/src/router/index.ts`
- Modify: `web/src/layout/MainLayout.vue`

- [ ] **Step 1: Add routes in router/index.ts**

In the MainLayout children routes, add after the monitor routes:
```typescript
{ path: 'webhooks', name: 'WebhookList', component: () => import('@/views/WebhookList.vue') },
{ path: 'webhooks/new', name: 'WebhookCreate', component: () => import('@/views/WebhookForm.vue') },
{ path: 'webhooks/:id/edit', name: 'WebhookEdit', component: () => import('@/views/WebhookForm.vue'), props: true },
{ path: 'webhooks/:id/deliveries', name: 'WebhookDeliveries', component: () => import('@/views/WebhookDeliveries.vue'), props: true },
```

- [ ] **Step 2: Add sidebar link in MainLayout.vue**

In the `ops` nav group, after the monitor entries, add:
```typescript
{ path: '/webhooks', label: t('layout.webhooks'), icon: Webhook },
```

Add `Webhook` to the lucide-vue-next import at the top of the file.

- [ ] **Step 3: Verify frontend builds**

Run: `cd /data/user/work/deploypilot-dev/web && npm run build`

- [ ] **Step 4: Commit**

```bash
git add web/src/router/index.ts web/src/layout/MainLayout.vue
git commit -m "feat(frontend): register webhook routes and add sidebar navigation"
```

---

### Task 11: Write unit tests

**Files:**
- Create: `internal/service/webhook_formatter_test.go`
- Create: `internal/service/outbound_webhook_service_test.go`

- [ ] **Step 1: Write formatter tests**

Test HMAC signature generation and verification:
- `TestSignWebhook` — sign a payload, verify signature matches expected format
- `TestSignWebhook_Verify` — sign and verify with same secret
- `TestJSONFormatter` — verify output is valid JSON with expected fields
- `TestSlackFormatter` — verify output has "attachments" array
- `TestDiscordFormatter` — verify output has "embeds" array
- `TestTeamsFormatter` — verify output has Adaptive Card structure
- `TestGetFormatter` — verify correct adapter returned for each format string

- [ ] **Step 2: Write service tests**

Test CRUD operations and delivery logic:
- `TestCreateWebhook` — create and verify DB record
- `TestListWebhooks` — list with pagination
- `TestEventFiltering` — verify events are filtered by type/severity/app/server

- [ ] **Step 3: Run tests**

Run: `cd /data/user/work/deploypilot-dev && go test ./internal/service/ -run TestSign -v && go test ./internal/service/ -run TestFormatter -v`

- [ ] **Step 4: Commit**

```bash
git add internal/service/webhook_formatter_test.go internal/service/outbound_webhook_service_test.go
git commit -m "test(webhook): add unit tests for HMAC, formatters, and service"
```

---

### Task 12: Push, create PR, and merge

- [ ] **Step 1: Push branch**

```bash
git push -u origin feat/phase-7.3-outbound-webhook
```

- [ ] **Step 2: Create PR**

Title: `[v1.7] Phase 7.3: Outbound Webhook with HMAC, format adapters, retry, and delivery log`
Body: `Closes #144`

- [ ] **Step 3: Monitor CI, fix any failures**

- [ ] **Step 4: Squash merge**

```bash
curl -X PUT -H "Authorization: token <PAT>" -H "Accept: application/vnd.github+json" \
  https://api.github.com/repos/Yogdunana/deploypilot/pulls/<PR>/merge \
  -d '{"merge_method":"squash"}'
```

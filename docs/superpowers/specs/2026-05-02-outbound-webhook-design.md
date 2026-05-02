# Phase 7.3: Outbound Webhook System

**日期**: 2026-05-02
**版本**: v1.7 Ecosystem Integration
**状态**: Approved

## 背景

DeployPilot 已有完善的事件总线（EventBus）和事件路由器（EventRouter），以及基础的 WebhookNotifier（HTTP POST JSON 通知）。但现有的 WebhookNotifier 仅用于通知渠道，缺少出站 Webhook 的关键能力：HMAC 签名验证、多平台格式适配、事件过滤、重试机制和投递日志。

## 设计目标

1. **HMAC 签名**：每个出站 Webhook 可配置签名密钥，接收方可验证请求来源
2. **多平台格式适配**：支持 JSON（通用）、Slack、Discord、Microsoft Teams 四种格式
3. **事件过滤**：按事件类型、严重级别、App/Server 维度过滤，避免无关事件投递
4. **重试机制**：指数退避重试（最多 5 次），失败事件记录到投递日志
5. **投递日志**：记录每次投递的状态码、延迟、错误信息，保留 7 天
6. **测试投递**：支持手动触发测试投递，方便配置验证

## 架构设计

```
事件源 (TypedEventBus)
  → EventRouter (已有，事件过滤 + 路由)
    → OutboundWebhookService (新增)
      → SignatureMiddleware (HMAC-SHA256)
      → FormatAdapter (JSON/Slack/Discord/Teams)
      → RetryQueue (指数退避, 最多 5 次)
      → DeliveryLog (投递日志, 7 天保留)
```

### 数据模型

**OutboundWebhook（数据库表 `outbound_webhooks`）**:

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | string (PK) | UUID |
| TenantID | string | 租户 ID |
| Name | string | 显示名称 |
| URL | string | 目标 URL |
| Secret | string | HMAC 签名密钥（加密存储） |
| Format | string | json / slack / discord / teams |
| EventTypes | string | JSON 数组，订阅的事件类型 |
| SeverityFilter | string | JSON 数组，严重级别过滤（空=全部） |
| AppFilter | string | JSON 数组，App 过滤（空=全部） |
| ServerFilter | string | JSON 数组，Server 过滤（空=全部） |
| Enabled | bool | 是否启用 |
| MaxRetries | int | 最大重试次数（默认 5） |
| Timeout | int | 请求超时秒数（默认 10） |
| Description | string | 描述 |
| LastDeliveryAt | time | 最后成功投递时间 |
| LastStatus | string | 最后投递状态 |
| CreatedAt / UpdatedAt | time | 时间戳 |

**WebhookDelivery（数据库表 `webhook_deliveries`）**:

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | string (PK) | UUID |
| WebhookID | string | 关联 Webhook |
| TenantID | string | 租户 ID |
| EventID | string | 事件 ID |
| EventType | string | 事件类型 |
| StatusCode | int | HTTP 响应码 |
| LatencyMs | int | 请求延迟（毫秒） |
| Attempt | int | 第几次尝试 |
| Success | bool | 是否成功 |
| ErrorResponse | string | 错误响应体 |
| RequestBody | string | 发送的请求体（调试用） |
| CreatedAt | time | 投递时间 |

### HMAC 签名

- Header: `X-Webhook-Signature: sha256=<hex_digest>`
- Header: `X-Webhook-Timestamp: <unix_seconds>`（防重放）
- 算法: `HMAC-SHA256(secret, timestamp + "." + body)`
- 验证方: 重新计算签名，检查时间戳在 5 分钟内

### 格式适配器

**JSONFormatter（默认）**: 标准 JSON payload
```json
{
  "event_id": "...",
  "event_type": "deploy",
  "timestamp": "2026-05-02T12:00:00Z",
  "payload": { ... }
}
```

**SlackFormatter**: Slack Incoming Webhook 格式
```json
{
  "text": "Deployment succeeded: my-app",
  "attachments": [{
    "color": "#36a64f",
    "fields": [
      { "title": "App", "value": "my-app", "short": true },
      { "title": "Server", "value": "prod-1", "short": true },
      { "title": "Status", "value": "success", "short": true }
    ]
  }]
}
```

**DiscordFormatter**: Discord Webhook 格式
```json
{
  "embeds": [{
    "title": "Deployment succeeded",
    "color": 3066993,
    "fields": [
      { "name": "App", "value": "my-app", "inline": true },
      { "name": "Server", "value": "prod-1", "inline": true }
    ],
    "timestamp": "2026-05-02T12:00:00Z"
  }]
}
```

**TeamsFormatter**: Microsoft Adaptive Card 格式
```json
{
  "type": "message",
  "attachments": [{
    "contentType": "application/vnd.microsoft.card.adaptive",
    "content": {
      "type": "AdaptiveCard",
      "body": [
        { "type": "TextBlock", "text": "Deployment succeeded", "weight": "bolder" },
        { "type": "FactSet", "facts": [
          { "title": "App", "value": "my-app" },
          { "title": "Server", "value": "prod-1" }
        ]}
      ]
    }
  }]
}
```

### 重试机制

- 策略：指数退避 `delay = min(2^attempt * 1s, 30s)`
- 最大重试次数：可配置（默认 5）
- 超过最大重试次数：标记为 failed，不再重试
- 重试在 goroutine 中异步执行，不阻塞事件处理

### 投递日志清理

- 定时任务：每小时清理 7 天前的投递日志
- 复用现有 `internal/service/cron_service.go` 的计划任务系统

## API 端点

所有端点在 `protected` 路由组内（需 JWT/API Key 认证）。

```
POST   /api/v1/webhooks                      创建出站 Webhook
GET    /api/v1/webhooks                      列表（分页）
GET    /api/v1/webhooks/:id                  详情
PUT    /api/v1/webhooks/:id                  更新
DELETE /api/v1/webhooks/:id                  删除
POST   /api/v1/webhooks/:id/test             测试投递（发送模拟事件）
GET    /api/v1/webhooks/:id/deliveries       投递日志（分页）
GET    /api/v1/webhooks/:id/deliveries/:did  投递详情
```

## 前端页面

### Webhook 列表页 (`/settings/webhooks`)
- 卡片列表，每个卡片显示：名称、URL（脱敏）、格式图标、状态指示灯、最后投递时间
- 创建按钮 → 弹窗/页面表单

### 创建/编辑表单
- 基本信息：名称、URL、描述
- 安全：Secret（密码输入框，可选）、格式选择（下拉）
- 事件过滤：事件类型多选、严重级别多选、App/Server 过滤
- 高级：超时、最大重试次数
- 测试投递按钮

### 投递日志页
- 表格：时间、事件类型、状态码、延迟、重试次数
- 点击展开：请求体、响应体
- 重试按钮

## 文件变更清单

### 后端新建
| 文件 | 说明 |
|------|------|
| `internal/model/outbound_webhook.go` | OutboundWebhook + WebhookDelivery 模型 |
| `internal/service/outbound_webhook_service.go` | CRUD + 投递逻辑 + HMAC + 格式化 |
| `internal/service/webhook_formatter.go` | 4 个格式适配器 |
| `internal/service/webhook_retry.go` | 重试队列 |
| `internal/api/outbound_webhook_api.go` | API 端点 |

### 后端修改
| 文件 | 变更 |
|------|------|
| `internal/api/router.go` | 注册 webhook 路由组 |
| `internal/service/event_router.go` | 添加 OutboundWebhook 作为转发目标 |

### 前端新建
| 文件 | 说明 |
|------|------|
| `web/src/api/modules/outbound_webhook.ts` | API 客户端 |
| `web/src/views/WebhookList.vue` | Webhook 列表页 |
| `web/src/views/WebhookForm.vue` | 创建/编辑表单 |
| `web/src/views/WebhookDeliveries.vue` | 投递日志页 |

### 前端修改
| 文件 | 变更 |
|------|------|
| `web/src/router/index.ts` | 添加 webhook 路由 |
| `web/src/layout/MainLayout.vue` | 侧边栏添加 Webhook 入口 |

## 测试策略

- 单元测试：HMAC 签名生成与验证、4 种格式化器输出、重试退避逻辑
- 集成测试：创建 Webhook → 触发事件 → 验证投递日志
- 手动测试：测试投递按钮 → 验证真实 Slack/Discord/Teams 收到消息

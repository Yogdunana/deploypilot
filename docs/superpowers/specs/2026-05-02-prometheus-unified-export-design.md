# Phase 6.3: Prometheus 指标导出统一

**日期**: 2026-05-02
**版本**: v1.6 Monitoring & Observability
**状态**: Approved

## 背景

当前项目中存在两套并行的 Prometheus 指标导出系统：

1. **原生系统** (`internal/metrics/`) — 在独立端口 9091 通过 `promhttp.Handler()` 提供，包含部署/容器/WebSocket/API 指标
2. **手工拼接系统** (`MonitorService.GetPrometheusMetrics()`) — 在 `/api/v1/metrics` 路径提供，手工拼接 Prometheus 文本格式，包含 uptime/heartbeat 指标，但缺少标准 `# TYPE` 和 `# HELP` 声明

此外，`MonitorScheduler` 每次检查完成后只通过 WebSocket 推送结果，没有更新任何 Prometheus Gauge，导致 Prometheus 无法采集到实时监控状态。

## 设计目标

1. **消除独立端口**：删除端口 9091 的独立 HTTP 服务器，所有指标通过主 API 端口导出
2. **统一指标格式**：将 uptime/heartbeat 指标从手工拼接迁移到原生 Prometheus 类型（Gauge）
3. **标准路径**：使用 `/metrics` 路径（Prometheus 生态惯例），删除 `/api/v1/metrics`
4. **认证与可控**：默认 JWT 认证保护，用户可通过设置决定是否公开访问
5. **实时更新**：MonitorScheduler 检查完成后自动更新 Prometheus Gauge

## 架构设计

### 指标端点

- **路径**: `/metrics`
- **Handler**: `promhttp.Handler()` (来自 `internal/metrics` 包)
- **认证**: 默认走 JWT 认证（注册在 `protected` 路由组内）；当 `monitor.metrics_public` 配置为 `true` 时，同时注册一个无需认证的公开版本
- **格式**: 标准 Prometheus exposition format（自动包含 `# TYPE`、`# HELP`、`# EOF`）

### 配置变更

`MonitorConfig` 结构体修改：

```go
type MonitorConfig struct {
    Enabled      bool `mapstructure:"enabled"`
    MetricsPublic bool `mapstructure:"metrics_public"`  // 新增：是否公开 /metrics 端点
    // MetricsPort int  `mapstructure:"metrics_port"`    // 删除：不再需要独立端口
}
```

- `monitor.metrics_public` 默认 `false`（需要 JWT 认证）
- 可通过 `config.yaml` 或环境变量 `DEPLOYPILOT_MONITOR_METRICS_PUBLIC` 配置
- 支持通过 MonitorSettings 前端页面运行时切换

### 新增 Prometheus 指标

在 `internal/metrics/metrics.go` 中新增 4 个 GaugeVec：

| 指标名 | 类型 | 标签 | 说明 |
|--------|------|------|------|
| `deploypilot_monitor_up` | GaugeVec | `name`, `type`, `target` | 监控目标状态 (0=down, 1=up) |
| `deploypilot_monitor_latency_ms` | GaugeVec | `name`, `type`, `target` | 最近一次检查延迟（毫秒） |
| `deploypilot_monitor_uptime_pct` | GaugeVec | `name`, `type`, `target` | SLA 可用率百分比 |
| `deploypilot_heartbeat_up` | GaugeVec | `name` | 心跳状态 (0=timeout, 1=alive) |

### MonitorScheduler 集成

在 `MonitorScheduler` 的 `notify()` 回调中，除了 WebSocket 推送外，同步更新 Prometheus Gauge：

```
MonitorScheduler.runMonitorChecks()
  → svc.CheckAllMonitors()
  → notify("uptime_check", results)
    → WebSocket Broadcast (已有)
    → metrics.UpdateMonitorGauges(results)  (新增)
```

### 完整指标清单（合并后）

| 指标 | 类型 | 标签 | 来源 |
|------|------|------|------|
| `deploypilot_deploy_total` | Counter | app, server, status | 已有 (deploy_service) |
| `deploypilot_deploy_duration_seconds` | Histogram | — | 已有 (deploy_service) |
| `deploypilot_active_containers` | Gauge | — | 已有 |
| `deploypilot_ws_connections` | Gauge | — | 已有 |
| `deploypilot_api_request_duration_seconds` | Histogram | method, path, status | 已有 |
| `deploypilot_credential_expiry_days` | Gauge | name | 已有 |
| `deploypilot_monitor_up` | Gauge | name, type, target | **新增** |
| `deploypilot_monitor_latency_ms` | Gauge | name, type, target | **新增** |
| `deploypilot_monitor_uptime_pct` | Gauge | name, type, target | **新增** |
| `deploypilot_heartbeat_up` | Gauge | name | **新增** |

加上 Prometheus Go runtime 默认指标（`go_*`, `process_*`），总共约 20+ 指标。

## 文件变更清单

### 后端修改

| 文件 | 变更 |
|------|------|
| `internal/metrics/metrics.go` | 新增 4 个 GaugeVec + `UpdateMonitorGauges()` + `UpdateHeartbeatGauges()` 函数 |
| `internal/service/monitor_scheduler.go` | 在 `notify()` 中调用 metrics 更新函数 |
| `internal/api/router.go` | 删除 `/api/v1/metrics` 旧路由；注册 `/metrics` 新路由（认证/公开双注册） |
| `internal/service/monitor_service.go` | 删除 `GetPrometheusMetrics()` 方法 |
| `internal/api/monitor_api.go` | 删除 `GetPrometheusMetrics` handler |
| `internal/config/config.go` | `MonitorConfig` 删除 `MetricsPort`，新增 `MetricsPublic` |
| `cmd/api-server/main.go` | 删除独立 metrics HTTP 服务器代码 |
| `cmd/deploypilot/serve.go` | 删除独立 metrics HTTP 服务器代码 |
| `configs/config.yaml.example` | 更新 monitor 配置段 |

### 前端修改

| 文件 | 变更 |
|------|------|
| `web/src/views/MonitorSettings.vue` | 新增「Prometheus 指标公开访问」开关 |

## 删除清单

- `MonitorService.GetPrometheusMetrics()` 方法（约 35 行手工拼接代码）
- `MonitorAPI.GetPrometheusMetrics` handler
- `/api/v1/metrics` 路由
- 端口 9091 独立 HTTP 服务器（main.go 和 serve.go 中各约 15 行）
- `MonitorConfig.MetricsPort` 字段

## 向后兼容性

- **Breaking Change**: 端口 9091 将不再可用；使用 Prometheus scrape 配置了 `:9091` 的用户需要改为 `http://<host>:<api-port>/metrics`
- **Breaking Change**: `/api/v1/metrics` 路径将不再可用；需改为 `/metrics`
- **Migration**: 在 Release Notes 中注明变更，提供配置迁移指引

## 测试策略

- 单元测试：`internal/metrics/metrics_test.go` — 验证 Gauge 注册、更新、标签正确性
- 集成测试：验证 `/metrics` 端点返回标准 Prometheus 格式
- 认证测试：验证默认需 JWT、公开模式下无需认证

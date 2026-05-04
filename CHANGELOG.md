# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.9.0] - 2026-05-04

### Added
- **License key rotation**: Ed25519 key pair rotation with auto-incrementing version numbers
- **Multi-key verification**: License Engine verifies against all trusted public keys
- **Key versioning**: Database-backed `license_signing_keys` table with version tracking
- **Shamir's Secret Sharing**: Split private key into N shares (threshold M) using GF(256)
- **Key rotation API**: 4 REST endpoints (rotate, list, version, backup-shamir)
- **Startup key loading**: All trusted keys loaded from DB at startup for rotation support

### Changed
- License Engine now maintains a key chain (`publicKeys []ed25519.PublicKey`)
- `LoadLicense()` verifies signature against all trusted keys (not just active)
- `RotatePublicKey()` replaces active key while retaining old keys for verification

## [1.8.0] - 2026-05-03

### Added
- **License Engine v2**: 4-tier system (Free/Pro/Enterprise/Custom) with UseType and Addon licensing
- **License pause/resume**: Suspend license with configurable resume date
- **License API**: 5 REST endpoints for license management (activate, status, list, revoke, validate)
- **CLI keygen tool**: `deploypilot license keygen --tier pro --days 365` with RSA-2048 key generation
- **License Frontend**: Management page with activation, status display, and license list
- **Feature Flags**: Dynamic feature gating with 5-min cache, per-tenant overrides, and tier-based evaluation
- **Feature Flags API**: 7 REST endpoints (CRUD + override + evaluate + categorize)
- **Feature Flags Frontend**: Management page with toggle, override dialog, and category filter
- **Trial Period**: 30-day auto-activation with SHA256 machine fingerprint (hostname|os|arch|cpu)
- **Trial conversion**: Auto-convert to licensed status when a license is activated
- **Trial extension**: Admin can extend trial by 1-365 days with reason
- **Degradation strategy**: 3-level system (none/partial/readonly) with audit trail
- **Read-only middleware**: Blocks POST/PUT/DELETE/PATCH when license/trial expired
- **Degradation audit**: Compliance logging for all access denials
- **Batch deploy API**: REST endpoint for sequential/parallel/rolling batch deployments
- **Clusters page**: Full management UI with create/edit dialogs and connection testing
- **Registries page**: Container registry management with provider badges
- **Plugins page**: Plugin lifecycle management with enable/disable/reload and provider filter
- **Activity Feed page**: Event log viewer with type filter, pagination, and stats
- **Batch Operations page**: Batch deploy form with strategy selection and results panel

### Changed
- Replaced 5 stub "coming soon" pages with fully functional management pages
- Updated i18n with 160+ new translations (English + Chinese)

### Security
- License RSA key pair generation and JWT signing
- Machine fingerprint prevents casual trial resets
- Audit trail for degradation events (action, feature, reason, tenant, user, IP)

## [1.10.0]

### Added
- Database migration system with golang-migrate (SQL files, up/down support, legacy compatibility)
- API versioning middleware with X-API-Version, Deprecation, Sunset headers
- Local development environment (docker-compose.dev.yml, Makefile targets, Air hot reload)
- Community documentation (CONTRIBUTING.md, CODE_OF_CONDUCT.md, ADR templates, CLAUDE.md)
- Seed data script for demo environments
- VS Code launch configuration

## [1.9.0]

### Added
- Audit log hash chain verification (HMAC-SHA256 tamper detection)
- GDPR data export/deletion and compliance report generation
- Audit log export (CSV/JSON formats)
- Per-user IP whitelist with CIDR support and enforcement middleware
- Device binding with trust management and new device detection
- Ed25519 code signing with key generation, rotation, and CLI tool

## [1.7.0] - 2026-04-30

### Added
- SSH known_hosts support with strict/non-strict host key checking modes
- Plugin config decryption using AES-256-GCM (values prefixed with "enc:")
- Changelog automation workflow (GitHub Actions)

## [1.6.0] - 2026-04-30

### Added
- Panel security hardening (security entrance, domain binding, IP whitelist)
- Password policy enforcement (min length, complexity requirements, expiry)
- 2FA/TOTP enforcement with role-based grace periods

## [1.5.0] - 2026-04-30

### Added
- Grafana integration (annotations sync, dashboard management)
- API Open Platform (OAuth2 client credentials flow)
- API versioning configuration

## [1.4.0] - 2026-04-30

### Added
- **Two-Factor Authentication (2FA/TOTP)** (PR #172): TOTP 服务端实现、两步登录流程、2FA 启用/禁用管理、AES-256-GCM 加密 secret 存储、一次性备份码生成
- **API Key 管理系统** (PR #171): API Key CRUD、scope 权限控制、可配置过期时间、key 前缀展示
- **Credential Vault 加固** (PR #170): HKDF 密钥派生增强、凭证加密存储改进、审计日志集成
- **审计日志增强** (PR #169): 时间范围过滤、保留策略配置、资源类型过滤
- **MCP 会话上下文记忆** (PR #167): list_recent_operations 工具、会话级操作历史
- **面板网站管理** (PR #166): DeleteReverseProxy、CreateWebsite、GetWebsiteList 面板工具
- **企业级前端 UI** (PR #131, #173): API Keys 管理页面、安全设置页面（2FA/TOTP 设置）、Login 2FA 验证步骤、Credentials 审计历史 Tab、侧边栏导航更新、~78 条 i18n 中英文翻译

### Changed
- **前端类型系统** (PR #131): LoginResponse 扩展 2FA 可选字段、新增 ApiKey/TwoFASetupResponse 接口、User 模型添加 totp_enabled 字段
- **Auth Store** (PR #131): 新增 pending2FAToken/pending2FAUserId 状态、requires2FA computed、verify2FACode/clear2FAPending 方法

## [1.3.0] - 2026-04-30

### Added
- **监控数据持久化** (PR #109): MetricRecord + AlertHistory GORM 模型，MetricStore（SaveMetrics/QueryMetrics/CleanupOldMetrics），Monitor 自动持久化，query_metric_history/query_alert_history MCP 工具
- **计划任务系统** (PR #109): robfig/cron/v3 调度器（秒级精度），ScheduledTask + TaskExecution 模型，3 种任务类型（shell/health_check/log_cleanup），5 个 MCP 工具（create/list/get_executions/toggle/delete）
- **Import cycle 修复** (PR #109): 修复 model 测试文件中的 import cycle（model test → database → model），改用直接 GORM SQLite
- **WebSocket 多实例广播** (PR #107): WSHub 通过 Redis Pub/Sub 实现跨实例消息广播，SourceInstance 防止消息回环，Redis 不可用时自动降级为本地模式
- **系统监控增强** (PR #107): 新增网络 I/O 指标（rx/tx bytes/packets/errors）、磁盘 I/O 指标（reads/writes/sectors）、远程服务器监控（server_id 参数）、修复 container_memory_used 类型 bug
- **ai-guide 踩坑 #24-#28**: CREATE TABLE 同步、mockDeployer 同步、Go 变量作用域、import 清理、Redis Pub/Sub 回环防止
- **多环境分离** (PR #105): App 模型新增 Environment 字段（production/staging/development/testing），API 支持 environment 过滤，MCP 工具接受 environment 参数
- **Redis 缓存层** (PR #105): Cache 接口 + RedisCache/MemoryCache 实现，权限检查缓存（5min TTL），DNS 记录缓存（10min TTL），路由/WS 中间件切换到缓存版本
- **环境变量模板** (PR #103): MySQL/Redis/PostgreSQL/MongoDB/Nginx 预置环境变量模板，2 个 MCP 工具（list_env_templates, get_env_template）
- **部署前预检增强** (PR #103): 新增磁盘空间检查（< 1GB error, < 5GB warning）+ 内存检查（< 128MB error, < 512MB warning），RunPreflightFull 全量非短路模式，run_preflight MCP 工具
- **Docker Compose 部署支持** (PR #101): ComposeDeployer 引擎（up/down/ps/logs/restart），App 模型新增 ComposeContent/ComposeProjectName 字段，5 个 compose MCP 工具
- **exec_command MCP 工具** (PR #101): 在本地/远程服务器执行任意命令（admin 级别）
- **list_images MCP 工具** (PR #101): 列出 Docker 镜像，支持 grep 过滤（viewer 级别）
- **port_forward MCP 工具** (PR #101): SSH 端口转发管理 create/delete/list（dev 级别）

## [1.2.0] - 2026-04-28

### Changed
- **Bridge God Object 拆分** (PR #85-#88): bridge.go 从 2986 行拆分为 18 个文件 448 行，87 个方法按领域分布到独立 service 文件
- **MCP Server 拆分** (PR #90): server.go 从 2510 行拆分为 34 个文件 71 行，63 个工具按领域分布
- **适配器注册表集成** (PR #93): DNS/Notify/CI/CD/Registry 服务消除 switch/case 硬编码，统一使用 plugin.Registry 查找和创建 provider 实例
- **Panel 适配器重构** (PR #96): 定义 PanelClient 接口，PanelProvider 改为依赖注入，消除 4 处 switch/case
- **OAuth 适配器重构** (PR #96): 定义 OAuthProvider 接口，提取 GitHub/Gitee provider 到独立文件，消除 3 处 switch/case
- **SSL 适配器重构** (PR #98): 定义 CertificateProvider 接口，注册 self-signed-ssl 插件
- **Database 驱动注册表** (PR #98): DriverFactory + RegisterDriver 替代 switch/case
- **熔断器** (PR #98): 通用 CircuitBreaker 实现（Closed/Open/HalfOpen 三态）
- **许可证切换** (PR #98): MIT → BUSL-1.1（Change Date: 2029-04-28, Change License: MIT）
- **plugin.Global() 自动注册** (PR #93): Global() 首次访问时自动调用 RegisterBuiltinPlugins()，确保所有内置 provider 可用

### Added
- **前端 Vitest 测试基础设施** (PR #91): 新增 vitest + @vue/test-utils + jsdom，14 个单元测试（utils + theme store），CI 集成 npm test
- **AI 操作指南** (PR #89, #92, #94): 17 条踩坑记录，项目结构路径表，MCP Server 层架构表，前端架构表

## [1.1.0] - 2026-04-27

### Added
- **Credential encryption**: AES-256-GCM encryption for SSH keys, passwords, and API tokens at rest
- **Remote deployment**: SSH-based remote deployment via server registration and credential management
- **Backup & restore**: Database backup and restore with configurable retention policies
- **Brute-force protection**: Progressive delay, account lockout, and IP-based rate limiting for login
- **Audit logging**: Comprehensive audit trail for all sensitive operations with external log support
- **Request tracing**: Distributed tracing support for debugging requests across services
- **Batch operations**: Batch deploy, batch backup, and batch DNS management MCP tools
- **Application templates**: 9 technology stack templates for quick app creation
- **Health check**: HTTP/TCP health probe MCP tool
- **Server connectivity test**: SSH connection verification MCP tool
- **Notification system**: Deployment and rollback event notifications
- **OAuth login**: GitHub and Gitee OAuth2 authentication with CSRF protection
- **Multi-platform releases**: Automated binary builds for linux/amd64 and linux/arm64
- **Docker image publishing**: Automated Docker image builds pushed to GHCR
- **Install script**: One-line installation script downloading from GitHub Releases
- **1Panel integration**: Firewall management and reverse proxy creation
- **BaoTa panel integration**: Server panel provider support
- **Monitoring**: System and container metrics collection with alerting
- **CI/CD integration**: GitHub and Gitea webhook-based CI/CD pipeline support
- **SSL certificate management**: Certificate lifecycle management
- **i18n**: English and Chinese localization for web dashboard
- **Web dashboard**: Full-featured Vue.js frontend with responsive design

### Changed
- Upgraded CI workflows to use latest GitHub Actions versions
- Improved test coverage to ~89.6%
- Enhanced RBAC with 4 roles: owner, admin, dev, viewer
- Refined MCP tool naming to align with PRD specification

## [1.0.0] - 2026-04-20

### Added
- **MCP Server**: Model Context Protocol server with 31 core tools for AI IDE integration
- **REST API**: Full REST API with JWT authentication
- **Deployment engine**: Docker-based application deployment with automatic detection
- **Container management**: Container lifecycle management (start, stop, restart, remove)
- **DNS management**: Multi-provider DNS record management (Aliyun, Tencent, WestDNS)
- **Server management**: SSH server registration and management
- **Credential management**: Secure credential storage for deployment targets
- **User management**: Registration, login, and role-based access control
- **WebSocket**: Real-time deployment log streaming
- **SSE**: Server-Sent Events for live status updates
- **Web dashboard**: Vue.js single-page application with Tailwind CSS
- **Docker Compose deployment**: One-command deployment with Docker Compose
- **Swagger documentation**: Auto-generated API documentation
- **Rate limiting**: Per-role API rate limiting
- **Configuration system**: YAML-based configuration with environment variable overrides

### Security
- JWT-based authentication with configurable expiration
- RBAC with owner, admin, dev, and viewer roles
- CORS configuration
- Input validation and sanitization

[1.4.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.4.0
[1.3.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.3.0
[1.2.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.2.0
[1.1.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.1.0
[1.0.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.0.0
[1.5.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.5.0
[1.6.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.6.0
[1.7.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.7.0
[1.8.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.8.0
[1.9.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.9.0
[1.10.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.10.0

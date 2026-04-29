# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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

[1.2.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.2.0
[1.1.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.1.0
[1.0.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.0.0

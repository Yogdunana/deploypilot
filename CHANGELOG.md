# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- **Bridge God Object 拆分** (PR #85-#88): bridge.go 从 2986 行拆分为 18 个文件 448 行，87 个方法按领域分布到独立 service 文件
- **MCP Server 拆分** (PR #90): server.go 从 2510 行拆分为 34 个文件 71 行，63 个工具按领域分布
- **适配器注册表集成** (PR #93): DNS/Notify/CI/CD/Registry 服务消除 switch/case 硬编码，统一使用 plugin.Registry 查找和创建 provider 实例
- **Panel 适配器重构** (PR #96): 定义 PanelClient 接口，PanelProvider 改为依赖注入，消除 4 处 switch/case
- **OAuth 适配器重构** (PR #96): 定义 OAuthProvider 接口，提取 GitHub/Gitee provider 到独立文件，消除 3 处 switch/case
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

[1.1.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.1.0
[1.0.0]: https://github.com/Yogdunana/deploypilot/releases/tag/v1.0.0

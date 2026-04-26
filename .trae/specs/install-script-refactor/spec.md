# DeployPilot 一键安装脚本优化 - 产品需求文档

## Overview
- **Summary**: 完善 DeployPilot 的一键安装脚本，提升用户体验，增加国内源支持、自定义配置和美观的 TUI 界面
- **Purpose**: 简化 DeployPilot 的安装流程，让用户能够快速、便捷地在服务器上部署 DeployPilot
- **Target Users**: 服务器管理员、开发者、想要快速部署 DeployPilot 的用户

## Goals
1. 学习并借鉴 1Panel 和宝塔的安装脚本优秀实践
2. 提供美观、友好的 TUI 安装界面
3. 支持自定义安装路径、端口、用户名和密码
4. 增加国内源支持，提升下载速度
5. 添加安装前确认流程
6. 提供一键安装和命令行参数两种使用方式

## Non-Goals
1. 不修改 DeployPilot 的核心功能代码
2. 不添加超出安装脚本范围的功能
3. 不实现复杂的用户交互功能（如在线升级等）

## Background & Context
- 当前 DeployPilot 已有基础的部署脚本，但功能较为简单
- 1Panel 和宝塔的安装脚本提供了很好的参考，具有：
  - 美观的 TUI 界面
  - 交互式配置
  - 国内源支持
  - 自动环境检测和依赖安装
  - 完善的服务管理

## Functional Requirements
1. **美观的 TUI 界面**
   - 使用颜色和格式化输出来提升视觉效果
   - 清晰的步骤提示
   - 友好的用户交互
   
2. **自定义配置**
   - 支持自定义安装路径（默认 /opt/deploypilot）
   - 支持自定义端口（默认 8080）
   - 支持自定义用户名（默认随机生成）
   - 支持自定义密码（默认随机生成）
   - 直接回车使用默认值
   
3. **国内源支持**
   - 提供国内镜像源选项
   - 提升安装包下载速度
   
4. **安装确认流程**
   - 显示安装配置摘要
   - 要求用户确认后再继续安装
   
5. **环境检测和依赖安装**
   - 自动检测系统架构和版本
   - 自动安装 Docker（如未安装）
   - 自动配置系统环境
   
6. **服务管理**
   - 自动配置 systemd 服务
   - 设置开机自启
   - 提供服务控制命令
   
7. **安装完成提示**
   - 显示访问地址
   - 显示用户名和密码
   - 提供安全提示

## Non-Functional Requirements
1. **稳定性**: 安装脚本在各种 Linux 发行版上都能正常工作
2. **可维护性**: 代码结构清晰，注释完善
3. **用户友好**: 错误提示友好，问题定位简单
4. **安全性**: 密码生成有足够的强度

## Constraints
- **技术**: 使用 bash 脚本实现，兼容主流 Linux 发行版
- **依赖**: 依赖 curl、wget 等常用工具
- **兼容性**: 支持 x86_64、arm64 架构
- **系统**: 支持 CentOS、Ubuntu、Debian 等主流发行版

## Assumptions
1. 用户有 root 或 sudo 权限
2. 服务器可以访问互联网
3. 目标系统未安装 DeployPilot 或已完全卸载

## Acceptance Criteria

### AC-1: 美观的 TUI 界面
- **Given**: 用户执行安装脚本
- **When**: 脚本运行时
- **Then**: 显示美观的彩色输出和清晰的步骤提示
- **Verification**: human-judgment

### AC-2: 自定义配置功能
- **Given**: 用户执行安装脚本
- **When**: 被提示输入配置时
- **Then**: 可以输入自定义值或直接回车使用默认随机值
- **Verification**: programmatic

### AC-3: 国内源支持
- **Given**: 用户执行安装脚本
- **When**: 选择使用国内源
- **Then**: 从国内镜像源下载安装包，速度更快
- **Verification**: programmatic

### AC-4: 安装确认流程
- **Given**: 用户完成配置输入
- **When**: 脚本显示配置摘要
- **Then**: 用户确认后才开始安装
- **Verification**: programmatic

### AC-5: 环境检测和依赖安装
- **Given**: 用户执行安装脚本
- **When**: 脚本开始运行
- **Then**: 自动检测系统环境，安装必要的依赖
- **Verification**: programmatic

### AC-6: 服务管理
- **Given**: 安装完成
- **When**: 系统重启
- **Then**: DeployPilot 服务自动启动
- **Verification**: programmatic

### AC-7: 安装完成提示
- **Given**: 安装完成
- **When**: 脚本结束
- **Then**: 显示访问地址、用户名、密码和安全提示
- **Verification**: human-judgment

## Open Questions
- 是否需要支持非 systemd 系统？（已有基础实现，保持即可）
- 是否需要提供离线安装选项？（暂不考虑）

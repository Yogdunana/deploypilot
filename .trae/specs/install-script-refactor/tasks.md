# DeployPilot 一键安装脚本优化 - 实现计划

## [ ] Task 1: 颜色和格式化输出功能
- **Priority**: P0
- **Depends On**: None
- **Description**: 
  - 定义 ANSI 颜色代码和格式化函数
  - 实现标题、成功、错误、警告等提示样式
  - 实现进度条和步骤指示器
- **Acceptance Criteria Addressed**: AC-1
- **Test Requirements**:
  - `human-judgment`: 输出视觉效果美观
  - `programmatic`: 颜色函数正确工作

## [ ] Task 2: 系统环境检测功能
- **Priority**: P0
- **Depends On**: Task 1
- **Description**:
  - 检测系统架构（x86_64、arm64 等）
  - 检测 Linux 发行版和版本
  - 检测 systemd 是否可用
  - 检测网络连接
- **Acceptance Criteria Addressed**: AC-5
- **Test Requirements**:
  - `programmatic`: 架构检测正确
  - `programmatic`: 发行版识别准确

## [ ] Task 3: 交互式配置输入功能
- **Priority**: P0
- **Depends On**: Task 1
- **Description**:
  - 实现安装路径输入（默认 /opt/deploypilot）
  - 实现端口输入（默认 8080）
  - 实现用户名输入（默认随机生成）
  - 实现密码输入（默认随机生成）
  - 实现国内源选择
  - 生成随机字符串的函数
- **Acceptance Criteria Addressed**: AC-2
- **Test Requirements**:
  - `programmatic`: 空输入使用默认值
  - `programmatic`: 随机字符串生成正常
  - `programmatic`: 自定义输入正确保存

## [ ] Task 4: 配置摘要和确认流程
- **Priority**: P0
- **Depends On**: Task 1, Task 3
- **Description**:
  - 显示完整的安装配置摘要
  - 等待用户确认
  - 确认后继续，否则退出
- **Acceptance Criteria Addressed**: AC-4
- **Test Requirements**:
  - `programmatic`: 配置摘要显示完整
  - `programmatic`: 确认流程正常工作

## [ ] Task 5: 依赖安装功能（Docker）
- **Priority**: P0
- **Depends On**: Task 1, Task 2
- **Description**:
  - 检查 Docker 是否已安装
  - 如未安装，使用国内源自动安装
  - 启动并启用 Docker 服务
- **Acceptance Criteria Addressed**: AC-5
- **Test Requirements**:
  - `programmatic`: Docker 安装成功
  - `programmatic`: Docker 服务正常运行

## [ ] Task 6: 主程序安装和配置
- **Priority**: P0
- **Depends On**: Task 1, Task 3, Task 5
- **Description**:
  - 创建安装目录结构
  - 下载 DeployPilot 二进制文件（支持国内源）
  - 创建配置文件
  - 设置正确的权限
- **Acceptance Criteria Addressed**: AC-2, AC-3
- **Test Requirements**:
  - `programmatic`: 目录结构创建正确
  - `programmatic`: 文件权限设置正确
  - `programmatic`: 配置文件生成正确

## [ ] Task 7: 服务配置（systemd）
- **Priority**: P0
- **Depends On**: Task 6
- **Description**:
  - 创建 systemd 服务文件
  - 启用并启动服务
  - 设置开机自启
  - 提供非 systemd 系统的备选方案
- **Acceptance Criteria Addressed**: AC-6
- **Test Requirements**:
  - `programmatic`: 服务文件创建正确
  - `programmatic`: 服务启动成功
  - `programmatic`: 开机自启已设置

## [ ] Task 8: 防火墙配置
- **Priority**: P1
- **Depends On**: Task 2, Task 3
- **Description**:
  - 检测并配置防火墙（ufw、firewalld）
  - 检测 1Panel 并同步防火墙规则
  - 开放 DeployPilot 端口
- **Acceptance Criteria Addressed**: AC-5
- **Test Requirements**:
  - `programmatic`: 防火墙规则正确添加

## [ ] Task 9: 安装完成提示
- **Priority**: P0
- **Depends On**: Task 1, Task 3, Task 6, Task 7
- **Description**:
  - 获取服务器 IP 地址
  - 显示访问地址
  - 显示用户名和密码
  - 提供安全提示
  - 提供管理命令说明
- **Acceptance Criteria Addressed**: AC-7
- **Test Requirements**:
  - `human-judgment`: 提示信息完整美观
  - `programmatic`: IP 地址获取正确

## [ ] Task 10: 脚本集成和测试
- **Priority**: P0
- **Depends On**: Task 1-9
- **Description**:
  - 集成所有功能到完整脚本
  - 添加错误处理和回滚机制
  - 添加日志记录
  - 测试脚本在不同场景下的工作
- **Acceptance Criteria Addressed**: AC-1, AC-2, AC-3, AC-4, AC-5, AC-6, AC-7
- **Test Requirements**:
  - `programmatic`: 脚本能正常执行完成
  - `programmatic`: 错误处理正常工作
  - `human-judgment`: 整体体验良好

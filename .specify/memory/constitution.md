<!--
===============================================================================
SYNC IMPACT REPORT
===============================================================================
Version Change: 0.0.0 → 1.0.0 (MAJOR - Initial constitution ratification)

Modified Principles: N/A (Initial version)

Added Sections:
  - I. Monorepo 微服务边界
  - II. 配置是信息的唯一来源
  - III. 服务生命周期一致性
  - IV. 可观测性与错误规范
  - V. 安全与认证规则
  - VI. 依赖与构建规范性
  - VII. 附加约束条件

Removed Sections: N/A (Initial version)

Templates Requiring Updates:
  ✅ .specify/templates/plan-template.md - No updates required (generic template)
  ✅ .specify/templates/spec-template.md - No updates required (generic template)
  ✅ .specify/templates/tasks-template.md - No updates required (generic template)

Follow-up TODOs: None
===============================================================================
-->

# GoBack 微服务框架 Constitution

本宪章定义了 GoBack 项目的最高技术原则与规范。所有代码变更、架构决策和功能实现 **MUST** 遵守以下原则。

## Core Principles

### I. Monorepo 微服务边界

本原则确保单仓多服务架构的边界清晰，防止模块间的不当耦合。

1. 本项目是 Go 单模块（`module github.com/goback`）的"单仓多服务"架构
2. 业务服务代码 **MUST** 放在 `services/<service_name>/internal/...`，入口 **MUST** 放在 `services/<service_name>/cmd/main.go`
3. 公共能力 **MUST** 放在 `pkg/...`（配置、日志、数据库、鉴权、中间件、路由、错误、缓存、广播等）
4. **禁止**把某个服务的 `internal` 包当做公共库被其他服务直接 import
   - `internal` 目录只对本服务生效的边界要保持清晰
   - 跨服务复用 **MUST** 上移到 `pkg/`

**Rationale**: 清晰的边界可防止服务间隐式依赖，确保每个服务可独立构建、测试和部署。

### II. 配置是信息的唯一来源

本原则确保所有运行参数可追溯、可审计、可环境切换。

1. 所有运行参数 **MUST** 来自配置系统（`pkg/config` + `configs/*.yaml` + 环境变量覆盖）
2. **禁止**在业务逻辑里硬编码：端口、依赖地址、密钥、数据库 DSN、服务名、basePath 等
3. 每次新增/修改配置项 **MUST** 同步更新：
   - `configs/config.yaml`（默认）
   - `configs/config.dev.yaml`（开发）
   - `configs/config.prod.yaml`（生产，如果适用）
   - 相关 README/部署说明（若影响启动方式）

**Rationale**: 单一配置来源消除环境差异导致的故障，简化运维和调试。

### III. 服务生命周期一致性

本原则确保所有服务具有统一的启动、运行和停止行为。

1. 服务启动/就绪/停止的结构 **MUST** 保持一致，优先复用 `pkg/lifecycle` 现有模式（OnStart/OnReady/OnStop、事件监听）
2. HTTP 框架以 Fiber 为准（现有服务已使用），路由注册优先走 `pkg/router` 约定（Controller + types.go）
3. 任何新增服务 **MUST** 明确：
   - `serviceName`（注册名）
   - node ID 规则
   - HTTP 监听地址/端口
   - 健康检查 `/health`

**Rationale**: 统一的生命周期管理简化监控、编排和故障排查。

### IV. 可观测性与错误规范

本原则确保系统行为可追踪、问题可定位。

1. 统一使用 `pkg/logger`（Zap）输出结构化日志
   - **禁止** `fmt.Println/Printf`（仅允许在配置尚未初始化前的极早期错误输出）
2. 日志 **MUST** 避免泄露敏感信息（JWT、密码、数据库密码、私钥、token 等）
3. 错误返回要可定位：
   - 优先使用 `pkg/errors` 与统一响应
   - **MUST** 保证调用方能分辨：未找到/参数错误/依赖不可用/内部错误

**Rationale**: 结构化日志和分类错误码是生产环境快速定位问题的基础。

### V. 安全与认证规则

本原则确保系统默认安全，任何例外必须显式声明。

1. 身份认证/授权以 `pkg/auth`（JWT、密码哈希）与 RBAC 约定为准
   - 任何绕过鉴权的变更 **MUST** 显式说明风险与范围
2. 新增接口 **MUST** 明确：
   - 是否需要 JWT
   - 是否需要 RBAC/数据权限（DataScope）
   - 拒绝策略：未授权 **MUST** 失败，而不是静默放行

**Rationale**: 显式安全声明防止权限遗漏，拒绝策略确保安全优先。

### VI. 依赖与构建规范性

本原则确保代码质量和跨平台兼容性。

1. Go 代码 **MUST** 通过 `gofmt`（`make fmt`）
2. 依赖变更后 **MUST** 保持 `go.mod`/`go.sum` 干净（`make deps` / `go mod tidy`）
3. 只在必要时引入新依赖；优先标准库或仓库已有 `pkg` 能力
4. 变更 **MUST** 兼容 Windows（当前开发环境）、Mac（需支持开发环境）与 Docker 部署路径

**Rationale**: 规范的依赖管理减少构建问题，跨平台兼容确保团队协作顺畅。

### VII. 附加约束条件

本原则涵盖部署和基础设施相关的强制要求。

1. Docker Compose 是官方主路径之一（`deployments/docker/docker-compose.yml`）
   - 影响服务启动顺序/依赖时 **MUST** 同步更新 compose
2. 本仓库存在"缓存微服务（redis-service，内存模式）"以及 `pkg/cache` 的 HTTP 客户端模式：
   - 默认假设内部网络调用
   - 端点/协议变更属于**破坏性变更**，**MUST** 同步修改 `pkg/cache` 与相关服务初始化逻辑
3. Makefile 是标准入口（`deps`/`fmt`/`test`/`build`/`run-*`）
   - 新增服务若希望纳入一键构建/运行，**MUST** 同步 Makefile 的 SERVICES 列表与相关目标

**Rationale**: 基础设施同步确保部署可重复、可预测。

## Quality Gates

代码合并前必须通过以下质量门禁：

| Gate | 要求 | 验证方式 |
|------|------|----------|
| 格式化 | 代码通过 `gofmt` | `make fmt` |
| 依赖卫生 | `go.mod`/`go.sum` 干净 | `make deps` |
| 边界检查 | 无跨服务 internal import | Code Review |
| 配置完整 | 新配置项已同步到所有环境文件 | Code Review |
| 安全声明 | 新接口已声明鉴权要求 | Code Review |

## Development Workflow

### 新增服务检查清单

新增服务前，**MUST** 确认以下事项：

- [ ] 服务名和端口已在 `configs/*.yaml` 中定义
- [ ] 入口文件位于 `services/<name>/cmd/main.go`
- [ ] 业务代码位于 `services/<name>/internal/`
- [ ] 已实现 `/health` 健康检查端点
- [ ] 已复用 `pkg/lifecycle` 生命周期模式
- [ ] 已添加到 `Makefile` 的 SERVICES 列表
- [ ] 已添加到 `docker-compose.yml`（如适用）

### 配置变更检查清单

修改配置时，**MUST** 同步更新：

- [ ] `configs/config.yaml`
- [ ] `configs/config.dev.yaml`
- [ ] `configs/config.prod.yaml`（如适用）
- [ ] 相关 README 文档

## Governance

### 宪章权威

1. 本宪章是项目的最高技术规范，其效力高于任何其他文档或实践
2. 所有 PR/Code Review **MUST** 验证是否符合宪章原则
3. 任何违反宪章的代码变更 **MUST** 显式说明原因并获得审批

### 修订流程

1. 宪章修订提案 **MUST** 以 PR 形式提交
2. 修订 **MUST** 包含：
   - 变更内容说明
   - 变更理由
   - 影响范围评估
   - 迁移计划（如适用）
3. 修订经审批后方可合并

### 版本规则

遵循语义化版本：

- **MAJOR**: 移除或重新定义核心原则（破坏性变更）
- **MINOR**: 新增原则/章节或重大扩展
- **PATCH**: 澄清、措辞修正、非语义性调整

### 合规检查

- 定期审查代码库是否符合宪章要求
- 新功能规划时 **MUST** 确认宪章合规性
- 架构决策 **MUST** 引用相关宪章原则

**Version**: 1.0.0 | **Ratified**: 2026-01-15 | **Last Amended**: 2026-01-15

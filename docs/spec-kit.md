# 用Spec-kit SDD规范驱动开发



- 命令 : /constitution 建立项目宪章，明确必须遵守的最高原则
- 要求其遵循代码质量（golangci-lint ） 测试标准（TDD、测试覆盖率）、性能要求
- 本项目宪章

```txt
- Monorepo微服务边界
    1. 本项目是 Go 单模块（module github.com/goback）的“单仓多服务”架构。
    2. 业务服务代码必须放在 services/<service_name>/internal/...，入口必须放在 services/<service_name>/cmd/main.go。
    3. 公共能力必须呀放在 pkg/... （配置、日志、数据库、鉴权、中间件、路由、错误、缓存、广播等）。
    4. 禁止把某个服务的 internal 包当做公共库被其他服务直接 import（internal 目录只对本服务生效的边界要保持清晰：跨服务复用请上移到pkg）。
- 配置是信息的唯一来源 
    1. 所有运行参数必须来自配置系统（pkg/config + configs/*.yaml + 环境变量覆盖）。
    2. 禁止在业务逻辑里硬编码：端口、依赖地址、密钥、数据库 DSN、服务名、basePath 等。
    3. 每次新增/修改配置项必须同步更新：
        3.1 configs/config.yaml（默认）
        3.2 configs/config.dev.yaml（开发）
        3.3 configs/config.prod.yaml（生产，如果适用）
        3.4 以及相关 README/部署说明（若影响启动方式）
- 服务生命周期一致性
    1. 服务启动/就绪/停止的结构保持一致，优先复用 pkg/lifecycle 现有模式（OnStart/OnReady/OnStop、事件监听）。
    2. HTTP框架以 Fiber 为准（现有服务已使用），路由注册优先走 pkg/router 约定（Controller + types.go）
    3. 任何新增服务必须明确
        3.1 serviceName（注册名）
        3.2 node ID 规则
        3.3 HTTP 监听地址/端口
        3.4 健康检查 /health
- 可观测性与错误规范
    1. 统一使用 pkg/logger（Zap）输出结构化日志；避免 fmt.Println/Printf（仅允许在配置尚未初始化前的极早期错误输出）。
    2. 日志必须避免泄露敏感信息（JWT、密码、数据库密码、私钥、token 等）。
    3. 错误返回要可定位：优先使用 pkg/errors 与统一响应（如服务已有约定），并保证调用方能分辨“未找到/参数错误/依赖不可用/内部错误”。
- 安全与认证规则
    1. 身份认证/授权以 pkg/auth（JWT、密码哈希）与 RBAC 约定为准；任何绕过鉴权的变更都必须显式说明风险与范围。
    2. 新增接口必须明确：
        2.1 是否需要JWT
        2.2 是否需要RBAC/数据权限（DataScope）
        2.3 是否决绝策略（未授权应失败，而不是静默放行）
- 依赖与构建规范性
    1. Go 代码必须通过 gofmt（make fmt）。
    2. 依赖变更后必须保持 go.mod/go.sum 干净（make deps / go mod tidy）。
    3. 只在必要时引入新依赖；优先标准库或仓库已有 pkg 能力。
    4. 变更必须兼容 Windows（当前开发环境）和 Mac（需支持开发环境）与 Docker 部署路径。
- 附加约束条件
    1.  Docker Compose 是官方主路径之一（deployments/docker/docker-compose.yml）。影响服务启动顺序/依赖时必须同步更新 compose。
    2. 本仓库存在“缓存微服务（redis-service，内存模式）”以及 pkg/cache 的 HTTP 客户端模式：
        2.1 默认假设内部网络调用
        2.2 端点/协议变更属于破坏性变更，必须同步修改 pkg/cache 与相关服务初始化逻辑
    3.  Makefile 是标准入口（deps/fmt/test/build/run-*）。新增服务若希望纳入一键构建/运行，需同步 Makefile 的 SERVICES 列表与相关目标。
```



- 命令： /specify  指定需求、构建TODO(建议按模块进行拆分),每个新需求就是一个新 specify（不要说明任何技术细节），把自己处于产品经理，提出需求PR
- 参考:

```txt
构建系统参数模块服务，用来存储系统运行时的一些必要配置，可以对参数配置分页查询、删除、新增、更新、获取、按键名获取值、备注
```

- 命令： /speckit.clarify 进一步澄清需求细节

- 命令：/plan 指定实现计划，提供实现计划、架构设计、API规范、数据模型、外部埋点和集成
- 参考：

```txt
- 项目环境和依赖技术栈
1. Language/Version: Go 1.25
2. Primary Dependencies: gofiber/fiber v2, go-micro.dev/v5, gorm, viper, zap
3. Storage: MySQL / PostgreSQL / SQLite（按环境配置）
4. Testing: go test（Makefile 已提供 test/cover）
5. Target Platform: Windows 开发 + Docker/Linux 部署（docker-compose）
6. Project Type: 单仓多服务（services/*）+ 公共库（pkg/*）
7. Performance Goals: N/A（本 feature 为文档与规范落地）

- 实现要求
1. 必须保持 services/<service>/cmd 与 services/<service_name>/internal 的边界清晰
2. 必须使用配置系统作为真源（不写死地址/端口/密钥）
3. 必须遵守日志与安全规则（不泄露敏感信息）
4. 必须保持构建与依赖卫生（gofmt、go mod tidy）

- 项目架构
goback/
- services/
  - gateway/
    - cmd/main.go
    - internal/...
  - user/
    - cmd/main.go
    - internal/...
  - rbac/
  - menu/
  - log/
  - dict/
  - redis/
    - cmd/main.go
    - internal/redis/...
- pkg/
  - auth/
  - broadcast/
  - cache/
  - config/
  - dal/
  - database/
  - errors/
  - lifecycle/
  - logger/
  - middleware/
  - registry/
  - response/
  - router/
  - ssql/
  - utils/
- configs/
- deployments/docker/
- docs/
- scripts/
- Makefile
- go.mod

Structure Decision: 该仓库属于“单仓多服务”而非 src/ 单项目结构；所有路径与任务描述必须使用上述真实目录。
```

- 命令: /tasks 用来生成执行任务，需自己手动核验是否需要修改

- 命令: /implement 按照tasks 执行任务
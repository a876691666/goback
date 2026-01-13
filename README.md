# GoBack 微服务框架

基于 go-micro 的企业级微服务基础架构，提供完整的用户、权限、菜单、日志、字典管理功能。

## 特性

- 🚀 **微服务架构**: 基于 go-micro v4，支持服务注册发现
- 🔐 **权限管理**: 集成 Casbin，支持 RBAC 权限模型
- 📝 **日志系统**: 基于 Zap，支持日志轮转和多输出
- 🗄️ **ORM**: 集成 GORM，支持 MySQL/PostgreSQL
- 🔄 **缓存**: 集成 Redis，支持数据缓存
- 📊 **SSQL解析器**: 自定义查询语法，支持复杂条件构建
- 🐳 **容器化**: 完整的 Docker 和 Docker Compose 支持

## 项目结构

```
goback/
├── cmd/                        # 各服务启动入口
│   ├── gateway/               # 网关服务 (8080)
│   ├── user/                  # 用户服务 (8081)
│   ├── rbac/                  # 权限服务 (8082)
│   ├── menu/                  # 菜单服务 (8083)
│   ├── log/                   # 日志服务 (8084)
│   └── dict/                  # 字典服务 (8085)
├── internal/                   # 内部包
│   ├── gateway/               # 网关服务实现
│   ├── user/                  # 用户服务实现
│   ├── rbac/                  # 权限服务实现
│   ├── menu/                  # 菜单服务实现
│   ├── log/                   # 日志服务实现
│   ├── dict/                  # 字典服务实现
│   └── model/                 # 数据模型
├── pkg/                        # 公共包
│   ├── config/                # 配置管理 (Viper)
│   ├── logger/                # 日志管理 (Zap)
│   ├── database/              # 数据库连接
│   ├── dal/                   # 数据访问层 (GORM)
│   ├── ssql/                  # SQL解析器
│   ├── auth/                  # 认证授权 (JWT/Casbin)
│   ├── middleware/            # 中间件
│   ├── response/              # 统一响应
│   ├── errors/                # 错误处理
│   └── utils/                 # 工具函数
├── configs/                    # 配置文件
│   ├── config.yaml            # 默认配置
│   ├── config.dev.yaml        # 开发环境
│   ├── config.prod.yaml       # 生产环境
│   └── rbac_model.conf        # Casbin 模型配置
├── docs/                       # 文档
│   ├── architecture.puml      # 架构图
│   ├── ssql_design.puml       # SSQL设计图
│   ├── rbac_model.puml        # RBAC模型图
│   └── service_communication.puml  # 服务通信图
├── deployments/               # 部署配置
│   └── docker/                # Docker配置
│       ├── docker-compose.yml
│       ├── Dockerfile.*
│       └── init.sql
└── Makefile                   # 构建脚本
```

## 核心特性

### 1. 微服务架构 (go-micro)
- 6个核心服务：gateway、user、rbac、menu、log、dict
- 服务注册与发现 (etcd)
- HTTP/gRPC 通信
- 负载均衡与熔断

### 2. ORM 基础建设 (GORM)
- 统一的 Repository 模式
- 通用 CRUD 操作
- 软删除支持
- 分页查询
- 事务管理

### 3. 权限管理 (Casbin)
- 1用户 <-> 1角色
- 1角色 <-> 多权限  
- 1权限 <-> 多(菜单/数据权限)
- 数据权限过滤 (DataScope)

### 4. 日志管理 (Zap)
- 多级别日志 (Debug/Info/Warn/Error)
- 日志轮转
- 结构化日志
- 请求链路追踪

### 5. 配置管理 (Viper)
- 多环境配置
- 配置热更新
- 环境变量支持

### 6. SSQL 解析器
- 简单 SQL 序列化/反序列化
- 支持深层嵌套 `||` 和 `&&`
- 基本比较操作符 (=、!=、>、<、in、like等)

## 快速开始

### 本地开发

```bash
# 安装依赖
make deps

# 构建所有服务
make build

# 运行单个服务
make run-gateway
make run-user
make run-rbac
make run-menu
make run-log
make run-dict
```

### Docker 部署

```bash
# 启动所有服务（包括 MySQL、Redis、Etcd）
cd deployments/docker
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f gateway

# 停止服务
docker-compose down
```

### API 端点

| 服务 | 端口 | 说明 |
|------|------|------|
| Gateway | 8080 | API 网关 |
| User | 8081 | 用户服务 |
| RBAC | 8082 | 权限服务 |
| Menu | 8083 | 菜单服务 |
| Log | 8084 | 日志服务 |
| Dict | 8085 | 字典服务 |

## API 示例

### 用户登录
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}'
```

### 获取用户列表
```bash
curl -X GET http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer <token>"
```

### 获取字典数据
```bash
curl -X GET http://localhost:8080/api/v1/dicts/sys_user_status
```

## SSQL 查询语法

SSQL 是一种简化的查询语法，用于构建复杂的 SQL 条件。

### 基本语法

```
字段名 操作符 值
```

### 支持的操作符

| 操作符 | 说明 | 示例 |
|--------|------|------|
| = | 等于 | `status = 1` |
| != | 不等于 | `status != 0` |
| > | 大于 | `age > 18` |
| >= | 大于等于 | `age >= 18` |
| < | 小于 | `age < 60` |
| <= | 小于等于 | `age <= 60` |
| ~ | 模糊匹配 | `name ~ "张"` |
| ?= | 包含 | `tags ?= ["a", "b"]` |
| ?!= | 不包含 | `tags ?!= ["c"]` |
| ?null | 是否为空 | `deleted_at ?null true` |
| >< | 区间 | `age >< [18, 60]` |

### 逻辑操作符

| 操作符 | 说明 |
|--------|------|
| && | 且 |
| \|\| | 或 |
| () | 分组 |

### 示例

```
# 简单查询
status = 1

# 组合查询
status = 1 && role_id = 2

# 复杂嵌套
(status = 1 && role_id = 2) || (status = 0 && role_id = 3)

# 模糊匹配
name ~ "张" && status = 1
```

## 配置示例

```yaml
app:
  name: goback
  env: dev

server:
  host: 0.0.0.0
  port: 8080
  readTimeout: 60
  writeTimeout: 60

database:
  driver: mysql
  host: localhost
  port: 3306
  database: goback
  username: root
  password: root
  maxIdleConns: 10
  maxOpenConns: 100
  connMaxLifetime: 3600

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

etcd:
  endpoints:
    - localhost:2379

jwt:
  secret: your-secret-key
  expire: 7200
  issuer: goback

casbin:
  modelPath: configs/rbac_model.conf

log:
  level: info
  format: json
  output: both
  filename: logs/app.log
  maxSize: 100
  maxBackups: 7
  maxAge: 30
  compress: true
```

## License

MIT License

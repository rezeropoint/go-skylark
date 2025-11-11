# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 快速参考

### 最常用命令
```bash
go build ./...                          # 构建所有模块
go test ./...                           # 运行所有测试
go test ./internal/query -v             # 测试特定包(详细输出)
go test -run TestQueryBuilder ./...    # 运行特定测试
go fmt ./...                            # 格式化代码
go vet ./...                            # 静态分析
go mod tidy                             # 整理依赖
```

### 核心架构原则(必须遵守)
1. **依赖方向**: Engine → Internal → Core (禁止反向依赖)
2. **Core 层纯净**: 禁止 `db` 标签、`sql.Null*` 等框架类型
3. **Manager 解耦**: 通过 Core 层函数类型注入,避免循环依赖
4. **数据流向**: `数据库 → model.go(DataModel) → core/(DomainModel) → 用户`

### 常见陷阱
- ❌ 在 Core 层使用 `sql.NullString`
- ❌ Manager 之间直接相互引用
- ❌ 在 Config 中包含具体实现类型(如 `*redis.Client`)
- ❌ 忘记在 model.go 使用 `sql.Null*` 处理可空字段
- ❌ 不使用预定义错误(`core/var.go`)

## 项目简介

go-skylark 是一个用于对接 Skylark 低代码平台的 Go SDK，提供流程管理、事件查询、统计分析等功能。采用三层架构（Engine → Internal → Core），支持多租户、远程数据库连接池、组织权限过滤等企业级特性。

## 核心架构

### 三层架构与依赖流向

```
Engine Layer (engine/)          ← 对外接口，聚合所有 Manager
    ↓ 依赖
Internal Layer (internal/*)     ← Manager 模式，业务逻辑实现
    ↓ 依赖
Core Layer (core/)              ← 领域模型，纯 Go 类型，无框架依赖
```

**关键原则**：
- Core 层不依赖任何框架（无 `db` 标签、无 `sql.Null*`）
- Internal 层通过 `model.go` 处理数据库类型，通过 `ToDomain()` 转换为 Core 类型
- Manager 之间通过 Core 层的**函数类型注入**解耦（避免循环依赖）

### 关键设计模式

#### 1. 函数注入解耦模式

**问题**：QueryManager 需要 PlatformManager 提供远程连接，但不能直接依赖

**解决方案**：
```go
// core/platform.go - 定义函数签名
type GetRemoteDBFunc func(ctx context.Context, tenantID string) (sqlx.SqlConn, error)

// internal/query/query.go - 接收函数
type queryManager struct {
    getRemoteDB core.GetRemoteDBFunc  // 函数类型，不依赖具体 Manager
}

// engine/handler.go - 注入实现
queryMgr := query.NewManager(
    platformMgr.GetRemoteDB,  // 传入方法引用
)
```

#### 2. 领域模型与数据模型分离

**数据流向**：`数据库 → DataModel (model.go) → DomainModel (core/) → 用户`

```go
// internal/event/model.go - 数据模型（允许框架类型）
type EventConfigModel struct {
    OrgFieldName sql.NullString `db:"org_field_name"`  // ✅ 框架类型
}
func (m *EventConfigModel) ToDomain() *core.EventConfig {
    return &core.EventConfig{
        OrgFieldName: convertNullString(m.OrgFieldName),  // 转换为指针
    }
}

// core/event.go - 领域模型（纯 Go 类型）
type EventConfig struct {
    OrgFieldName *string  // ✅ 使用指针表示可空
}
```

#### 3. 聚合管理模式（DDD）

事件配置和字段配置作为**聚合根**一起管理：
- 创建/更新使用**数据库事务**保证原子性
- 更新字段时采用**完整替换策略**（先删除全部，再插入）
- 删除事件时字段**级联删除**（ON DELETE CASCADE）

## 核心模块职责

### Engine 层（engine/）

**职责**：对外统一接口，管理所有 Manager 生命周期

**初始化顺序**（`engine/handler.go:32-115`）：
```
1. Cache → 2. Flows/Forms → 3. Platform → 4. Mapping → 5. Event → 6. Query/Stats
```

**接口分类**（15个方法）：
- 旧模块：CreateFlow、CreateFormRow、UpdateFlowJourneyStatus
- 平台配置：Create/Get/Update/Delete/ValidatePlatformConfig
- 事件配置：Create/Update/Get/List/DeleteEventWithFields
- 组织映射：Create/Get/List/Update/DeleteOrgMapping
- 远程查询：QueryEventData、GetEventDetail、GetFlowList、GetFlowFields
- 统计分析：GetDurationStats、GetStatusStats、GetTrendStats、GetNodeStats、GetUserStats、GetOrgStats、GetPendingStats

### Internal 层关键模块

#### internal/platform（平台管理）
- 管理远程 Skylark 平台配置（PostgreSQL 存储）
- **管理远程数据库连接池**（`map[tenantID]sqlx.SqlConn`，并发安全）
- 提供 `GetRemoteDB()` 给 Query/Stats 使用

#### internal/event（事件配置）
- 聚合管理事件配置+字段配置（事务保证原子性）
- 验证远程 flow_id 存在性
- 提供字段配置给 Query/Stats 使用（通过函数注入）

#### internal/mapping（组织映射）
- 管理组织映射：`(TenantID + RemoteOrgValue) → LocalOrgID`
- 创建时验证 local_org_id 存在于 organizations 表
- 删除时检查是否被事件配置使用
- 提供映射列表给 Query/Stats 使用（通过函数注入）

#### internal/query（远程查询引擎）
- 构建 Skylark PostgreSQL 查询（`assignments_{flow_id}` 表）
- **Journey 聚合**：使用 `DISTINCT ON (slp_journey_id)` 获取最新 Assignment
- **组织权限过滤**：通过 OrgMapping 计算用户可见的远程组织值，构建 `WHERE org_field = ANY($1)` 条件
- **虚拟状态支持**：`pending`（只有1个节点）、`processing`（多个节点）
- **用户名批量转换**：Redis 批量查询远程 users 表（TTL 24h）

#### internal/stats（统计分析）
- 提供 7 种统计维度（时长、状态、趋势、节点、用户、组织、待处理）
- **多事件ID聚合**：支持 `EventConfigIDs` 数组，合并多个事件的统计结果
- 缓存统计结果（Redis，TTL 5分钟）

#### internal/flows & internal/forms（旧模块，仍在使用）
- 负责**写操作**（调用 Skylark REST API）
- 处理字段类型转换（Base64 → 七牛云 URL）
- 与新模块分工：旧模块写，新模块读

### Core 层（core/）

**领域模型**：
- `EventConfig`/`FieldConfig`：事件和字段配置
- `OrgMapping`：组织映射
- `QueryRequest`/`StatsRequest`：查询和统计请求
- `TypedValue`：带类型的值（支持 string、imageURL、imageBase64）

**函数类型**（用于依赖注入）：
- `GetRemoteDBFunc`：获取远程数据库连接
- `GetEventConfigWithFieldsFunc`：获取事件配置
- `ListOrgMappingsFunc`：获取组织映射列表

**错误定义**：36 个预定义错误（`core/var.go`）

## 测试规范

### 测试文件组织
- 测试文件命名: `*_test.go`
- 单元测试: `Test<FunctionName>`
- 基准测试: `Benchmark<FunctionName>`
- 示例测试: `Example<FunctionName>`

### 测试覆盖率要求
- 新增代码: 核心逻辑 ≥ 80%
- 边界情况: 必须覆盖错误处理路径
- Mock 使用: 使用 `gomock` 或接口注入方式

### 测试示例
```go
// ✅ 正确：测试使用接口注入 Mock
func TestQueryManager_Query(t *testing.T) {
    mockDB := sqlmock.New()
    mockCache := &MockCacheInterface{}

    mgr := query.NewManager(query.Config{
        Cache: mockCache,
        GetRemoteDB: func(tenantID string) (sqlx.SqlConn, error) {
            return mockDB, nil
        },
    })

    // 测试逻辑...
}
```

## 常用开发命令

### 构建和测试
```bash
# 构建所有模块
go build ./...

# 运行所有测试
go test ./...

# 运行特定包的测试（带详细输出）
go test ./internal/query -v

# 运行单个测试
go test -run TestQueryBuilder ./internal/query

# 测试覆盖率
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 基准测试
go test -bench=. ./internal/query

# 格式化代码
go fmt ./...

# 静态分析
go vet ./...
```

### 模块管理
```bash
# 整理依赖
go mod tidy

# 验证依赖
go mod verify

# 更新依赖
go get -u ./...

# 检查循环依赖
go mod graph | grep 'go-skylark'
```

## Manager 开发规范

### 文件组织（所有 internal/* 必须遵守）

| 文件 | 职责 | 必需性 |
|------|-----|--------|
| `<manager>.go` | 接口定义 + NewManager() 构造函数 | ✅ 必需 |
| `handler.go` | 接口实现 | ✅ 必需 |
| `model.go` | 数据库模型 + ToDomain() 转换方法 | ✅ 必需 |
| `helpers.go` | 辅助函数（convertNullString 等） | 🟡 推荐 |
| `config.go` | Manager 配置结构 | ✅ 必需 |

### model.go 规范

**数据模型 → 领域模型转换** (详见 `DEVELOPMENT.md`)：
```go
// internal/*/model.go - 允许框架类型
type EventConfigModel struct {
    OrgFieldName sql.NullString `db:"org_field_name"`  // ✅ 可空字段
}
func (m *EventConfigModel) ToDomain() *core.EventConfig {
    return &core.EventConfig{
        OrgFieldName: convertNullString(m.OrgFieldName),  // 转为 *string
    }
}

// core/*.go - 纯 Go 类型
type EventConfig struct {
    OrgFieldName *string  // ✅ 使用指针表示可空
}
```

### 关键规范

**数据库查询**：
- 使用 `go-zero sqlx`: `conn.QueryRow(&model, query, args...)`
- 数组类型使用 `pq.Array`: `Tags pq.StringArray`

**错误处理**：
- 使用预定义错误 (`core/var.go`): `return core.ErrEventConfigNotFound`
- logx 使用 `Error`，禁止使用 `Warn`

## 重要约定

### Manager 依赖注入 (避免循环依赖)

✅ **正确**: Config 只包含配置参数，依赖通过 NewManager 参数传入
```go
type Config struct {
    MaxPageSize int // ✅ 只有配置参数
}
func NewManager(config Config, db sqlx.SqlConn, cache core.CacheInterface) (Manager, error)
```

❌ **错误**: 在 Config 中包含依赖
```go
type Config struct {
    Cache core.CacheInterface // ❌ 接口
    DB    sqlx.SqlConn        // ❌ 运行时实例
}
```

### 字段类型处理

| 字段后缀 | 类型 | 处理方式 |
|---------|------|----------|
| `_Img` | 图片 URL | 直接传递 |
| `_Base64Img` | Base64 图片 | 上传七牛云 → URL |

**选项字段**: RadioButton、Checkbox、SelectField、MultipleSelectField
**判断函数**: `core.IsOptionField(fieldType)`

### Skylark 远程表结构

```sql
-- 动态表名：assignments_{flow_id}
assignments_123
├─ slp_assignment_id    # 主键
├─ slp_journey_id       # 流程实例ID（聚合键）
├─ slp_status           # 状态
├─ slp_vertex_id        # 节点ID
├─ slp_user_id          # 处理人ID
└─ 业务字段...

-- 节点表
vertices
├─ id
├─ name                 # 节点名
└─ alias_name           # 节点别名

-- 用户表
users
├─ id
└─ name                 # 用户姓名
```

### 缓存键设计（所有键定义在 core/*.go）

| 数据类型 | 缓存键 | TTL |
|---------|--------|-----|
| Flow列表 | `skylark:flows:{tenant}:{namespace}` | 1h |
| Flow字段 | `skylark:flow_fields:{tenant}:{flow_id}` | 1h |
| 用户名 | `skylark:users:{tenant}:{user_id}` | 24h |
| 组织映射 | `skylark:mapping:{id}` | 30天 |
| 统计结果 | `skylark:stats:{type}:{tenant}:{event}:{hash}` | 5分钟 |

## 开发技巧

### 调试特定功能
```bash
# 查看函数调用链
go build -gcflags="-m" ./internal/query

# 查看接口实现
grep -r "implements.*Interface" ./internal

# 查找 TODO 标记
grep -r "TODO\|FIXME" ./
```

### 代码质量检查
```bash
# 查找 Core 层的框架依赖（应该为空）
grep -r 'db:"' ./core
grep -r 'sql\.Null' ./core

# 检查循环依赖
go list -f '{{.ImportPath}} {{.Imports}}' ./... | grep -E "(query.*platform|platform.*query)"
```

### 性能分析
```bash
# CPU 分析
go test -cpuprofile=cpu.prof -bench=. ./internal/query
go tool pprof cpu.prof

# 内存分析
go test -memprofile=mem.prof -bench=. ./internal/query
go tool pprof mem.prof
```

## 常见问题

### Q: 如何添加新的 Manager?
1. 在 `internal/<manager>/` 创建目录
2. 创建 `<manager>.go`, `handler.go`, `model.go`, `config.go`
3. 在 `engine/handler.go` 初始化 Manager
4. 参考 `internal/event/` 或 `internal/mapping/` 实现

### Q: Manager 之间如何通信?
- 使用 Core 层函数类型注入 (如 `GetRemoteDBFunc`)
- 通过接口注入 (如 `CacheInterface`)
- 禁止直接依赖具体 Manager

### Q: 如何处理可空字段?
- Internal 层 model.go: 使用 `sql.NullString`
- Core 层: 使用指针 `*string`
- 转换函数: `helpers.go` 中的 `convertNullString()`

## 参考文档

- **DEVELOPMENT.md**：详细的架构设计、开发规范、代码示例
- **README.md**：项目介绍、快速开始
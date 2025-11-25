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
- ❌ 在 Config 中包含具体实现类型(如 `*redis.Client`)或函数类型
- ❌ 忘记在 model.go 使用 `sql.Null*` 处理可空字段
- ❌ 不使用预定义错误(`core/errors.go`)
- ❌ Flows 模块依赖注入不完整（需要 getPlatformConfig、getRemoteUserIDs、getRemoteDB）
- ❌ **将系统管理功能暴露在 Engine 层**（组织/用户管理应在 Admin 层）
- ❌ **将业务流程功能暴露在 Admin 层**（流程/查询/统计应在 Engine 层）
- ❌ **混淆 slp_status 和 slp_category**（状态判断必须结合 category 字段）
- ❌ **使用不存在的状态值**（如 `completed`/`rejected`，应使用 `finished`/`aborted`）
- ❌ **基于节点数判断虚拟状态**（应基于是否有 `category='processed' AND status='approved'`）

## 项目简介

go-skylark 是一个用于对接 Skylark 低代码平台的 Go SDK，提供流程管理、事件查询、统计分析等功能。采用三层架构（Engine → Internal → Core），支持多租户、远程数据库连接池、组织权限过滤等企业级特性。

## 核心架构

### 三层架构与依赖流向

```
对外接口层 (两个入口)
├─ Admin Layer (admin/)         ← 系统管理引擎（组织/用户管理、ID映射）
└─ Engine Layer (engine/)       ← 业务流程引擎（流程管理、查询统计）
    ↓ 依赖
Internal Layer (internal/*)     ← Manager 模式，业务逻辑实现
    ↓ 依赖
Core Layer (core/)              ← 领域模型，纯 Go 类型，无框架依赖
```

**关键原则**：
- **职责分离**：Admin 负责系统管理，Engine 负责业务流程，两者独立初始化
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

### 对外接口层职责分离

SDK 提供两个对外入口，职责明确分离：

#### Admin 层（admin/）- 系统管理引擎

**职责**：系统管理功能（组织管理、用户管理、ID映射管理）

**使用场景**：系统管理微服务使用，用于管理租户、组织、用户等基础数据

**接口分类**（14个方法）：
- **组织管理**（5个）：CreateOrganization、CreateSubOrganization、DeleteOrganization、UpdateOrganization、GetOrgSyncStatus
- **用户管理**（2个）：CreateUser、GetUserSyncStatus
- **组织ID映射管理**（2个）：BindOrganization（绑定已存在的远程组织）、UnbindOrganization（解绑组织映射）
- **用户ID映射管理**（2个）：BindUser（绑定已存在的远程用户）、UnbindUser（解绑用户映射）
- **组织成员管理**（2个）：AddMember（添加单个成员，使用本地用户ID）、RemoveMember（移除单个成员，使用本地用户ID）
- **平台配置**（1个）：通过 internal/platform 访问（不直接暴露）

**关键特性**：
- 所有操作自动维护本地ID与远程ID的双向映射
- 映射关系存储在本地数据库（skylark_org_mappings、skylark_user_mappings）
- 支持缓存优化（Redis，TTL 30天）
- 绑定/解绑时会验证远程资源存在性（调用 Skylark API）

#### Engine 层（engine/）- 业务流程引擎

**职责**：业务流程功能（流程管理、事件查询、统计分析）

**使用场景**：业务微服务使用，用于创建流程、查询事件数据、统计分析等

**初始化顺序**（`engine/handler.go:36-140`）：
```
1. Cache → 2. Platform → 3. Organization → 4. User → 5. Flows/Forms → 6. Mapping → 7. Event → 8. Query/Stats
```

**依赖关系**：
- Organization、User 依赖 Platform（获取 API 配置）
- Flows 依赖 Platform（获取 API 配置 + 远程 DB 连接）和 User（获取远程用户 ID）
- Event 依赖 Platform（获取远程 DB 连接，验证远程 flow_id）
- Query、Stats 依赖 Platform（获取远程 DB 连接）、Event（获取事件配置）、Mapping（获取组织映射）

**接口分类**（37个方法）：
- **流程管理**（13个）：CreateFlow、UpdateFlowJourneyStatus、GetFlowJourneyBySN、GetFlowJourneyAssignments、GetFlowJourneyDetail、GetFlowDetail、GetUserAssignments、GetProposedJourneys、SearchJourneys、GetJourneyMoments、GetCurrentProcessingUsers、AbortJourney、CreateFormRow
- **平台配置**（5个）：CreatePlatformConfig、GetPlatformConfig、UpdatePlatformConfig、DeletePlatformConfig、ValidatePlatformConfig
- **事件配置**（5个）：CreateEventWithFields、UpdateEventWithFields、GetEventWithFields、ListEventWithFields、DeleteEvent
- **组织映射**（5个）：CreateOrgMapping、GetOrgMapping、ListOrgMappings、UpdateOrgMapping、DeleteOrgMapping
- **远程查询**（4个）：QueryEventData、GetEventDetail、GetFlowList、GetFlowFields
- **统计分析**（7个）：GetDurationStats、GetStatusStats、GetTrendStats、GetNodeStats、GetUserStats、GetOrgStats、GetPendingStats

**关键特性**：
- 所有业务操作透明使用本地ID，SDK内部自动转换为远程ID
- 用户无需关心ID映射细节，只需确保先通过 AdminEngine 完成初始化

### Internal 层关键模块

#### internal/platform（平台管理）
- 管理远程 Skylark 平台配置（PostgreSQL 存储）
- **管理远程数据库连接池**（`map[tenantID]sqlx.SqlConn`，并发安全）
- 提供 `GetRemoteDB()` 给 Query/Stats/Flows 使用
- 提供 `GetAPIConfig()` 给 Organization/User/Flows 使用

#### internal/organization（组织管理）
- 管理本地组织 ID 与远程组织 ID 的映射关系
- 查询远程 Skylark 数据库的 organizations 表
- 支持批量转换：`[]localOrgID → []remoteOrgID`
- 提供组织信息缓存（Redis，TTL 可配置）
- **绑定/解绑功能**：
  - `BindOrganization`：绑定已存在的远程组织（验证远程资源存在性）
  - `UnbindOrganization`：解绑组织映射（只删除本地映射，不删除远程资源）

#### internal/user（用户管理）
- 管理本地用户 ID 与远程用户 ID 的映射关系
- 查询远程 Skylark 数据库的 users 表
- **批量查询用户名**：Redis 缓存（TTL 24h）
- 支持批量转换：`[]localUserID → []remoteUserID`
- **绑定/解绑功能**：
  - `BindUser`：绑定已存在的远程用户（验证远程资源存在性）
  - `UnbindUser`：解绑用户映射（只删除本地映射，不删除远程用户）

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
- **虚拟状态支持**：
  - `pending`（待处理）：流程未结束 + 无 `category='processed' AND status='approved'` 的节点
  - `processing`（处理中）：流程未结束 + 有 `category='processed' AND status='approved'` 的节点
- **用户名批量转换**：Redis 批量查询远程 users 表（TTL 24h）

#### internal/stats（统计分析）
- 提供 7 种统计维度（时长、状态、趋势、节点、用户、组织、待处理）
- **多事件ID聚合**：支持 `EventConfigIDs` 数组，合并多个事件的统计结果
- 缓存统计结果（Redis，TTL 5分钟）

#### internal/flows（流程管理）
- **完整的流程生命周期管理**（13个接口）
  - 写操作：创建流程、创建表单、更新状态、终止流程（CreateFlow、CreateFormRow、UpdateFlowJourneyStatus、AbortJourney）
  - 读操作：查询流程、获取详情、搜索、审批历史、当前处理人等（9个查询接口）
- **性能优化机制**（enrichment.go）：
  - 问题：GetUserAssignments 返回的 assignment 缺少 flow_id 和 flow_title
  - 方案：映射库 + 批量查询 + Redis 缓存三层优化
  - 效果：API 调用减少 95%+，耗时从 2.8 秒降至 60-360ms
- **依赖注入**：getRemoteDB 函数（查询远程数据库 journeys 表）
- **配置化支持**：FlowInfoCacheTTL（Flow 信息缓存 TTL，默认 3600 秒）
- 详细文档：`internal/flows/README.md`

#### internal/forms（表单管理）
- 负责**写操作**（调用 Skylark REST API）
- 处理字段类型转换（Base64 → 七牛云 URL）

### Core 层（core/）

**领域模型**：
- `EventConfig`/`FieldConfig`：事件和字段配置
- `OrgMapping`：组织映射
- `QueryRequest`/`StatsRequest`：查询和统计请求
- `TypedValue`：带类型的值（支持 string、imageURL、imageBase64）

**函数类型**（用于依赖注入）：
- `GetRemoteDBFunc`：获取远程数据库连接（query/stats/flows 使用）
- `GetPlatformConfigFunc`：获取平台 API 配置（flows 使用）
- `GetEventConfigWithFieldsFunc`：获取事件配置
- `ListOrgMappingsFunc`：获取组织映射列表
- `GetRemoteUserIDsFunc`：获取远程用户 ID（flows 使用）

**错误定义**：36 个预定义错误（`core/var.go`）

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
| `handler.go` | 接口实现（导出方法，实现接口） | ✅ 必需 |
| `model.go` | 数据库模型（包含 db 标签和 ToDomain 方法） | ✅ 必需 |
| `helpers.go` | 普通函数（辅助方法，如 convertNullString） | 🟡 推荐 |
| `internal.go` | 未导出的方法（私有方法实现） | 🟡 推荐 |
| `sql.go` | 数据库表初始化 SQL（仅包含建表语句） | 🟡 可选 |
| `config.go` | Manager 配置结构体 | ✅ 必需 |

**重要约定**：
- `model.go` 只包含结构体定义和导出方法（如 `ToDomain()`）
- `helpers.go` 包含辅助方法（普通函数，如类型转换函数）
- `internal.go` 包含未导出的方法（小写开头的私有方法）
- `sql.go` 只包含数据库表初始化 SQL，查询语句应内联在代码中

### model.go 规范

**数据模型 → 领域模型转换**：
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

**辅助函数** (定义在 helpers.go):
```go
func convertNullString(ns sql.NullString) *string {
    if ns.Valid {
        return &ns.String
    }
    return nil
}

func convertNullTime(nt sql.NullTime) *time.Time {
    if nt.Valid {
        return &nt.Time
    }
    return nil
}
```

### 关键规范

**数据库查询**：
- 使用 `go-zero sqlx`: `conn.QueryRow(&model, query, args...)`
- 数组类型使用 `pq.Array`: `Tags pq.StringArray`

**错误处理**：
- 使用预定义错误 (`core/var.go`): `return core.ErrEventConfigNotFound`
- logx 使用 `Error`，禁止使用 `Warn`

## 核心设计理念

### Core 层黄金法则

#### ✅ Core 层应该是什么

1. **纯粹的领域模型**
   - 代表业务概念: `TypedValue`、`FieldMapping`、`QueryCondition`
   - 包含业务规则: `IsOptionField()`、`IsImageField()`
   - 独立于技术实现

2. **稳定的抽象**
   - 变化频率低（业务概念变化缓慢）
   - 被多个模块依赖
   - 定义清晰的接口（如 `CacheInterface`）

3. **框架无关**
   - 只使用 Go 标准库类型
   - 不依赖特定框架（Redis、ORM、HTTP）

#### ❌ Core 层不应该是什么

1. **不是数据传输对象 (DTO)**
   - 不包含 JSON 序列化逻辑
   - 不为 API 响应格式设计
   - ❌ 禁止定义 Request/Response 结构（例外：跨多层传递的 DTO）

2. **不是 ORM 模型**
   - **禁止出现**: `db:"field_name"` 标签
   - **禁止出现**: `bson:"field_name"` 标签
   - **禁止出现**: `gorm:` 标签

3. **不是数据库查询结果**
   - 不为数据库扫描而设计
   - 不包含 `sql.NullString`、`sql.NullTime` 等框架类型

## 重要约定

### Manager 依赖注入 (避免循环依赖)

✅ **正确**: Config 只包含配置参数，依赖通过 NewManager 参数传入
```go
// 示例 1：Query Manager
type Config struct {
    MaxPageSize int // ✅ 只有配置参数
}
func NewManager(config Config, db sqlx.SqlConn, cache core.CacheInterface) (Manager, error)

// 示例 2：Flows Manager（包含缓存配置）
type Config struct {
    FlowInfoCacheTTL int // ✅ 缓存 TTL 配置（秒）
}
func NewManager(config *Config, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc, getRemoteUserIDs core.GetRemoteUserIDsFunc, getRemoteDB core.GetRemoteDBFunc) (Manager, error)
```

❌ **错误**: 在 Config 中包含依赖
```go
type Config struct {
    Cache       core.CacheInterface // ❌ 接口类型
    GetRemoteDB core.GetRemoteDBFunc // ❌ 函数类型
    DB          sqlx.SqlConn        // ❌ 运行时实例
}
```

**原因**: Config 结构体应该只存储**静态配置参数**，所有依赖（接口、函数、运行时实例）都应该通过 **NewManager 函数参数**传入。

**优势**: Config 职责单一、依赖显式化、易于测试、避免循环依赖

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
├─ slp_category         # 类别（proposed/processed/cc）⭐ 关键字段
├─ slp_status           # 状态（processing/finished/aborted/approved/refused 等）⭐ 关键字段
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

**关键字段说明**：
- **`slp_category`**（类别）：
  - `proposed`：发起任务，代表**整个流程的状态**
  - `processed`：处理任务，代表**单个节点的状态**
  - `cc`：抄送任务
- **`slp_status`**（状态）：
  - 当 `category='proposed'` 时：`processing`（进行中）、`finished`（已完成）、`aborted`（已终止）
  - 当 `category='processed'` 时：`processing`（处理中）、`approved`（已同意）、`refused`（已拒绝）、`transferred`（已转交）等

### Skylark 数据模型与状态常量

#### 数据模型核心理解

```
一个 Journey（流程实例）包含多个 Assignment（任务记录）：

┌─ Assignment（发起节点）
│  ├─ slp_category: 'proposed'   ← 代表整个流程的状态
│  └─ slp_status: 'processing' | 'finished' | 'aborted'
│
├─ Assignment（审批/处理节点）
│  ├─ slp_category: 'processed'  ← 只代表单个节点的状态
│  └─ slp_status: 'approved' | 'refused' | 'transferred' | 'processing' | ...
│
└─ ...（更多 processed 节点）
```

#### 状态常量定义

**Category 常量**（`core/assignment.go`）：
```go
AssignmentCategoryProposed  = "proposed"   // 发起任务
AssignmentCategoryProcessed = "processed"  // 处理任务
AssignmentCategoryCC        = "cc"         // 抄送任务
```

**流程状态常量**（`core/journey.go`，用于 Journey 和 `category='proposed'` 的 Assignment）：
```go
StatusProcessing = "processing"  // 流程进行中
StatusFinished   = "finished"    // 流程已完成
StatusAborted    = "aborted"     // 流程已终止
```

**节点状态常量**（`core/assignment.go`，仅用于 `category='processed'`）：
```go
AssignmentStatusApproved     = "approved"      // 审批同意
AssignmentStatusRefused      = "refused"       // 审批拒绝
AssignmentStatusTransferred  = "transferred"   // 转交
AssignmentStatusWithdrawn    = "withdrawn"     // 撤回
AssignmentStatusResubmitted  = "resubmitted"   // 重新提交
AssignmentStatusAutoApproved = "auto_approved" // 自动审批通过
```

#### 虚拟状态判断规则

**pending（待处理）**：
- 条件：`category='proposed'` 的 `status NOT IN ('finished', 'aborted')`
- 且：不存在 `category='processed' AND status='approved'` 的 assignment

**processing（处理中）**：
- 条件：`category='proposed'` 的 `status NOT IN ('finished', 'aborted')`
- 且：存在至少一个 `category='processed' AND status='approved'` 的 assignment

**关键原则**：
- ❌ **错误**：基于节点数判断（`COUNT(DISTINCT slp_vertex_id)`）
- ✅ **正确**：基于是否有审批通过的节点（`category='processed' AND status='approved'`）

### 缓存键设计（所有键定义在 core/*.go）

| 数据类型 | 缓存键 | TTL | 说明 |
|---------|--------|-----|------|
| Flow 列表 | `skylark:flows:{tenant}:{namespace}` | 2 分钟 | Query 模块使用 |
| Flow 字段 | `skylark:flow_fields:{tenant}:{flow_id}` | 2 分钟 | Query 模块使用 |
| **Flow 信息（API）** | `skylark:flow:api:{tenant}:{flow_id}` | **可配置**（默认 1h） | **Flows enrichment 使用** |
| 用户名 | `skylark:users:{tenant}:{user_id}` | 24h | User 模块使用 |
| 组织映射 | `skylark:mapping:{id}` | 30 天 | Mapping 模块使用 |
| 统计结果 | `skylark:stats:{type}:{tenant}:{event}:{hash}` | 5 分钟 | Stats 模块使用 |

**注意**：Flow 信息（API）的 TTL 通过 `flows.Config.FlowInfoCacheTTL` 配置（单位：秒）

## 开发技巧

### 调试特定功能
```bash
# 查看函数调用链
go build -gcflags="-m" ./internal/query

# 查看接口实现
grep -r "implements.*Interface" ./internal

# 查找 TODO 标记
grep -r "TODO\|FIXME" ./

# 快速定位某个功能的实现
grep -r "func.*QueryEventData" ./

# 检查某个 Manager 的所有方法
grep -r "^func.*Manager" ./internal/query/
```

### 性能优化技巧

#### Flows 模块性能优化机制（enrichment）

**问题**：GetUserAssignments 返回的 assignment 列表缺少 flow_id 和 flow_title

**方案**（`internal/flows/enrichment.go`）：
```
1. 提取唯一的 journey_id → 批量查询 journey_id → flow_id 映射（远程数据库）
2. 提取唯一的 flow_id → 批量查询 flow 信息（Redis 缓存优先）
3. 合并数据，填充 assignment.FlowID 和 assignment.FlowTitle
```

**效果**：
- API 调用：从 54 次降至 1-4 次（减少 95%+）
- 总耗时：缓存命中时约 60ms，未命中时约 360ms（优化前 2.8 秒）

**配置**：
```go
// internal/flows/config.go
type Config struct {
    FlowInfoCacheTTL int // 默认 3600 秒（1 小时）
}
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

## 性能优化设计模式

### 批量查询优化模式

**适用场景**: 需要从远程数据库或 API 获取多个关联对象的信息

**问题**: N+1 查询问题导致性能瓶颈

**解决方案**:
```go
// ❌ 错误：N+1 查询
for _, item := range items {
    relatedData, _ := fetchRelatedData(item.ID)  // N 次查询
}

// ✅ 正确：批量查询（4 步法）
ids := extractUniqueIDs(items)                              // 1. 提取唯一 ID
results, _ := db.Query("... WHERE id = ANY($1)", pq.Array(ids))  // 2. 批量查询
mapping := buildMapping(results)                            // 3. 构建映射
fillData(items, mapping)                                    // 4. 填充数据
```

**关键原则**:
- 使用 `pq.Array` 进行批量 SQL 查询（PostgreSQL）
- 容错处理：部分失败不影响整体流程

### 多级缓存策略

**适用场景**: 频繁访问的关联数据（如 Flow 信息、用户信息）

**缓存层级**:
```
1. Redis 缓存（优先） → 2. 批量 API 调用 → 3. 异步回写缓存
```

**实现规范**:
```go
func batchGetData(ctx context.Context, ids []int64) (map[int64]*Data, error) {
    result, missedIDs := make(map[int64]*Data), []int64{}

    // Level 1: 批量查询 Redis
    for _, id := range ids {
        if data, err := cache.Get(ctx, id); err == nil {
            result[id] = data
        } else {
            missedIDs = append(missedIDs, id)
        }
    }

    // Level 2: 批量 API 调用（只查询未命中的）
    for _, id := range missedIDs {
        if data, err := fetchFromAPI(ctx, id); err == nil {
            result[id] = data
            go cache.Set(context.Background(), id, data, ttl) // Level 3: 异步回写
        }
    }

    return result, nil
}
```

**关键原则**:
- TTL 通过 Config 配置（如 `FlowInfoCacheTTL`）
- 异步回写使用 `context.Background()`（避免主请求取消影响缓存）

### 数据补充（Enrichment）模式

**适用场景**: API 返回的数据缺少关联信息，需要自动补充

**解决方案**:
```go
func GetList(ctx context.Context, req *Request) ([]*Item, error) {
    items, err := fetchItemsFromAPI(ctx, req)  // 1. 获取主数据
    if err != nil {
        return nil, err
    }

    // 2. 自动补充关联信息（失败不影响主流程）
    if err := enrichItems(ctx, items); err != nil {
        logx.Errorf("补充数据失败: %v", err)  // 只记录日志
    }

    return items, nil  // 3. 返回数据（可能包含补充字段）
}

func enrichItems(ctx context.Context, items []*Item) error {
    ids := extractUniqueIDs(items)             // 提取唯一 ID
    dataMap, _ := batchGetData(ctx, ids)       // 批量查询（应用多级缓存）
    for _, item := range items {               // 填充可选字段（指针类型）
        if data, ok := dataMap[item.ID]; ok {
            item.RelatedData = data
        }
    }
    return nil
}
```

**关键原则**:
- Enrichment 是**可选的**（失败不影响主流程）
- 补充字段使用**指针类型**（如 `*string`）
- 用户无需关心 enrichment 细节（SDK 内部自动处理）

**示例参考**: `internal/flows/enrichment.go`

### 性能优化清单

添加新接口时，检查是否需要性能优化：

- [ ] 是否存在 N+1 查询？→ 使用批量查询
- [ ] 是否频繁访问相同数据？→ 添加缓存
- [ ] 是否需要关联数据？→ 考虑 enrichment 模式
- [ ] 缓存 TTL 是否可配置？→ 添加到 Config
- [ ] 是否有容错机制？→ 优化失败不影响主流程
- [ ] 是否记录性能指标？→ 添加日志（可选）

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

### Q: Flows 模块的性能优化如何工作?
- **触发时机**：调用 `GetUserAssignments` 或 `GetProposedJourneys` 时自动触发
- **优化流程**：
  1. 提取 journey_id → 批量查询远程数据库获取 flow_id
  2. 提取 flow_id → 批量查询 Redis 缓存获取 flow 信息
  3. 未命中缓存的 flow_id → 调用 API 获取并异步回写缓存
- **配置项**：`FlowInfoCacheTTL`（默认 3600 秒）
- **容错处理**：优化失败不影响主流程，只记录日志

## 参考文档

- **README.md**：项目介绍、快速开始、安装说明
- **internal/flows/README.md**：流程模块详细文档（性能优化机制详解）
- **internal/cache/README.md**：缓存模块详细文档
- **internal/stats/MULTI_EVENT_STATS.md**：多事件统计设计文档
- **Skylark流程API文档.md**、**Skylark组织API文档.md**、**Skylark用户API文档.md**：Skylark 平台 API 参考
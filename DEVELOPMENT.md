# DEVELOPMENT.md

本文档面向 go-skylark SDK 的内部开发者，阐述架构设计理念、开发规范和最佳实践。

---

## 📐 架构设计理念

### 三层架构与依赖规则

```
┌─────────────────────────────────────────────┐
│  Engine Layer (对外接口)                    │  ← 用户直接调用
│  - engine/engine.go: SkylarkEngine 接口     │
│  - engine/handler.go: 接口实现              │
├─────────────────────────────────────────────┤
│  Internal Layer (内部实现)                  │  ← Manager 模式
│  - internal/platform/: 平台配置管理         │
│  - internal/flows/: 流程管理                │
│  - internal/forms/: 表单管理                │
│  - internal/query/: 查询管理                │
│  - internal/stats/: 统计分析                │
│  - internal/event/: 事件管理                │
│  - internal/mapping/: 映射管理              │
├─────────────────────────────────────────────┤
│  Core Layer (领域模型)                      │  ← 纯粹的业务概念
│  - core/field.go: 字段类型系统              │
│  - core/query.go: 查询条件模型              │
│  - core/auth.go: 认证模型                   │
│  - core/errors.go: 错误定义                 │
└─────────────────────────────────────────────┘
```

**依赖规则**：
- ✅ **允许**：Engine → Internal → Core
- ❌ **禁止**：Core → Internal、Core → Engine
- ⚠️ **原则**：依赖只能向下，不能向上或横向

---

### Core 层的黄金法则

#### ✅ **Core 层应该是什么**

1. **纯粹的领域模型**
   - 代表业务概念：`TypedValue`、`FieldMapping`、`QueryCondition`
   - 包含业务规则：`IsOptionField()`、`IsImageField()`
   - 独立于技术实现

2. **稳定的抽象**
   - 变化频率低（业务概念变化缓慢）
   - 被多个模块依赖
   - 定义清晰的接口（如 `CacheInterface`）

3. **框架无关**
   - 只使用 Go 标准库类型
   - 不依赖特定框架（Redis、ORM、HTTP）

#### ❌ **Core 层不应该是什么**

1. **不是数据传输对象 (DTO)**
   - 不包含 JSON 序列化逻辑
   - 不为 API 响应格式设计
   - ❌ **禁止定义 Request/Response 结构**（例外：跨多层传递的 DTO，如协议层→Engine→Internal）

2. **不是 ORM 模型**
   - **禁止出现**：`db:"field_name"` 标签
   - **禁止出现**：`bson:"field_name"` 标签
   - **禁止出现**：`gorm:` 标签

3. **不是数据库查询结果**
   - 不为数据库扫描而设计
   - 不包含 `sql.NullString`、`sql.NullTime` 等框架类型

#### 🎯 **Core 层组织原则**

**API 层按领域组织**（而非技术分类）：
- ✅ `flow.go` - 流程领域（API路径、操作常量）
- ✅ `form.go` - 表单领域
- ✅ `image.go` - 图片上传领域
- ❌ ~~`constants.go`~~ - 避免技术分类文件

**Manager 接口设计原则**（Internal 层）：
- ✅ **直接使用领域模型**作为参数（如 `eventConfig *query.EventConfig, fields []query.FieldConfig`）
- ❌ **禁止定义 Request 结构体**（REST 层有 go-zero 生成的结构体，无需在包内重复定义）

---

### Internal 层的职责边界

#### Manager 模式

所有 Internal 子包采用 **Manager 模式**：
- 对外暴露接口（如 `SkylarkFlowRegistry`）
- 通过 `New<Manager>()` 函数创建实例
- 依赖通过构造函数注入

#### 文件组织规范

**所有 Manager 子包必须遵守以下文件组织**：

| 文件名 | 职责 | 必需性 |
|--------|-----|--------|
| `<manager>.go` | 接口定义 | ✅ 必需 |
| `handler.go` | 接口实现 | ✅ 必需 |
| `model.go` | 数据库模型（包含 db 标签和 ToDomain 方法） | ✅ 必需 |
| `helpers.go` | 未导出的辅助函数（convertNullString等） | 🟡 推荐 |
| `internal.go` | 内部方法（不对外暴露） | 🟡 推荐 |
| `sql.go` | SQL 语句常量 | 🟡 可选 |
| `config.go` | Manager 配置结构体 | ✅ 必需 |

**重要约定**：
- `model.go` 只包含结构体定义和导出方法（如 `ToDomain()`）
- `helpers.go` 包含所有未导出的辅助函数（如类型转换函数）

#### model.go 的作用

```go
// internal/platform/model.go - 数据库模型（允许框架类型）
type PlatformConfigModel struct {
    ID          string         `db:"id"`
    Description sql.NullString `db:"description"` // ✅ 可空字段
    // ...
}

func (m *PlatformConfigModel) ToDomain() *core.PlatformConfig {
    return &core.PlatformConfig{
        Description: convertNullString(m.Description), // 转为 *string
        // ...
    }
}

// core/platform.go - 领域模型（纯 Go 类型）
type PlatformConfig struct {
    Description *string // ✅ 使用指针表示可空
    // ...
}
```

#### Internal 层的其他职责

1. **API 请求封装**
   - 调用 Skylark 远程 API
   - 处理 HTTP 请求/响应
   - 错误转换（HTTP 错误 → Core 错误）

2. **数据库操作**
   - 执行 SQL 查询
   - 使用 `model.go` 中的结构体扫描结果
   - 转换为 Core 层类型返回

3. **缓存管理**
   - 通过 `CacheInterface` 操作缓存
   - 字段映射缓存
   - 配置缓存

4. **业务逻辑编排**
   - 协调多个操作
   - 事务管理
   - 重试逻辑

#### Internal 层的边界

✅ **应该做**：
- 返回纯 Core 对象
- 包含技术实现细节
- 使用 `sql.NullString` 等框架类型（仅在 model.go 中）

❌ **不应该做**：
- 不暴露数据库模型给上层
- 不在接口签名中使用 `sql.Null*` 类型
- 不包含业务规则（业务规则属于 Core 层）

---

### Engine 层的职责边界

#### 单一职责

Engine 层是**用户唯一的入口点**：

```go
// 用户代码
engine := skylark.NewEngine(config)
flow, err := engine.CreateFlow(authHeader, templateID, data)
```

#### 职责清单

1. **接口聚合**
   - 将多个 Manager 组合成统一接口
   - 提供高层抽象（如 `CreateFlow`、`CreateFormRow`）

2. **依赖注入**
   - 创建和管理 Manager 实例
   - 注入缓存、数据库连接等依赖

3. **配置管理**
   - 解析用户配置
   - 初始化各个 Manager

#### 边界

✅ **应该做**：
- 组合调用多个 Manager
- 返回 Core 层类型
- 处理全局错误

❌ **不应该做**：
- 不包含业务逻辑（委托给 Manager）
- 不直接操作数据库
- 不直接调用外部 API

---

## 🎯 核心概念

### Skylark 平台

Skylark 是一个**低代码平台**，通过积木式搭建模式快速构建业务应用。

**核心能力**：
- 📋 **表单管理**：自定义字段、类型丰富
- 🔄 **流程管理**：工作流编排、状态流转
- 👥 **组织管理**：多租户、角色权限
- 🏷️ **标签管理**：分类、筛选
- 📊 **数据查询**：灵活的条件组合
- 📈 **统计分析**：时长统计、事件统计

### go-skylark 能力矩阵

| 模块 | 写操作 | 读操作 | 统计分析 | 状态 |
|------|--------|--------|---------|------|
| **Platform** | 管理平台配置 | 查询配置 | - | ✅ |
| **Flow** | 创建流程、更新状态 | 查询流程 | 时长统计 | ✅ |
| **Form** | 创建表单行 | 查询表单 | - | ✅ |
| **Event** | 配置事件 | 查询事件 | 待处理统计 | ✅ |
| **Mapping** | 配置映射 | 查询映射 | - | ✅ |
| **Query** | - | 复杂条件查询 | - | ✅ |
| **Stats** | - | - | 多维度统计 | ✅ |

### Manager 设计规范

所有 Internal 层的模块应遵循统一的 Manager 模式：

#### Manager 职责划分

| Manager 类型 | 职责 | 依赖注入需求 | 示例 |
|-------------|------|-------------|------|
| **配置管理** | 管理本地配置、远程连接 | DB、Cache | Platform、Event、Mapping |
| **API 代理** | 调用远程 API、数据转换 | Cache、GetPlatformConfig | Flows、Forms |
| **查询引擎** | 构建查询、执行查询 | DB、Cache、GetRemoteDB | Query、Stats |
| **映射转换** | ID 映射、数据转换 | DB、Cache、GetRemoteDB | Organization、User |

#### Manager 必须实现的接口

每个 Manager 应该根据职责实现以下接口类型：

**1. 配置管理接口**
```go
// 平台配置管理
type Manager interface {
    Create(ctx context.Context, config *Config) error
    Get(ctx context.Context, id string) (*Config, error)
    Update(ctx context.Context, config *Config) error
    Delete(ctx context.Context, id string) error
}
```

**2. API 代理接口**
```go
// 远程 API 调用
type Registry interface {
    CreateResource(ctx context.Context, req *Request) error
    GetResource(ctx context.Context, id string) (*Resource, error)
    ListResources(ctx context.Context, params *ListParams) ([]*Resource, int, error)
}
```

**3. 查询引擎接口**
```go
// 数据查询
type Manager interface {
    Query(ctx context.Context, req *QueryRequest) (*QueryResult, error)
    GetDetail(ctx context.Context, req *DetailRequest) (*DetailResult, error)
}
```

**4. 映射转换接口**
```go
// ID 映射
type Manager interface {
    GetRemoteIDs(ctx context.Context, tenantID string, localIDs []string) ([]int, error)
    BatchGetInfo(ctx context.Context, tenantID string, ids []int) (map[int]*Info, error)
}
```

#### 核心 Manager 示例

**PlatformManager**：管理远程平台配置和数据库连接池
- 提供 `GetRemoteDB()` 给其他 Manager 使用
- 提供 `GetAPIConfig()` 给 API 代理 Manager 使用

**FlowManager**：流程生命周期管理（写+读）
- 依赖：Platform（API 配置 + 远程 DB）、User（用户 ID 映射）
- 性能优化：enrichment 自动补充关联数据

**QueryManager**：复杂查询引擎（只读）
- 依赖：Platform（远程 DB）、Event（事件配置）、Mapping（组织映射）
- 特性：组织权限过滤、虚拟状态支持

**OrganizationManager / UserManager**：ID 映射转换
- 依赖：Platform（远程 DB）
- 特性：批量查询、缓存优化

---

## 🏗️ 数据存储架构

### 混合存储策略

```
┌─────────────────────────────────────────────┐
│  Skylark 远程 API                           │  ← 主数据源
│  - 创建流程 (POST /flows)                   │
│  - 创建表单 (POST /forms)                   │
│  - 更新状态 (PUT /flows/:id/status)         │
└─────────────────────────────────────────────┘
             ↕ (API 调用)
┌─────────────────────────────────────────────┐
│  go-skylark SDK                             │
└─────────────────────────────────────────────┘
             ↕ (查询 & 缓存)
┌─────────────────────────────────────────────┐
│  本地数据库 (PostgreSQL/MySQL)              │  ← 查询优化
│  - platform_configs: 平台配置               │
│  - event_configs: 事件配置                  │
│  - org_mappings: 组织映射                   │
│  - field_configs: 字段配置                  │
└─────────────────────────────────────────────┘
             ↕ (缓存)
┌─────────────────────────────────────────────┐
│  Redis 缓存                                 │  ← 性能优化
│  - 字段映射缓存 (field:mapping:{tenantID})  │
│  - 配置缓存 (platform:config:{tenantID})    │
└─────────────────────────────────────────────┘
```

### 数据流向

#### 写操作（创建流程）

```
用户代码
  ↓ engine.CreateFlow()
Engine Layer
  ↓ flowManager.CreateFlow()
Internal Layer (FlowManager)
  ↓ 1. 获取字段映射 (缓存 → 数据库 → API)
  ↓ 2. 转换字段值 (TypedValue → API 格式)
  ↓ 3. 上传图片 (Base64 → 七牛云)
  ↓ 4. 调用远程 API (POST /flows)
Skylark API
  ← 返回流程 ID
用户代码
```

#### 读操作（查询流程）

```
用户代码
  ↓ queryManager.QueryFlows()
Internal Layer (QueryManager)
  ↓ 1. 构建 SQL 查询
  ↓ 2. 执行数据库查询
  ↓ 3. 扫描结果 → model.go 结构体
  ↓ 4. 转换 model → core 对象
  ← 返回 []core.FlowInfo
用户代码
```

---

## 📋 开发规范

### 数据库查询规范

#### 使用 sql.Null* 处理可空字段

**原因**：Skylark 平台的可选字段在数据库中可能为 NULL。

**规范**：
```go
// ✅ 正确：在 model.go 中使用 sql.Null*
type PlatformConfigModel struct {
	Description sql.NullString `db:"description"`
	UpdatedAt   sql.NullTime   `db:"updated_at"`
}

// ❌ 错误：直接使用 string/time.Time
type PlatformConfigModel struct {
	Description string    `db:"description"` // 遇到 NULL 会报错
	UpdatedAt   time.Time `db:"updated_at"`  // 遇到 NULL 会报错
}
```

**转换函数**：
```go
// helpers.go
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

#### 使用 go-zero sqlx

```go
// 查询单行
var model PlatformConfigModel
err := conn.QueryRow(&model, query, args...)

// 查询多行
var models []PlatformConfigModel
err := conn.QueryRows(&models, query, args...)
```

#### 数组类型使用 pq.Array

```go
import "github.com/lib/pq"

type QueryModel struct {
	Tags pq.StringArray `db:"tags"`
}
```

---

### 领域模型转换规范

#### 转换方向

```
数据库 → model.go → core 对象 → 用户
```

#### 转换函数位置

**推荐方式：在 model.go 中定义**

```go
// internal/platform/model.go
func (m *PlatformConfigModel) ToDomain() *core.PlatformConfig {
    return &core.PlatformConfig{
        Description: convertNullString(m.Description),
        // ...
    }
}

// internal/platform/handler.go - 使用
func (h *handler) Get(tenantID string) (*core.PlatformConfig, error) {
    var model PlatformConfigModel
    err := h.db.QueryRow(&model, query, tenantID)
    return model.ToDomain(), err  // 直接转换
}
```

---

### 错误处理规范

#### 错误分类

go-skylark 将错误分为两大类（参考 `core/errors.go`）：

1. **🔵 API 请求错误**（13 个）
   - `ErrInvalidAuthHeader` - 认证头无效
   - `ErrInvalidFlowTemplateID` - 流程模板 ID 无效
   - `ErrCreateFlowFailed` - 创建流程失败
   - ...

2. **🟢 数据库查询错误**（23 个）
   - `ErrNoPlatformConfig` - 未找到平台配置
   - `ErrNoEventConfig` - 未找到事件配置
   - `ErrQueryFlowFailed` - 查询流程失败
   - ...

#### 错误使用

```go
// Internal 层 - 使用预定义错误
func (h *handler) Get(tenantID string) (*core.Config, error) {
    var model ConfigModel
    if err := h.db.QueryRow(&model, query, tenantID); err != nil {
        if errors.Is(err, sqlx.ErrNotFound) {
            return nil, core.ErrPlatformConfigNotFound // ✅ 预定义错误
        }
        return nil, fmt.Errorf("查询失败: %w", err)
    }
    return model.ToDomain(), nil
}

// 用户代码 - 判断错误类型
if errors.Is(err, core.ErrPlatformConfigNotFound) {
    // 处理未找到的情况
}
```

#### logx 使用规范

```go
// ✅ 使用 Error
logx.Error("创建流程失败", err)

// ❌ 不使用 Warn（go-zero 项目规范）
logx.Warn("警告信息") // 禁止使用
```

---

### Manager 解耦机制

#### 问题：Manager 之间如何通信？

示例：`OrganizationManager` 需要调用 `PlatformManager` 获取配置，同时需要缓存支持。

#### 解决方案：通过 NewManager 函数参数注入依赖

**核心原则**：依赖通过构造函数参数传入，不放在 Config 中。

```go
// config.go - 只包含配置参数
type Config struct {
	// 空或只包含基础配置
}

// 依赖通过 NewManager 参数传入
func NewManager(
	config Config,                        // 纯配置
	db sqlx.SqlConn,                      // 依赖1
	cache core.CacheInterface,            // 依赖2
	getPlatformConfig core.GetPlatformConfigFunc, // 依赖3（函数注入）
) (Manager, error)

// 使用示例
orgMgr, _ := organization.NewManager(
	organization.Config{}, db, cacheMgr, platformMgr.Get,
)
```

**优势**：Config 职责单一、依赖显式化、易于测试、避免循环依赖

---

### Config 结构体规范

#### 黄金法则

✅ **Config 只能包含**：
- 基础配置参数（字符串、数字、布尔值、time.Duration 等）

❌ **Config 禁止包含**：
- 具体实现类型（如 `*redis.Client`、`sqlx.SqlConn`）
- 接口类型（如 `CacheInterface`）
- 函数类型（如 `GetRemoteDBFunc`）
- 运行时实例（如 `*sql.DB`）

**原因**：Config 结构体应该只存储**静态配置参数**，所有依赖（接口、函数、运行时实例）都应该通过 **NewManager 函数参数**传入。

#### 示例

```go
// ✅ 正确的 Config
type Config struct {
	MaxPageSize      int           // 数字配置
	DefaultPageSize  int           // 数字配置
	FlowListCacheTTL time.Duration // 时长配置
	QueryTimeout     time.Duration // 时长配置
}

// ❌ 错误的 Config
type Config struct {
	Cache       core.CacheInterface // ❌ 接口类型
	GetRemoteDB GetRemoteDBFunc     // ❌ 函数类型
	RedisClient *redis.Client       // ❌ 具体实现
	DB          sqlx.SqlConn        // ❌ 运行时实例
}
```

#### 正确的依赖注入方式

```go
// ✅ 正确：依赖通过 NewManager 参数传入
func NewManager(
    config Config,                   // 纯配置参数
    db sqlx.SqlConn,                 // 依赖1
    cache core.CacheInterface,       // 依赖2
    getRemoteDB core.GetRemoteDBFunc // 依赖3（函数注入）
) (Manager, error)

// ❌ 错误：依赖放在 Config 中
type Config struct {
    DB    sqlx.SqlConn              // ❌ 运行时实例
    Cache core.CacheInterface       // ❌ 接口类型
}
```

---

### 数据库连接依赖注入

#### 多租户数据库场景

每个租户使用独立数据库，需要动态获取连接。

#### 解决方案：函数注入

```go
// core/platform.go - 定义函数类型
type GetRemoteDBFunc func(ctx context.Context, tenantID string) (sqlx.SqlConn, error)

// internal/query/handler.go - 使用
func (h *handler) Query(tenantID string) ([]core.FlowInfo, error) {
    conn, _ := h.getRemoteDB(ctx, tenantID)  // 动态获取租户数据库
    var models []FlowInfoModel
    conn.QueryRows(&models, query)
    // ...
}
```

---

## 🚀 性能优化设计模式

### 批量查询优化模式

**适用场景**：需要从远程数据库或 API 获取多个关联对象的信息

**问题**：N+1 查询问题导致性能瓶颈

**解决方案**：
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

**关键原则**：
- 使用 `pq.Array` 进行批量 SQL 查询（PostgreSQL）
- 容错处理：部分失败不影响整体流程

### 多级缓存策略

**适用场景**：频繁访问的关联数据（如 Flow 信息、用户信息）

**缓存层级**：
```
1. Redis 缓存（优先） → 2. 批量 API 调用 → 3. 异步回写缓存
```

**实现规范**：
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

**关键原则**：
- TTL 通过 Config 配置（如 `FlowInfoCacheTTL`）
- 异步回写使用 `context.Background()`（避免主请求取消影响缓存）

### 数据补充（Enrichment）模式

**适用场景**：API 返回的数据缺少关联信息，需要自动补充

**解决方案**：
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

**关键原则**：
- Enrichment 是**可选的**（失败不影响主流程）
- 补充字段使用**指针类型**（如 `*string`）
- 用户无需关心 enrichment 细节（SDK 内部自动处理）

**示例参考**：`internal/flows/enrichment.go`

### 性能优化清单

添加新接口时，检查是否需要性能优化：

- [ ] 是否存在 N+1 查询？→ 使用批量查询
- [ ] 是否频繁访问相同数据？→ 添加缓存
- [ ] 是否需要关联数据？→ 考虑 enrichment 模式
- [ ] 缓存 TTL 是否可配置？→ 添加到 Config
- [ ] 是否有容错机制？→ 优化失败不影响主流程
- [ ] 是否记录性能指标？→ 添加日志（可选）

---

## 🔧 字段类型系统

### TypedValue 设计

Skylark 平台的字段值需要携带类型信息。

```go
// core/field.go
type TypedValue struct {
	Type  string      // 字段类型
	Value interface{} // 字段值
}

// 示例
name := TypedValue{Type: "string", Value: "张三"}
avatar := TypedValue{Type: "imageURL", Value: "https://example.com/avatar.jpg"}
file := TypedValue{Type: "imageBase64", Value: "base64encodedstring..."}
```

### 特殊字段后缀

#### 图片字段

| 后缀 | 含义 | 处理方式 |
|------|-----|---------|
| `_Img` | 图片 URL 字段 | 直接传递 URL |
| `_Base64Img` | Base64 编码图片 | 上传到七牛云 → 转换为 URL |

**示例**：
```go
data := map[string]core.TypedValue{
    "Avatar_Img":       {Type: "imageURL", Value: "https://..."},
    "IDCard_Base64Img": {Type: "imageBase64", Value: "data:image/png;base64,..."},
}
```

### 选项字段类型

支持的选项类型（参考 `core/var.go`）：
- `RadioButton` - 单选按钮
- `Checkbox` - 复选框
- `SelectField` - 下拉选择
- `MultipleSelectField` - 多选下拉

**判断函数**：
```go
// core/field.go
func IsOptionField(fieldType string) bool {
	return fieldType == RadioButton ||
	       fieldType == Checkbox ||
	       fieldType == SelectField ||
	       fieldType == MultipleSelectField
}
```

---


## 📚 关键文件索引

### Core 层

| 文件 | 描述 | 关键类型 |
|------|-----|---------|
| `core/field.go` | 字段类型系统 | `TypedValue`、`FieldMapping`、`FieldOption` |
| `core/query.go` | 查询条件模型 | `QueryCondition`、`QueryOptions` |
| `core/auth.go` | 认证模型 | `AuthHeader` |
| `core/errors.go` | 错误定义 | 36 个预定义错误 |
| `core/var.go` | 常量定义 | API 路径、字段类型常量 |
| `core/cache.go` | 缓存接口 | `CacheInterface` |
| `core/address.go` | 地址处理 | 地址相关功能 |
| `core/platform.go` | 平台配置 | `PlatformConfig` |
| `core/event.go` | 事件配置 | `EventConfig` |
| `core/flow.go` | 流程信息 | `FlowInfo` |
| `core/mapping.go` | 组织映射 | `OrgMapping` |
| `core/stats.go` | 统计结果 | `DurationStats` |

### Engine 层

| 文件 | 描述 |
|------|-----|
| `engine/engine.go` | `SkylarkEngine` 接口定义 |
| `engine/handler.go` | Engine 实现 |
| `engine/config.go` | Engine 配置 |

### Internal 层

| 子包 | 核心文件 | 接口 |
|------|---------|------|
| `internal/cache/` | `cache.go`, `handler.go` | 实现 `CacheInterface` |
| `internal/platform/` | `platform.go`, `handler.go` | `SkylarkPlatformRegistry` |
| `internal/flows/` | `flows.go`, `handler.go` | `SkylarkFlowRegistry` |
| `internal/forms/` | `forms.go`, `handler.go` | `SkylarkFormRegistry` |
| `internal/query/` | `query.go`, `handler.go`, `build.go` | `SkylarkQueryRegistry` |
| `internal/stats/` | `stats.go`, `handler.go` | `SkylarkStatsRegistry` |
| `internal/event/` | `event.go`, `handler.go` | `SkylarkEventRegistry` |
| `internal/mapping/` | `mapping.go`, `handler.go` | `SkylarkMappingRegistry` |
| `internal/images/` | `images.go` | 图片处理 |

---

## 📖 相关文档

### 项目文档
- [INTEGRATION_PLAN.md](./INTEGRATION_PLAN.md) - 整合计划与进度跟踪
- [CLAUDE.md](./CLAUDE.md) - Claude Code 助手指引

### 模块文档
- [internal/cache/README.md](./internal/cache/README.md) - 缓存模块详细文档（优化策略、监控指标、最佳实践）
- [internal/flows/README.md](./internal/flows/README.md) - 流程模块详细文档（11个接口、性能优化、使用示例）
- [internal/stats/MULTI_EVENT_STATS.md](./internal/stats/MULTI_EVENT_STATS.md) - 多事件统计设计文档

### 外部文档
- [Go 官方文档](https://go.dev/doc/)
- [go-zero 文档](https://go-zero.dev/)
- [sqlx 文档](https://jmoiron.github.io/sqlx/)

---

**最后更新**：2025-11-12
**维护者**：go-skylark 开发团队

---

## 📝 更新日志

### 2025-11-12
- 添加"性能优化设计模式"章节（批量查询、多级缓存、Enrichment 模式）
- 更新"四大管理器"为"Manager 设计规范"（更具指导性）
- 补充 internal/flows 模块文档链接
- 添加性能优化清单（供开发时参考）

### 2025-11-07
- 初始版本

# Skylarkq 整合计划

## 一、整合概述

### 1.1 目标
将 `nexlyn/pkg/skylarkq` 的查询和统计分析能力整合到 `go-skylark` 项目中，形成完整的 Skylark 低代码平台 SDK。

### 1.2 整合策略
采用**直接复制后批量替换**的方式，利用子包文件夹名不重复的优势，简化整合流程。

### 1.3 整合后的能力矩阵

| 能力分类 | 来源 | 状态 |
|---------|------|------|
| **写操作** | | |
| 创建流程 | go-skylark | ✅ 保留 |
| 创建表单行 | go-skylark | ✅ 保留 |
| 更新流程任务状态 | go-skylark | ✅ 保留 |
| 图片上传 | go-skylark | ✅ 保留 |
| **读操作** | | |
| Journey 查询 | skylarkq | 🆕 新增 |
| 事件详情查询 | skylarkq | 🆕 新增 |
| Flow 列表查询 | skylarkq | 🆕 新增 |
| **配置管理** | | |
| 平台配置管理 | skylarkq | 🆕 新增 |
| 事件配置管理 | skylarkq | 🆕 新增 |
| 组织映射管理 | skylarkq | 🆕 新增 |
| **统计分析** | | |
| 处理时长统计 | skylarkq | 🆕 新增 |
| 状态统计 | skylarkq | 🆕 新增 |
| 趋势统计 | skylarkq | 🆕 新增 |
| 节点统计 | skylarkq | 🆕 新增 |
| 处理人统计 | skylarkq | 🆕 新增 |
| 组织统计 | skylarkq | 🆕 新增 |
| 待处理统计 | skylarkq | 🆕 新增 |
| **基础设施** | | |
| 字段映射缓存 | go-skylark | ✅ 保留 |
| 分布式锁 | go-skylark | ✅ 保留 |
| 用户名缓存 | skylarkq | 🆕 新增 |
| 多层缓存策略 | skylarkq | 🆕 新增 |

---

## 二、整合步骤

### ✅ 阶段一：文件复制与结构调整（已完成）

#### 1.1 复制 skylarkq 文件到 go-skylark

**从 `D:\Documents\Code_git\nexlyn\pkg\skylarkq\` 复制以下目录：**

```bash
# 复制 core 下的新文件
skylarkq/core/platform.go       → go-skylark/core/platform.go
skylarkq/core/event.go          → go-skylark/core/event.go
skylarkq/core/mapping.go        → go-skylark/core/mapping.go
skylarkq/core/flow.go           → go-skylark/core/flow.go
skylarkq/core/query.go          → go-skylark/core/query.go
skylarkq/core/stats.go          → go-skylark/core/stats.go
skylarkq/core/errors.go         → go-skylark/core/errors_query.go  # 避免冲突

# 复制 internal 下的新模块（整个目录）
skylarkq/internal/platform/     → go-skylark/internal/platform/
skylarkq/internal/event/        → go-skylark/internal/event/
skylarkq/internal/mapping/      → go-skylark/internal/mapping/
skylarkq/internal/query/        → go-skylark/internal/query/
skylarkq/internal/stats/        → go-skylark/internal/stats/

# 复制文档
skylarkq/DEVELOPMENT.md         → go-skylark/docs/SKYLARKQ_DEVELOPMENT.md
skylarkq/internal/stats/MULTI_EVENT_STATS.md → go-skylark/docs/MULTI_EVENT_STATS.md
```

**保留 go-skylark 现有文件：**
```
go-skylark/core/auth.go
go-skylark/core/address.go
go-skylark/core/cache.go
go-skylark/core/field.go
go-skylark/core/utils.go
go-skylark/core/var.go
go-skylark/internal/cache/
go-skylark/internal/flows/
go-skylark/internal/forms/
go-skylark/internal/images/
```

#### 1.2 创建新目录

```bash
# 创建文档目录
go-skylark/docs/

# 整合后的目录结构
go-skylark/
├── core/
│   ├── address.go          # 保留
│   ├── auth.go             # 保留
│   ├── cache.go            # 保留
│   ├── event.go            # 新增
│   ├── errors_query.go     # 新增（来自 skylarkq/errors.go）
│   ├── field.go            # 保留
│   ├── flow.go             # 新增
│   ├── mapping.go          # 新增
│   ├── platform.go         # 新增
│   ├── query.go            # 新增
│   ├── stats.go            # 新增
│   ├── utils.go            # 保留
│   └── var.go              # 保留
├── engine/
│   ├── config.go           # 需扩展
│   ├── engine.go           # 需扩展
│   └── handler.go          # 需扩展
├── internal/
│   ├── cache/              # 保留
│   ├── event/              # 新增
│   ├── flows/              # 保留
│   ├── forms/              # 保留
│   ├── images/             # 保留
│   ├── mapping/            # 新增
│   ├── platform/           # 新增
│   ├── query/              # 新增
│   └── stats/              # 新增
└── docs/
    ├── SKYLARKQ_DEVELOPMENT.md
    └── MULTI_EVENT_STATS.md
```

---

### ✅ 阶段二：批量替换导入路径（已完成）

#### 2.1 替换规则

**在所有新复制的文件中执行以下替换：**

| 原路径 | 新路径 |
|--------|--------|
| `github.com/your-org/nexlyn/pkg/skylarkq/core` | `github.com/your-org/go-skylark/core` |
| `github.com/your-org/nexlyn/pkg/skylarkq/engine` | `github.com/your-org/go-skylark/engine` |
| `github.com/your-org/nexlyn/pkg/skylarkq/internal/platform` | `github.com/your-org/go-skylark/internal/platform` |
| `github.com/your-org/nexlyn/pkg/skylarkq/internal/event` | `github.com/your-org/go-skylark/internal/event` |
| `github.com/your-org/nexlyn/pkg/skylarkq/internal/mapping` | `github.com/your-org/go-skylark/internal/mapping` |
| `github.com/your-org/nexlyn/pkg/skylarkq/internal/query` | `github.com/your-org/go-skylark/internal/query` |
| `github.com/your-org/nexlyn/pkg/skylarkq/internal/stats` | `github.com/your-org/go-skylark/internal/stats` |

**注意：** 需要先确定 go-skylark 的实际 module 路径（从 `go.mod` 获取）

#### 2.2 批量替换工具命令

**使用 VSCode 全局替换：**
1. 打开 VSCode
2. 按 `Ctrl+Shift+H` 打开全局替换
3. 勾选"使用正则表达式"
4. 查找：`github\.com/[^/]+/nexlyn/pkg/skylarkq`
5. 替换为：`github.com/your-org/go-skylark`（根据实际 module 路径）
6. 仅在新复制的文件中替换

**或使用命令行（Windows PowerShell）：**
```powershell
# 在 go-skylark 目录下执行
$files = Get-ChildItem -Recurse -Include *.go
foreach ($file in $files) {
    (Get-Content $file.PSPath) |
    ForEach-Object { $_ -replace 'github\.com/[^/]+/nexlyn/pkg/skylarkq', 'github.com/your-org/go-skylark' } |
    Set-Content $file.PSPath
}
```

---

### ✅ 阶段三：Core 文件优化（已完成）

完成了 core 目录下所有文件的注释优化和职能重组：

#### 3.1 文件用途分类注释
- **🔵 API 请求文件**（3个）：auth.go, address.go, utils.go
- **🟢 数据库查询文件**（6个）：platform.go, event.go, mapping.go, query.go, stats.go, flow.go
- **🔵🟢 混合文件**（2个）：field.go, cache.go（带分隔注释）

#### 3.2 错误和常量重组
- **errors.go**：合并了 var.go 的错误定义，按用途分类（🔵 13个 + 🟢 23个）
- **var.go**：仅保留常量定义，分为 7 组

#### 3.3 编译验证
```bash
✅ go build ./...   # 编译通过
✅ go vet ./...     # 静态分析通过
✅ go fmt ./...     # 代码格式化完成
```

---

### 阶段四：合并冲突和适配（待执行）

#### 4.1 领域模型重构（优先级 P0）

**背景：**
当前 Core 层存在严重的领域设计问题，违反了 DDD 原则：
- ❌ Core 层包含 50+ 个 `db:` 标签
- ❌ 所有 Internal 子包缺失 `model.go` 文件
- ❌ Core 层使用 `sql.NullString` 等框架类型

**目标：**
- Core 层成为纯粹的领域模型（无框架依赖）
- Internal 层通过 `model.go` 处理数据库映射
- 明确的转换边界（model → domain）

**详细计划见阶段六**

#### 4.2 字段类型系统整合

**问题：**
- `go-skylark` 使用 `TypedValue` 结构
- `skylarkq` 使用 `FieldConfig` 结构

**解决方案：扩展 `core/field.go`**

```go
// 保留原有的 TypedValue, FieldMapping, FieldOption

// 新增 skylarkq 的字段配置
type FieldConfig struct {
    ID            string    `db:"id"`
    EventConfigID string    `db:"event_config_id"`
    FieldName     string    `db:"field_name"`
    DisplayName   string    `db:"display_name"`
    FieldType     string    `db:"field_type"`
    IsVisible     bool      `db:"is_visible"`
    DisplayOrder  int       `db:"display_order"`
    IsSearchable  bool      `db:"is_searchable"`
    TenantID      string    `db:"tenant_id"`
    CreatedAt     time.Time `db:"created_at"`
    UpdatedAt     time.Time `db:"updated_at"`
}

// FieldMapping 和 FieldConfig 的转换方法
func (fc *FieldConfig) ToFieldMapping() FieldMapping {
    return FieldMapping{
        ID:          0, // FieldConfig 不存储远程 ID
        IdentityKey: fc.FieldName,
        Type:        fc.FieldType,
        Options:     []FieldOption{}, // 需要单独加载
    }
}
```

#### 4.3 缓存接口适配

**问题：**
- `go-skylark` 使用 `CacheInterface` 接口抽象
- `skylarkq` 直接使用 `*redis.Redis` 客户端

**解决方案：保留接口抽象，扩展实现**

**扩展 `core/cache.go` 接口：**
```go
type CacheInterface interface {
    // 原有方法（流程/表单字段映射缓存）
    ClearFieldMappingsCache(ctx context.Context, cacheKey string) error
    GetFieldMappingsFromCache(ctx context.Context, cacheKey string) (map[string]FieldMapping, bool, error)
    SaveFieldMappingsToCache(ctx context.Context, cacheKey string, fieldMappings map[string]FieldMapping) error
    AcquireLock(ctx context.Context, key string, value string, expiry int) (bool, error)
    AcquireLockWithRetry(ctx context.Context, key string, value string, expiry int) error
    ReleaseLock(ctx context.Context, key string, value string) error
    ExtendLock(ctx context.Context, key string, value string, expiry int) error

    // 新增方法（skylarkq 查询缓存）
    // 通用缓存操作
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value string) error
    Setex(ctx context.Context, key string, value string, seconds int) error
    Del(ctx context.Context, keys ...string) error

    // 结构化数据缓存（JSON）
    GetJSON(ctx context.Context, key string, dest interface{}) error
    SetJSON(ctx context.Context, key string, value interface{}) error
    SetJSONEx(ctx context.Context, key string, value interface{}, seconds int) error
}
```

**扩展 `internal/cache/cache.go` 实现：**
```go
// 实现新增的缓存方法
func (c *SkylarkCache) Get(ctx context.Context, key string) (string, error) {
    return c.redisClient.GetCtx(ctx, key)
}

func (c *SkylarkCache) Setex(ctx context.Context, key string, value string, seconds int) error {
    return c.redisClient.SetexCtx(ctx, key, value, seconds)
}

func (c *SkylarkCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
    val, err := c.redisClient.GetCtx(ctx, key)
    if err != nil {
        return err
    }
    return json.Unmarshal([]byte(val), dest)
}

func (c *SkylarkCache) SetJSONEx(ctx context.Context, key string, value interface{}, seconds int) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return c.redisClient.SetexCtx(ctx, key, string(data), seconds)
}
```

**修改 skylarkq 模块的缓存调用：**

在 `internal/platform/`, `internal/event/`, `internal/mapping/`, `internal/query/`, `internal/stats/` 的所有文件中：

```go
// 原代码（skylarkq）
type platformManager struct {
    localDB     sqlx.SqlConn
    remoteDBs   map[string]sqlx.SqlConn
    rdb         *redis.Redis  // 直接依赖 Redis
}

// 修改为（go-skylark）
type platformManager struct {
    localDB     sqlx.SqlConn
    remoteDBs   map[string]sqlx.SqlConn
    cache       core.CacheInterface  // 使用接口
}

// 修改缓存调用
// 原：m.rdb.Setex(key, val, ttl)
// 新：m.cache.Setex(ctx, key, val, ttl)
```

#### 4.4 Engine 接口扩展

**扩展 `engine/engine.go`：**

```go
// SkylarkEngine - 原有的创建能力
type SkylarkEngine interface {
    CreateFlow(ctx context.Context, app string, flowID int64, userID int64,
               authHeader string, data map[string]core.TypedValue) error
    CreateFormRow(ctx context.Context, app string, formID int64, userID int64,
                  authHeader string, data map[string]core.TypedValue) error
    UpdateFlowJourneyStatus(ctx context.Context, app string, flowID int64,
                           journeyID int64, assignmentID int64, userID int64,
                           authHeader string, operation string,
                           options flows.UpdateJourneyStatusOptions) error
}

// SkylarkQuery - 新增的查询能力（来自 skylarkq）
type SkylarkQuery interface {
    // 平台配置管理
    CreatePlatformConfig(ctx context.Context, config *core.PlatformConfig) error
    GetPlatformConfig(ctx context.Context, tenantID string) (*core.PlatformConfig, error)
    UpdatePlatformConfig(ctx context.Context, config *core.PlatformConfig) error
    DeletePlatformConfig(ctx context.Context, tenantID string) error
    ValidatePlatformConnection(ctx context.Context, tenantID string) error

    // 事件配置管理
    CreateEventConfigWithFields(ctx context.Context, eventConfig *core.EventConfigWithFields) error
    GetEventConfigWithFields(ctx context.Context, eventConfigID string) (*core.EventConfigWithFields, error)
    ListEventConfigsWithFields(ctx context.Context, tenantID string) ([]*core.EventConfigWithFields, error)
    UpdateEventConfigWithFields(ctx context.Context, eventConfig *core.EventConfigWithFields) error
    DeleteEventConfig(ctx context.Context, eventConfigID string) error

    // 组织映射管理
    CreateOrgMapping(ctx context.Context, mapping *core.OrgMapping) error
    GetOrgMapping(ctx context.Context, id string) (*core.OrgMapping, error)
    ListOrgMappings(ctx context.Context, tenantID string) ([]*core.OrgMapping, error)
    UpdateOrgMapping(ctx context.Context, mapping *core.OrgMapping) error
    DeleteOrgMapping(ctx context.Context, id string) error

    // 远程数据查询
    GetFlowList(ctx context.Context, req *core.FlowListRequest) (*core.FlowListResponse, error)
    GetFlowFields(ctx context.Context, req *core.FlowFieldsRequest) (*core.FlowFieldsResponse, error)
    QueryEventData(ctx context.Context, req *core.QueryRequest) (*core.QueryResponse, error)
    GetEventDetail(ctx context.Context, req *core.DetailRequest) (*core.DetailResponse, error)

    // 统计分析
    GetDurationStats(ctx context.Context, req *core.StatsRequest) (*core.DurationStats, error)
    GetStatusStats(ctx context.Context, req *core.StatsRequest) (*core.StatusStats, error)
    GetTrendStats(ctx context.Context, req *core.StatsRequest) (*core.TrendStats, error)
    GetNodeStats(ctx context.Context, req *core.StatsRequest) (*core.NodeStats, error)
    GetUserStats(ctx context.Context, req *core.StatsRequest) (*core.UserStats, error)
    GetOrgStats(ctx context.Context, req *core.StatsRequest) (*core.OrgStats, error)
    GetPendingStats(ctx context.Context, req *core.StatsRequest) (*core.PendingStats, error)

    // 资源管理
    Close() error
}

// SkylarkSDK - 统一的 SDK 接口
type SkylarkSDK interface {
    SkylarkEngine
    SkylarkQuery
}
```

**扩展 `engine/config.go`：**

```go
type Config struct {
    // 原有配置
    Cache *cache.Config
    Flows *flows.Config
    Forms *forms.Config

    // 新增配置
    LocalDB      sqlx.SqlConn        // 本地数据库（存储配置）
    Platform     *platform.Config    // 平台管理配置
    Event        *event.Config       // 事件管理配置
    Mapping      *mapping.Config     // 映射管理配置
    Query        *query.Config       // 查询配置
    Stats        *stats.Config       // 统计配置
}
```

**扩展 `engine/handler.go`：**

```go
type skylarkSDK struct {
    // 原有管理器
    cache *cache.SkylarkCache
    flows flows.SkylarkFlowRegistry
    forms forms.SkylarkFormRegistry

    // 新增管理器
    platform platform.Manager
    event    event.Manager
    mapping  mapping.Manager
    query    query.Manager
    stats    stats.Manager
}

func NewSkylarkSDK(config *Config, redisClient *redis.Redis) (SkylarkSDK, error) {
    // 初始化缓存
    cacheInstance, err := cache.NewSkylarkCache(config.Cache, redisClient)
    if err != nil {
        return nil, err
    }

    // 初始化原有管理器
    flowsRegistry, err := flows.NewSkylarkFlowRegistry(config.Flows, cacheInstance)
    if err != nil {
        return nil, err
    }

    formsRegistry, err := forms.NewSkylarkFormRegistry(config.Forms, cacheInstance)
    if err != nil {
        return nil, err
    }

    // 初始化新管理器（按依赖顺序）
    platformMgr := platform.NewManager(config.Platform, config.LocalDB)

    eventMgr := event.NewManager(config.Event, config.LocalDB, platformMgr.GetRemoteDB)

    mappingMgr := mapping.NewManager(config.Mapping, config.LocalDB, cacheInstance)

    queryMgr := query.NewManager(config.Query, config.LocalDB,
                                  platformMgr.GetRemoteDB,
                                  eventMgr.GetWithFields,
                                  mappingMgr.ListOrgMappings,
                                  cacheInstance)

    statsMgr := stats.NewManager(config.Stats, config.LocalDB,
                                  platformMgr.GetRemoteDB,
                                  eventMgr.GetWithFields,
                                  mappingMgr.ListOrgMappings,
                                  cacheInstance)

    return &skylarkSDK{
        cache:    cacheInstance,
        flows:    flowsRegistry,
        forms:    formsRegistry,
        platform: platformMgr,
        event:    eventMgr,
        mapping:  mappingMgr,
        query:    queryMgr,
        stats:    statsMgr,
    }, nil
}

// 实现 SkylarkEngine 接口（委托给原有管理器）
func (s *skylarkSDK) CreateFlow(ctx context.Context, app string, flowID int64, userID int64,
                                authHeader string, data map[string]core.TypedValue) error {
    return s.flows.CreateFlow(ctx, app, flowID, userID, authHeader, data)
}

// ... 其他 SkylarkEngine 方法

// 实现 SkylarkQuery 接口（委托给新管理器）
func (s *skylarkSDK) CreatePlatformConfig(ctx context.Context, config *core.PlatformConfig) error {
    return s.platform.Create(ctx, config)
}

func (s *skylarkSDK) QueryEventData(ctx context.Context, req *core.QueryRequest) (*core.QueryResponse, error) {
    return s.query.QueryEventData(ctx, req)
}

// ... 其他 SkylarkQuery 方法

func (s *skylarkSDK) Close() error {
    return s.platform.Close()
}
```

---

### 阶段五：依赖管理与编译验证（待执行）

#### 5.1 更新 go.mod

**检查并添加缺失的依赖：**

```bash
cd D:\Documents\Code_git\go-skylark
go mod tidy
```

**预期新增依赖（来自 skylarkq）：**
- `github.com/lib/pq` - PostgreSQL 数组支持
- 其他 go-zero 相关依赖（可能已存在）

#### 5.2 编译验证

```bash
# 编译所有包
go build ./...

# 检查语法错误
go vet ./...

# 格式化代码
go fmt ./...
```

#### 5.3 修复编译错误

**常见问题：**

1. **未导出的类型或方法：**
   - 检查所有从 skylarkq 复制的类型是否首字母大写
   - 确保需要对外暴露的方法都是导出的

2. **循环依赖：**
   - 检查 `core` 包是否被 `internal` 包正确引用
   - 确保依赖关系是单向的：`internal` → `core`

3. **接口不匹配：**
   - 确保所有 Manager 正确实现了对应接口
   - 检查方法签名是否一致

---

### ⏳ 阶段六：领域驱动设计重构（待执行）

#### 6.1 问题概述

go-skylark 的 Core 层目前**违反了领域驱动设计原则**，主要表现为：

1. ❌ **Core 层包含数据库标签**：50+ 个 `db:` 标签散布在 7 个 Core 文件中
2. ❌ **缺少数据模型转换层**：所有 Internal 子包缺失 `model.go` 文件
3. ❌ **Core 层使用框架类型**：`sql.NullString`、`sql.NullTime` 出现在领域模型中

#### 6.2 详细问题清单

**问题 1：Core 层包含 db 标签**

| 文件 | 问题结构体 | db 标签数量 | 影响 |
|------|-----------|-----------|------|
| `core/platform.go` | `PlatformConfig` | 11 个 | 与 ORM 耦合 |
| `core/event.go` | `EventConfig` | 12 个 | 与 ORM 耦合 |
| `core/field.go` | `FieldConfig` | 8 个 | 与 ORM 耦合 |
| `core/mapping.go` | `OrgMapping` | 6 个 | 与 ORM 耦合 |
| `core/flow.go` | `FlowInfo`、`FieldMetadata` | 5 个 | 与 ORM 耦合 |
| `core/stats.go` | `DurationStats`、`EventPendingStats` | 8 个 | 与 ORM 耦合 |
| **总计** | - | **50+ 个** | 严重违反 DDD |

**问题 2：缺失 model.go 文件**

| Internal 子包 | 是否有 model.go | 需要创建 |
|--------------|----------------|---------|
| `internal/cache/` | ❌ | ✅ |
| `internal/event/` | ❌ | ✅ |
| `internal/flows/` | ❌ | ✅ |
| `internal/forms/` | ❌ | ✅ |
| `internal/mapping/` | ❌ | ✅ |
| `internal/platform/` | ❌ | ✅ |
| `internal/query/` | ❌ | ✅ |
| `internal/stats/` | ❌ | ✅ |

#### 6.3 重构目标

```
当前架构（有问题）：
┌─────────────────────────────────────┐
│  Engine Layer (对外接口)             │
├─────────────────────────────────────┤
│  Internal Layer (Manager 实现)      │  ← 直接使用 Core 结构体扫描数据库
├─────────────────────────────────────┤
│  Core Layer (领域模型 + ORM 模型)   │  ← ❌ 包含 db: 标签（混合职责）
└─────────────────────────────────────┘

目标架构（正确）：
┌─────────────────────────────────────┐
│  Engine Layer (对外接口)             │
├─────────────────────────────────────┤
│  Internal Layer                     │
│  ├─ handler.go (业务逻辑)           │
│  ├─ model.go (数据模型 + db标签)    │  ← ✅ 分离清晰
│  └─ helpers.go (转换函数)           │
├─────────────────────────────────────┤
│  Core Layer (纯领域模型)            │  ← ✅ 无框架依赖
└─────────────────────────────────────┘
```

#### 6.4 分阶段重构计划

##### Phase 1：创建 model.go 文件

**工作量**：8 个子包 × 1-2 小时 = 1-2 天

**任务清单**：

- [ ] `internal/platform/model.go`
  - 创建 `PlatformConfigModel` 结构体
  - 迁移 `core.PlatformConfig` 的 db 标签
  - 实现 `ToDomain()` 方法

- [ ] `internal/event/model.go`
  - 创建 `EventConfigModel` 结构体
  - 迁移 `core.EventConfig` 的 db 标签
  - 实现 `ToDomain()` 方法

- [ ] `internal/mapping/model.go`
  - 创建 `OrgMappingModel` 结构体
  - 迁移 `core.OrgMapping` 的 db 标签
  - 实现 `ToDomain()` 方法

- [ ] `internal/flows/model.go`
  - 创建 `FlowInfoModel`、`FieldMetadataModel`
  - 迁移 `core.FlowInfo`、`core.FieldMetadata` 的 db 标签

- [ ] `internal/forms/model.go`
  - 创建表单相关 Model 结构体

- [ ] `internal/query/model.go`
  - 创建查询结果 Model 结构体

- [ ] `internal/stats/model.go`
  - 创建 `DurationStatsModel`、`EventPendingStatsModel`
  - 迁移 `core.DurationStats`、`core.EventPendingStats` 的 db 标签

- [ ] `internal/cache/model.go`（如有需要）

**示例代码**：

```go
// internal/platform/model.go
package platform

import (
	"database/sql"
	"time"
	"github.com/your-org/go-skylark/core"
)

// PlatformConfigModel 是数据库查询专用结构体
// ✅ 允许包含 db 标签
// ✅ 允许使用 sql.Null* 类型
type PlatformConfigModel struct {
	ID          string         `db:"id"`
	TenantID    string         `db:"tenant_id"`
	Host        string         `db:"host"`
	Token       string         `db:"token"`
	Description sql.NullString `db:"description"` // 可空字段
	CreatedBy   sql.NullString `db:"created_by"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   sql.NullTime   `db:"updated_at"`
}

// ToDomain 将数据库模型转换为领域模型
func (m *PlatformConfigModel) ToDomain() *core.PlatformConfig {
	return &core.PlatformConfig{
		ID:          m.ID,
		TenantID:    m.TenantID,
		Host:        m.Host,
		Token:       m.Token,
		Description: convertNullString(m.Description),
		CreatedBy:   convertNullString(m.CreatedBy),
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   convertNullTime(m.UpdatedAt),
	}
}

// 辅助转换函数
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

##### Phase 2：清理 Core 层

**工作量**：7 个文件 × 1 小时 = 1 天

**任务清单**：

- [ ] `core/platform.go` - 移除 `PlatformConfig` 的 db 标签
- [ ] `core/event.go` - 移除 `EventConfig` 的 db 标签
- [ ] `core/field.go` - 移除 `FieldConfig` 的 db 标签
- [ ] `core/mapping.go` - 移除 `OrgMapping` 的 db 标签
- [ ] `core/flow.go` - 移除 `FlowInfo`、`FieldMetadata` 的 db 标签
- [ ] `core/stats.go` - 移除 `DurationStats`、`EventPendingStats` 的 db 标签
- [ ] 将 `sql.NullString` 替换为 `*string`
- [ ] 将 `sql.NullTime` 替换为 `*time.Time`

**修改示例**：

```go
// core/platform.go (改进后)
package core

import "time"

// PlatformConfig 是纯粹的领域模型
// ❌ 不包含任何框架标签
// ❌ 不使用 sql.Null* 类型
type PlatformConfig struct {
	ID          string
	TenantID    string
	Host        string
	Token       string
	Description *string    // 使用指针表示可空
	CreatedBy   *string
	CreatedAt   time.Time
	UpdatedAt   *time.Time
}
```

##### Phase 3：更新 handler.go

**工作量**：8 个子包 × 2 小时 = 2 天

**任务清单**：

- [ ] 修改数据库查询代码，扫描到 `Model` 结构体
- [ ] 调用 `ToDomain()` 转换为 Core 对象
- [ ] 更新所有返回值
- [ ] 更新批量查询逻辑

**修改示例**：

```go
// internal/platform/handler.go

// 修改前
func (h *handler) GetPlatformConfig(ctx context.Context, tenantID string) (*core.PlatformConfig, error) {
	var config core.PlatformConfig // 直接扫描到 Core 对象
	err := h.db.QueryRowCtx(ctx, &config, getPlatformConfigSQL, tenantID)
	if err != nil {
		return nil, core.ErrNoPlatformConfig
	}
	return &config, nil
}

// 修改后
func (h *handler) GetPlatformConfig(ctx context.Context, tenantID string) (*core.PlatformConfig, error) {
	var model PlatformConfigModel // 扫描到 Model
	err := h.db.QueryRowCtx(ctx, &model, getPlatformConfigSQL, tenantID)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, core.ErrNoPlatformConfig
		}
		return nil, fmt.Errorf("查询平台配置失败: %w", err)
	}
	return model.ToDomain(), nil // 转换为 Core 对象
}

// 批量查询示例
func (h *handler) ListPlatformConfigs(ctx context.Context) ([]*core.PlatformConfig, error) {
	var models []PlatformConfigModel
	err := h.db.QueryRowsCtx(ctx, &models, listPlatformConfigsSQL)
	if err != nil {
		return nil, fmt.Errorf("查询平台配置列表失败: %w", err)
	}

	// 转换为领域对象
	configs := make([]*core.PlatformConfig, len(models))
	for i, model := range models {
		configs[i] = model.ToDomain()
	}
	return configs, nil
}
```

##### Phase 4：测试与验证

**工作量**：1 天

**任务清单**：

- [ ] 编译验证 `go build ./...`
- [ ] 运行测试 `go test ./...`
- [ ] 手动测试核心功能：
  - [ ] 平台配置管理
  - [ ] 事件配置管理
  - [ ] 流程查询
  - [ ] 统计分析
- [ ] 检查所有 Core 文件不包含 `db:` 标签
- [ ] 使用 grep 验证：`grep -r 'db:"' ./core`（应无结果）
- [ ] 使用 grep 验证：`grep -r 'sql\.Null' ./core`（应无结果）

#### 6.5 进度跟踪

| 阶段 | 预计工作量 | 状态 | 完成时间 | 备注 |
|------|-----------|------|---------|------|
| Phase 1: 创建 model.go | 1-2 天 | ⏳ 未开始 | - | 8 个子包 |
| Phase 2: 清理 Core 层 | 1 天 | ⏳ 未开始 | - | 7 个文件 |
| Phase 3: 更新 handler.go | 2 天 | ⏳ 未开始 | - | 8 个子包 |
| Phase 4: 测试与验证 | 1 天 | ⏳ 未开始 | - | 全面测试 |
| **总计** | **5-6 天** | - | - | 约 1 周 |

#### 6.6 验收标准

**必须满足**：
- ✅ Core 层所有结构体不包含 `db:` 标签
- ✅ Core 层不使用 `sql.Null*` 等框架类型
- ✅ 所有 Internal 子包都有 `model.go` 文件
- ✅ 所有 Manager 方法返回 Core 层对象
- ✅ `go build ./...` 编译通过
- ✅ 现有功能不受影响

**推荐满足**：
- 🟡 添加单元测试覆盖转换逻辑
- 🟡 更新 DEVELOPMENT.md 文档
- 🟡 添加代码注释说明架构改进

---

## 四、风险评估与应对

### 4.1 潜在风险

| 风险 | 影响 | 概率 | 应对措施 |
|-----|------|------|---------|
| 包路径替换遗漏 | 编译失败 | 中 | 使用正则全局搜索验证 |
| 接口不匹配 | 运行时错误 | 低 | 编译期类型检查 |
| 缓存接口适配错误 | 功能异常 | 中 | 单元测试验证 |
| 数据库连接池管理 | 资源泄漏 | 低 | 添加 Close 方法 |
| 依赖冲突 | 编译失败 | 低 | go mod tidy 解决 |
| 循环依赖 | 编译失败 | 中 | 严格遵循分层架构 |

---

## 五、时间估算

| 阶段 | 预计时间 | 实际时间 | 状态 | 备注 |
|-----|---------|---------|------|------|
| 阶段一：文件复制与结构调整 | 2 小时 | - | ✅ 已完成 | 手动操作 |
| 阶段二：批量替换导入路径 | 0.5 小时 | - | ✅ 已完成 | 自动化工具 |
| 阶段三：Core 文件优化 | 2 小时 | - | ✅ 已完成 | 注释与重组 |
| 阶段四：合并冲突和适配 | 3 小时 | - | ⏳ 待执行 | 核心工作 |
| 阶段五：依赖管理与编译验证 | 1 小时 | - | ⏳ 待执行 | 修复编译错误 |
| **阶段六：领域驱动设计重构** | **5-6 天** | - | ⏳ 待执行 | **架构改进（重要）** |
| 文档更新 | 1 小时 | - | ✅ 已完成 | DEVELOPMENT.md |
| **不含重构总计** | **~8.5 小时** | - | - | 约 1-2 个工作日 |
| **含重构总计** | **~7-8 天** | - | - | 约 1.5-2 周 |

---

## 六、后续优化建议

### 6.1 短期优化（1-2周）

1. **添加单元测试**
   - 为所有 Manager 添加单元测试
   - 测试覆盖率达到 70%+

2. **完善错误处理**
   - 统一错误码定义
   - 添加错误链追踪

3. **性能优化**
   - 数据库连接池调优
   - Redis 连接池调优
   - 批量查询优化

### 6.2 中期优化（1-2月）

1. **添加监控指标**
   - API 调用耗时
   - 数据库查询耗时
   - 缓存命中率

2. **日志系统**
   - 使用结构化日志（logx）
   - 添加分布式追踪（OpenTelemetry）

3. **配置中心集成**
   - 支持 Nacos/Consul 配置中心
   - 动态配置热更新

### 6.3 长期优化（3-6月）

1. **性能测试**
   - 压力测试
   - 并发测试
   - 性能基准测试

2. **高可用设计**
   - Redis 哨兵/集群支持
   - 数据库主从切换
   - 降级策略

3. **API 版本管理**
   - 支持多版本 API
   - 向后兼容策略

---

## 七、检查清单

### 整合前检查

- [ ] 备份当前代码（创建 Git 分支）
- [ ] 确认 skylarkq 代码位置正确
- [ ] 确认 go.mod 中的 module 路径
- [ ] 准备好本地数据库和 Redis

### 整合中检查

- [ ] 所有文件复制完成
- [ ] 导入路径批量替换完成
- [ ] 错误定义合并完成
- [ ] 字段类型系统整合完成
- [ ] 缓存接口适配完成
- [ ] Engine 接口扩展完成
- [ ] Manager 初始化顺序正确

### 整合后检查

- [ ] `go build ./...` 编译通过
- [ ] `go vet ./...` 无警告
- [ ] `go fmt ./...` 格式正确
- [ ] `go mod tidy` 依赖整理完成
- [ ] 数据库迁移脚本测试通过
- [ ] README.md 更新完成
- [ ] 架构文档创建完成
- [ ] 迁移指南创建完成

---

## 八、联系与支持

如在整合过程中遇到问题，可参考：

1. **原项目文档：**
   - `docs/SKYLARKQ_DEVELOPMENT.md`（659行详细文档）
   - `docs/MULTI_EVENT_STATS.md`（多事件统计说明）

2. **代码注释：**
   - 所有核心函数都有详细注释
   - 复杂逻辑有算法说明

3. **Git 历史：**
   - 查看原 skylarkq 的提交历史
   - 理解功能演进过程

---

**文档版本：** v1.0
**创建日期：** 2025-11-07
**预计完成日期：** 2025-11-09
**负责人：** Claude Code

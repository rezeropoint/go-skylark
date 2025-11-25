# Go-Skylark

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Go-Skylark 是一个用于对接 Skylark 低代码平台的企业级 Go SDK，提供**系统管理**和**业务流程**双引擎架构，支持组织管理、用户管理、流程管理、事件查询、统计分析等完整功能。

## ✨ 核心特性

### 双引擎架构

**Admin Engine（系统管理引擎）** - 14个接口
- ✅ **组织管理**：创建、删除、更新组织，查询同步状态
- ✅ **用户管理**：创建用户，查询同步状态
- ✅ **ID映射管理**：绑定/解绑组织和用户的本地ID与远程ID
- ✅ **组织成员管理**：添加/移除成员

**Engine Layer（业务流程引擎）** - 37个接口
- ✅ **流程管理**：创建流程、更新状态、查询详情、搜索、审批历史、终止流程（13个接口）
- ✅ **平台配置**：管理远程 Skylark 平台连接（5个接口）
- ✅ **事件配置**：事件与字段配置聚合管理（5个接口）
- ✅ **组织映射**：多租户组织权限映射（5个接口）
- ✅ **远程查询**：查询事件数据、Flow列表、Flow字段（4个接口）
- ✅ **统计分析**：7种统计维度（时长、状态、趋势、节点、用户、组织、待处理）

### 企业级特性

- ✅ **职责分离**：Admin负责系统管理，Engine负责业务流程，独立初始化
- ✅ **三层架构**：Engine/Admin → Internal → Core（DDD 领域驱动设计）
- ✅ **ID映射透明化**：用户使用本地ID，SDK自动转换为远程ID
- ✅ **多租户支持**：远程数据库连接池管理
- ✅ **缓存优化**：Redis 多级缓存（Flow信息、用户名、组织映射、统计结果）
- ✅ **组织权限过滤**：基于组织映射的数据权限控制
- ✅ **事务支持**：聚合根原子性操作
- ✅ **性能优化**：批量查询、enrichment 模式（API调用减少95%+）
- ✅ **错误处理**：36个预定义错误类型

## 📦 安装

```bash
go get github.com/rezeropoint/go-skylark
```

## 🚀 快速开始

### 1. 初始化 Admin Engine（系统管理）

```go
package main

import (
    "context"
    "github.com/rezeropoint/go-skylark/admin"
    "github.com/rezeropoint/go-skylark/core"
)

func main() {
    // 创建 Admin Engine 配置
    adminConfig := &admin.Config{
        LocalDB: localDBConn,   // 本地数据库连接
        Cache:   cacheInterface, // Redis 缓存接口
    }

    // 初始化 Admin Engine
    adminEngine, err := admin.NewAdminEngine(adminConfig)
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 创建组织（自动维护本地ID↔远程ID映射）
    org := &core.Organization{
        TenantID:      "tenant_001",
        LocalID:       "org_local_001",
        Name:          "技术部",
        ParentLocalID: nil, // 根组织
    }
    err = adminEngine.CreateOrganization(ctx, org)

    // 创建用户（自动维护本地ID↔远程ID映射）
    user := &core.User{
        TenantID:  "tenant_001",
        LocalID:   "user_local_001",
        Name:      "张三",
        Email:     "zhangsan@example.com",
        OrgLocalID: "org_local_001",
    }
    err = adminEngine.CreateUser(ctx, user)

    // 绑定已存在的远程组织
    err = adminEngine.BindOrganization(ctx, "tenant_001", "org_local_002", "remote_org_456")

    // 添加组织成员（使用本地用户ID）
    err = adminEngine.AddMember(ctx, "tenant_001", "org_local_001", "user_local_001")
}
```

### 2. 初始化 Engine（业务流程）

```go
import (
    "github.com/rezeropoint/go-skylark/engine"
)

func main() {
    // 创建 Engine 配置
    engineConfig := &engine.Config{
        BaseURL:    "https://your-skylark-instance.com",
        AuthHeader: "your-auth-header",
        LocalDB:    localDBConn,
        Cache:      cacheInterface,
        FlowInfoCacheTTL: 3600, // Flow 信息缓存 TTL（秒），默认 1 小时
    }

    // 初始化 Engine（业务流程引擎）
    skylark, err := engine.NewSkylarkEngine(engineConfig)
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 1. 配置平台连接
    platformConfig := &core.PlatformConfig{
        TenantID:    "tenant_001",
        Host:        "postgresql://remote-db:5432",
        DBName:      "skylark_db",
        Username:    "admin",
        Password:    "password",
        Namespace:   stringPtr("default"),
        Description: stringPtr("生产环境数据库"),
    }
    err = skylark.CreatePlatformConfig(ctx, platformConfig)

    // 2. 配置事件和字段
    eventConfig := &core.EventConfigWithFields{
        EventConfig: core.EventConfig{
            TenantID:     "tenant_001",
            FlowID:       "flow_123",
            Name:         "采购申请",
            OrgFieldName: stringPtr("department"),
        },
        Fields: []core.FieldConfig{
            {FieldName: "title", FieldType: "TextField", IsRequired: true},
            {FieldName: "amount", FieldType: "NumberField", IsRequired: true},
        },
    }
    err = skylark.CreateEventWithFields(ctx, eventConfig)

    // 3. 配置组织映射（数据权限过滤）
    mapping := &core.OrgMapping{
        TenantID:       "tenant_001",
        EventConfigID:  "event_001",
        LocalOrgID:     "org_local_001", // 本地组织ID
        RemoteOrgValue: "技术部",         // 远程数据库中的组织字段值
    }
    err = skylark.CreateOrgMapping(ctx, mapping)

    // 4. 创建流程（使用本地ID，SDK自动转换）
    flowData := map[string]core.TypedValue{
        "title":  {Type: "string", Value: "采购笔记本电脑"},
        "amount": {Type: "number", Value: 5000},
    }
    err = skylark.CreateFlow(ctx, "Bearer token", "flow_template_id", flowData)

    // 5. 查询流程（自动过滤用户可见的组织数据）
    queryReq := &core.QueryRequest{
        TenantID:       "tenant_001",
        EventConfigIDs: []string{"event_001"},
        UserLocalOrgIDs: []string{"org_local_001"}, // 用户所属组织（本地ID）
        Conditions: []core.QueryCondition{
            {Field: "status", Operator: "=", Value: "approved"},
        },
        Page:     1,
        PageSize: 20,
    }
    result, err := skylark.QueryEventData(ctx, queryReq)

    // 6. 获取统计数据
    statsReq := &core.StatsRequest{
        TenantID:       "tenant_001",
        EventConfigIDs: []string{"event_001"}, // 支持多事件聚合
        UserLocalOrgIDs: []string{"org_local_001"},
        StartDate:      "2025-01-01",
        EndDate:        "2025-01-31",
    }
    stats, err := skylark.GetDurationStats(ctx, statsReq)

    // 7. 获取用户待办任务（自动补充 flow_id 和 flow_title）
    assignments, err := skylark.GetUserAssignments(ctx, "tenant_001", "Bearer token")
    // assignments 包含 FlowID 和 FlowTitle（通过 enrichment 自动填充）
}

func stringPtr(s string) *string { return &s }
```

## 🏗️ 架构设计

### 双引擎架构

```
对外接口层（两个独立入口）
┌─────────────────────────────────┐  ┌─────────────────────────────────┐
│ Admin Engine (系统管理引擎)      │  │ Engine Layer (业务流程引擎)      │
│ - 组织管理 (5)                   │  │ - 流程管理 (13)                  │
│ - 用户管理 (2)                   │  │ - 平台配置 (5)                   │
│ - 组织ID映射管理 (2)             │  │ - 事件配置 (5)                   │
│ - 用户ID映射管理 (2)             │  │ - 组织映射 (5)                   │
│ - 组织成员管理 (2)               │  │ - 远程查询 (4)                   │
│ - 平台配置 (1, 通过Internal访问)  │  │ - 统计分析 (7)                   │
└─────────────────────────────────┘  └─────────────────────────────────┘
             ↓                                    ↓
┌──────────────────────────────────────────────────────────────────────┐
│ Internal Layer (业务逻辑层 - Manager 模式)                            │
│ - platform/      平台配置管理 + 远程数据库连接池                       │
│ - organization/  组织管理 + 本地ID↔远程ID映射                          │
│ - user/          用户管理 + 本地ID↔远程ID映射                          │
│ - event/         事件配置管理（聚合根）                                │
│ - mapping/       组织映射管理（权限过滤）                              │
│ - query/         远程查询引擎（Journey聚合、组织过滤）                 │
│ - stats/         统计分析引擎（多事件聚合、缓存）                      │
│ - flows/         流程管理（Enrichment性能优化、13个接口）              │
│ - forms/         表单管理（写操作、字段类型转换）                      │
│ - cache/         缓存管理                                             │
└──────────────────────────────────────────────────────────────────────┘
             ↓
┌──────────────────────────────────────────────────────────────────────┐
│ Core Layer (领域模型层 - 纯 Go 类型)                                  │
│ - 领域模型定义（Organization、User、EventConfig、OrgMapping 等）      │
│ - 函数类型注入（GetRemoteDBFunc、GetPlatformConfigFunc 等）           │
│ - 业务规则函数（IsOptionField、IsImageField）                         │
│ - 错误类型定义（36个预定义错误）                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### 核心设计原则

1. **职责分离**：Admin 负责系统管理，Engine 负责业务流程，两者独立初始化
2. **依赖方向**：Engine/Admin → Internal → Core（禁止反向依赖）
3. **Core 层纯净**：无 `db` 标签、无 `sql.Null*` 类型、无框架依赖
4. **Manager 解耦**：通过 Core 层函数类型注入避免循环依赖
5. **数据流向**：`数据库 → DataModel (model.go) → DomainModel (core/) → 用户`
6. **ID 映射透明化**：用户使用本地 ID，SDK 内部自动转换为远程 ID

详细架构设计和开发规范请参考 [CLAUDE.md](./CLAUDE.md)

## 📋 核心接口

### Admin Engine 接口

```go
type AdminEngine interface {
    // 组织管理（5个）
    CreateOrganization(ctx context.Context, org *Organization) error
    CreateSubOrganization(ctx context.Context, org *Organization) error
    DeleteOrganization(ctx context.Context, tenantID, localID string) error
    UpdateOrganization(ctx context.Context, org *Organization) error
    GetOrgSyncStatus(ctx context.Context, tenantID, localID string) (*OrgSyncStatus, error)

    // 用户管理（2个）
    CreateUser(ctx context.Context, user *User) error
    GetUserSyncStatus(ctx context.Context, tenantID, localID string) (*UserSyncStatus, error)

    // 组织ID映射管理（2个）
    BindOrganization(ctx context.Context, tenantID, localID, remoteID string) error
    UnbindOrganization(ctx context.Context, tenantID, localID string) error

    // 用户ID映射管理（2个）
    BindUser(ctx context.Context, tenantID, localID, remoteID string) error
    UnbindUser(ctx context.Context, tenantID, localID string) error

    // 组织成员管理（2个）
    // 添加/移除单个成员，SDK内部自动将本地用户ID转换为远程用户ID
    AddMember(ctx context.Context, tenantID, orgLocalID, userLocalID string) error
    RemoveMember(ctx context.Context, tenantID, orgLocalID, userLocalID string) error
}
```

### Engine Layer 接口（部分）

```go
type SkylarkEngine interface {
    // 流程管理（13个）
    CreateFlow(ctx, authHeader, templateID string, data map[string]TypedValue) error
    UpdateFlowJourneyStatus(ctx, authHeader, journeyID, status string) error
    GetFlowJourneyBySN(ctx, tenantID, authHeader, serialNumber string) (*Journey, error)
    GetFlowJourneyAssignments(ctx, tenantID, authHeader, journeyID string) ([]*Assignment, error)
    GetFlowJourneyDetail(ctx, tenantID, authHeader, journeyID string) (*JourneyDetail, error)
    GetFlowDetail(ctx, tenantID, authHeader, flowID string) (*FlowDetail, error)
    GetUserAssignments(ctx, tenantID, authHeader string) ([]*Assignment, error) // 自动enrichment
    GetProposedJourneys(ctx, tenantID, authHeader string, req *ProposedJourneysRequest) ([]*Journey, error)
    SearchJourneys(ctx, tenantID, authHeader string, req *SearchJourneysRequest) (*SearchJourneysResult, error)
    GetJourneyMoments(ctx, tenantID, authHeader, journeyID string) ([]*Moment, error)
    GetCurrentProcessingUsers(ctx, tenantID, authHeader, journeyID string) ([]*User, error)
    AbortJourney(ctx, tenantID, authHeader, journeyID string) error
    CreateFormRow(ctx, authHeader, formID string, data map[string]TypedValue) error

    // 平台配置（5个）
    CreatePlatformConfig(ctx, config *PlatformConfig) error
    GetPlatformConfig(ctx, tenantID string) (*PlatformConfig, error)
    UpdatePlatformConfig(ctx, config *PlatformConfig) error
    DeletePlatformConfig(ctx, tenantID string) error
    ValidatePlatformConfig(ctx, tenantID string) error

    // 事件配置（5个）
    CreateEventWithFields(ctx, config *EventConfigWithFields) error
    UpdateEventWithFields(ctx, config *EventConfigWithFields) error
    GetEventWithFields(ctx, eventID string) (*EventConfigWithFields, error)
    ListEventWithFields(ctx, tenantID string) ([]*EventConfigWithFields, error)
    DeleteEvent(ctx, eventID string) error

    // 组织映射（5个）
    CreateOrgMapping(ctx, mapping *OrgMapping) error
    GetOrgMapping(ctx, mappingID string) (*OrgMapping, error)
    ListOrgMappings(ctx, tenantID string) ([]*OrgMapping, error)
    UpdateOrgMapping(ctx, mapping *OrgMapping) error
    DeleteOrgMapping(ctx, mappingID string) error

    // 远程查询（4个）
    QueryEventData(ctx, req *QueryRequest) (*QueryResult, error)
    GetEventDetail(ctx, req *DetailRequest) (*EventDetail, error)
    GetFlowList(ctx, tenantID, namespace string) ([]*FlowInfo, error)
    GetFlowFields(ctx, tenantID, flowID string) ([]*FieldInfo, error)

    // 统计分析（7个）
    GetDurationStats(ctx, req *StatsRequest) (*DurationStatsResult, error)
    GetStatusStats(ctx, req *StatsRequest) (*StatusStatsResult, error)
    GetTrendStats(ctx, req *TrendStatsRequest) (*TrendStatsResult, error)
    GetNodeStats(ctx, req *StatsRequest) (*NodeStatsResult, error)
    GetUserStats(ctx, req *StatsRequest) (*UserStatsResult, error)
    GetOrgStats(ctx, req *StatsRequest) (*OrgStatsResult, error)
    GetPendingStats(ctx, req *StatsRequest) (*PendingStatsResult, error)
}
```

## 🛠️ 开发

### 常用命令

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

# 代码格式化
go fmt ./...

# 静态分析
go vet ./...

# 整理依赖
go mod tidy

# 检查循环依赖
go mod graph | grep 'go-skylark'
```

### 项目结构

```
go-skylark/
├── admin/                 # Admin Engine（系统管理引擎）
│   ├── admin.go           # AdminEngine 接口定义
│   ├── handler.go         # 接口实现
│   └── config.go          # Admin 配置
├── engine/                # Engine Layer（业务流程引擎）
│   ├── engine.go          # SkylarkEngine 接口定义
│   ├── handler.go         # 接口实现
│   └── config.go          # Engine 配置
├── internal/              # Internal Layer（Manager 模式）
│   ├── platform/          # 平台配置管理 + 远程数据库连接池
│   ├── organization/      # 组织管理 + 本地ID↔远程ID映射
│   ├── user/              # 用户管理 + 本地ID↔远程ID映射
│   ├── event/             # 事件配置管理（聚合根）
│   ├── mapping/           # 组织映射管理（权限过滤）
│   ├── query/             # 远程查询引擎（Journey聚合、组织过滤）
│   ├── stats/             # 统计分析引擎（多事件聚合、缓存）
│   ├── flows/             # 流程管理（Enrichment性能优化）
│   │   ├── README.md      # Flows 模块详细文档（性能优化机制）
│   │   └── enrichment.go  # Assignment enrichment（API调用减少95%+）
│   ├── forms/             # 表单管理（写操作、字段类型转换）
│   ├── cache/             # 缓存管理
│   │   └── README.md      # 缓存模块详细文档
│   └── images/            # 图片处理（Base64 → 七牛云）
├── core/                  # Core Layer（领域模型层）
│   ├── event.go           # 事件配置模型
│   ├── platform.go        # 平台配置模型 + 函数类型注入
│   ├── mapping.go         # 组织映射模型
│   ├── organization.go    # 组织模型
│   ├── user.go            # 用户模型
│   ├── query.go           # 查询请求模型
│   ├── stats.go           # 统计请求模型
│   ├── field.go           # 字段类型系统
│   ├── errors.go          # 错误定义
│   └── var.go             # 常量定义（36个预定义错误）
├── CLAUDE.md              # Claude Code 开发指引（架构原则、开发规范、常用命令）
└── README.md              # 项目概览（本文档）
```

## 🚀 性能优化设计

### Flows 模块 Enrichment 机制

**问题**：GetUserAssignments 返回的 assignment 列表缺少 `flow_id` 和 `flow_title`

**解决方案**（`internal/flows/enrichment.go`）：
1. 提取唯一的 `journey_id` → 批量查询 `journey_id → flow_id` 映射（远程数据库）
2. 提取唯一的 `flow_id` → 批量查询 Flow 信息（Redis 缓存优先）
3. 合并数据，填充 `assignment.FlowID` 和 `assignment.FlowTitle`

**效果**：
- API 调用：从 54 次降至 1-4 次（减少 **95%+**）
- 总耗时：缓存命中时约 **60ms**，未命中时约 **360ms**（优化前 **2.8秒**）

**配置**：
```go
engineConfig := &engine.Config{
    FlowInfoCacheTTL: 3600, // Flow 信息缓存 TTL（秒），默认 3600
}
```

详细文档：[internal/flows/README.md](./internal/flows/README.md)

### 多级缓存策略

| 数据类型 | 缓存键 | TTL | 使用模块 |
|---------|--------|-----|---------|
| Flow 列表 | `skylark:flows:{tenant}:{namespace}` | 2分钟 | Query |
| Flow 字段 | `skylark:flow_fields:{tenant}:{flow_id}` | 2分钟 | Query |
| Flow 信息（API） | `skylark:flow:api:{tenant}:{flow_id}` | **可配置**（默认1小时） | **Flows Enrichment** |
| 用户名 | `skylark:users:{tenant}:{user_id}` | 24小时 | User |
| 组织映射 | `skylark:mapping:{id}` | 30天 | Mapping |
| 统计结果 | `skylark:stats:{type}:{tenant}:{event}:{hash}` | 5分钟 | Stats |

## 📚 文档

- **[CLAUDE.md](./CLAUDE.md)** - Claude Code 开发指引，包含：
  - 快速参考（常用命令、架构原则、常见陷阱）
  - 三层架构与依赖流向
  - 关键设计模式（函数注入解耦、领域模型分离、聚合管理）
  - Manager 开发规范（文件组织、model.go 规范）
  - 性能优化设计模式（批量查询、多级缓存、Enrichment）

- **[internal/flows/README.md](./internal/flows/README.md)** - Flows 模块详细文档
  - 性能优化机制（Enrichment）
  - 批量查询策略
  - 缓存策略

- **[internal/cache/README.md](./internal/cache/README.md)** - 缓存模块详细文档

- **[internal/stats/MULTI_EVENT_STATS.md](./internal/stats/MULTI_EVENT_STATS.md)** - 多事件统计设计文档

## 🔧 依赖

- [go-zero](https://github.com/zeromicro/go-zero) - 微服务框架（sqlx、logx）
- [go-redis](https://github.com/redis/go-redis) - Redis 客户端
- [lib/pq](https://github.com/lib/pq) - PostgreSQL 驱动
- Go 1.19+

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

在提交代码前，请确保：
1. 遵循 [CLAUDE.md](./CLAUDE.md) 中的开发规范
2. 所有测试通过：`go test ./...`
3. 代码已格式化：`go fmt ./...`
4. 通过静态分析：`go vet ./...`

## 📄 许可证

[MIT License](LICENSE)

## 🔗 相关链接

- [问题反馈](https://github.com/rezeropoint/go-skylark/issues)
- [版本历史](https://github.com/rezeropoint/go-skylark/releases)

# Go-Skylark

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.19-blue)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

Go-Skylark 是一个用于对接 Skylark 低代码平台的企业级 Go SDK，提供流程管理、事件查询、统计分析等完整功能。

## ✨ 核心特性

### 写操作
- ✅ **流程管理**：创建流程、更新流程状态
- ✅ **表单管理**：创建表单行、字段类型转换
- ✅ **图片处理**：Base64 自动上传七牛云

### 读操作与查询
- ✅ **事件查询**：支持复杂条件查询、组织权限过滤
- ✅ **统计分析**：7 种统计维度（时长、状态、趋势、节点、用户、组织、待处理）
- ✅ **远程数据**：查询 Flow 列表、Flow 字段、Flow 详情

### 配置管理
- ✅ **平台配置**：管理远程 Skylark 平台连接
- ✅ **事件配置**：事件与字段配置（聚合管理）
- ✅ **组织映射**：多租户组织映射管理

### 企业级特性
- ✅ **三层架构**：Engine → Internal → Core（DDD 领域驱动设计）
- ✅ **多租户支持**：远程数据库连接池管理
- ✅ **缓存优化**：Redis 多级缓存（配置、字段映射、统计结果）
- ✅ **组织权限**：基于组织映射的数据权限过滤
- ✅ **事务支持**：聚合根原子性操作
- ✅ **错误处理**：36 个预定义错误类型

## 📦 安装

```bash
go get github.com/your-username/go-skylark
```

## 🚀 快速开始

### 基础使用

```go
package main

import (
    "context"
    "github.com/your-username/go-skylark/engine"
    "github.com/your-username/go-skylark/core"
)

func main() {
    // 1. 创建引擎配置
    config := &engine.Config{
        BaseURL:    "https://your-skylark-instance.com",
        AuthHeader: "your-auth-header",
        LocalDB:    localDBConn,   // 本地数据库连接
        Cache:      cacheInterface, // Redis 缓存接口
    }

    // 2. 初始化 Skylark 引擎
    skylark, err := engine.NewSkylarkEngine(config)
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 3. 创建流程
    flowData := map[string]core.TypedValue{
        "title": {Type: "string", Value: "测试流程"},
        "amount": {Type: "number", Value: 1000},
    }
    err = skylark.CreateFlow(ctx, "Bearer token", "flow_template_id", flowData)

    // 4. 查询事件数据
    queryReq := &core.QueryRequest{
        TenantID:       "tenant_001",
        EventConfigIDs: []string{"event_001"},
        Conditions: []core.QueryCondition{
            {Field: "status", Operator: "=", Value: "approved"},
        },
        Page: 1,
        PageSize: 20,
    }
    result, err := skylark.QueryEventData(ctx, queryReq)

    // 5. 获取统计数据
    statsReq := &core.StatsRequest{
        TenantID:       "tenant_001",
        EventConfigIDs: []string{"event_001"},
        StartDate:      "2025-01-01",
        EndDate:        "2025-01-31",
    }
    stats, err := skylark.GetDurationStats(ctx, statsReq)
}
```

### 配置管理示例

```go
// 创建平台配置
platformConfig := &core.PlatformConfig{
    TenantID:    "tenant_001",
    Host:        "postgresql://remote-db:5432",
    DBName:      "skylark_db",
    Username:    "admin",
    Password:    "password",
    Description: stringPtr("生产环境数据库"),
}
err := skylark.CreatePlatformConfig(ctx, platformConfig)

// 创建事件配置
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
err := skylark.CreateEventWithFields(ctx, eventConfig)

// 创建组织映射
mapping := &core.OrgMapping{
    TenantID:       "tenant_001",
    EventConfigID:  "event_001",
    LocalOrgID:     "org_local_001",
    RemoteOrgValue: "部门A",
}
err := skylark.CreateOrgMapping(ctx, mapping)
```

## 🏗️ 架构设计

### 三层架构

```
┌─────────────────────────────────────────┐
│  Engine Layer (对外接口)                │  ← 用户直接调用
│  - 聚合所有 Manager                      │
│  - 统一接口入口                          │
├─────────────────────────────────────────┤
│  Internal Layer (业务逻辑)              │  ← Manager 模式
│  - platform/  平台配置管理               │
│  - event/     事件配置管理               │
│  - mapping/   组织映射管理               │
│  - query/     远程查询引擎               │
│  - stats/     统计分析引擎               │
│  - flows/     流程管理（写操作）         │
│  - forms/     表单管理（写操作）         │
│  - cache/     缓存管理                   │
├─────────────────────────────────────────┤
│  Core Layer (领域模型)                  │  ← 纯 Go 类型
│  - 领域模型定义                          │
│  - 业务规则函数                          │
│  - 错误类型定义                          │
└─────────────────────────────────────────┘
```

### 核心设计原则

1. **依赖方向**：Engine → Internal → Core（禁止反向依赖）
2. **Core 层纯净**：无 `db` 标签、无 `sql.Null*` 类型
3. **Manager 解耦**：通过函数类型注入避免循环依赖
4. **数据流向**：`数据库 → DataModel → DomainModel → 用户`

详细架构设计请参考 [DEVELOPMENT.md](./DEVELOPMENT.md)

## 📋 核心接口

```go
type SkylarkEngine interface {
    // 旧模块 - 写操作
    CreateFlow(ctx, authHeader, templateID string, data map[string]TypedValue) error
    CreateFormRow(ctx, authHeader, formID string, data map[string]TypedValue) error
    UpdateFlowJourneyStatus(ctx, authHeader, flowID, status string) error

    // 平台配置管理
    CreatePlatformConfig(ctx, config *PlatformConfig) error
    GetPlatformConfig(ctx, tenantID string) (*PlatformConfig, error)
    UpdatePlatformConfig(ctx, config *PlatformConfig) error
    DeletePlatformConfig(ctx, tenantID string) error
    ValidatePlatformConfig(ctx, tenantID string) error

    // 事件配置管理
    CreateEventWithFields(ctx, config *EventConfigWithFields) error
    UpdateEventWithFields(ctx, config *EventConfigWithFields) error
    GetEventWithFields(ctx, eventID string) (*EventConfigWithFields, error)
    ListEventConfigs(ctx, tenantID string) ([]*EventConfig, error)
    DeleteEventWithFields(ctx, eventID string) error

    // 组织映射管理
    CreateOrgMapping(ctx, mapping *OrgMapping) error
    GetOrgMapping(ctx, mappingID string) (*OrgMapping, error)
    ListOrgMappings(ctx, tenantID string) ([]*OrgMapping, error)
    UpdateOrgMapping(ctx, mapping *OrgMapping) error
    DeleteOrgMapping(ctx, mappingID string) error

    // 远程查询
    QueryEventData(ctx, req *QueryRequest) (*QueryResult, error)
    GetEventDetail(ctx, req *DetailRequest) (*EventDetail, error)
    GetFlowList(ctx, tenantID, namespace string) ([]*FlowInfo, error)
    GetFlowFields(ctx, tenantID, flowID string) ([]*FieldInfo, error)

    // 统计分析
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

### 构建项目

```bash
# 构建所有模块
go build ./...

# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/query -v

# 测试覆盖率
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 代码格式化
go fmt ./...

# 静态分析
go vet ./...

# 整理依赖
go mod tidy
```

### 项目结构

```
go-skylark/
├── core/              # 领域模型层（纯 Go 类型）
│   ├── event.go       # 事件配置模型
│   ├── platform.go    # 平台配置模型
│   ├── mapping.go     # 组织映射模型
│   ├── query.go       # 查询请求模型
│   ├── stats.go       # 统计请求模型
│   ├── field.go       # 字段类型系统
│   ├── errors.go      # 错误定义
│   └── var.go         # 常量定义
├── engine/            # 引擎层（对外接口）
│   ├── engine.go      # SkylarkEngine 接口定义
│   ├── handler.go     # 接口实现
│   └── config.go      # 引擎配置
├── internal/          # 内部实现层（Manager 模式）
│   ├── platform/      # 平台配置管理
│   ├── event/         # 事件配置管理
│   ├── mapping/       # 组织映射管理
│   ├── query/         # 远程查询引擎
│   ├── stats/         # 统计分析引擎
│   ├── flows/         # 流程管理（写操作）
│   ├── forms/         # 表单管理（写操作）
│   ├── cache/         # 缓存管理
│   └── images/        # 图片处理
├── CLAUDE.md          # Claude Code 开发指引
├── DEVELOPMENT.md     # 详细架构设计文档
└── README.md          # 项目概览（本文档）
```

## 📚 文档

- **[CLAUDE.md](./CLAUDE.md)** - Claude Code 开发指引，包含架构原则、开发规范、常用命令
- **[DEVELOPMENT.md](./DEVELOPMENT.md)** - 详细的架构设计、设计模式、最佳实践

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

- [Skylark 低代码平台](https://skylark.com)
- [问题反馈](https://github.com/your-username/go-skylark/issues)
- [版本历史](https://github.com/your-username/go-skylark/releases)

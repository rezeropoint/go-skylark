# Flows 模块文档

本模块负责与 Skylark 低代码平台的流程（Flow）相关功能，提供流程创建、查询、更新等完整的生命周期管理。

## 模块概述

### 核心功能（11个接口）

#### 流程管理
- **CreateFlow**: 创建并启动流程
- **UpdateJourneyStatus**: 更新流程任务状态（支持审批、回退、转交等操作）

#### 流程查询
- **GetJourneyBySN**: 根据流程编号查询流程记录
- **GetJourneyAssignments**: 获取流程节点处理信息列表
- **GetJourneyDetail**: 获取流程记录详情（包含字段值和附件）
- **GetFlowDetail**: 获取流程详情（包含字段、节点、边信息）

#### 工作台支持
- **GetUserAssignments**: 获取用户处理的任务列表
- **GetProposedJourneys**: 获取用户发起的流程列表

#### 搜索与历史
- **SearchJourneys**: 搜索流程记录（支持多条件筛选）
- **GetJourneyMoments**: 获取流程审批历史

#### 辅助功能
- **GetCurrentProcessingUsers**: 获取当前流程任务的处理者
- **AbortJourney**: 终止流程任务

### 文件组织

```
internal/flows/
├── flows.go            # 接口定义
├── handler.go          # 接口实现（960+ 行）
├── model.go            # 数据模型及转换
├── config.go           # 配置结构
├── enrichment.go       # 性能优化逻辑（flow信息补充）
├── helpers.go          # 辅助函数
└── README.md           # 本文档
```

---

## 性能优化方案：映射库+缓存

### 问题背景

`GetUserAssignments` 返回的 assignment 列表中只有 `journey_id`，缺少 `flow_id` 和 `flow_title`，导致前端需要额外请求才能显示流程名称。

### 优化架构

采用**三层优化策略**：映射库 + 缓存 + 批量查询

```
┌─────────────────────────────────────────────────────────────┐
│  用户请求：获取待处理列表（100条assignment）                   │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Step 1: 调用 Skylark API 获取 assignment 列表               │
│  返回：[{id, journey_id, vertex_id, ...}, ...]              │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Step 2: 提取所有唯一的 journey_id                            │
│  结果：[3990, 3887, 3445, ...]（假设50个不同的journey）       │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Step 3: 批量查询 journey_id → flow_id 映射                  │
│  查询远程 PostgreSQL（通过 getRemoteDB 依赖注入）             │
│    SELECT journey_id, flow_id FROM journeys                 │
│    WHERE journey_id = ANY($1)                               │
│  结果：{3990: 135, 3887: 135, 3445: 142, ...}               │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Step 4: 提取所有唯一的 flow_id                              │
│  结果：[135, 142, 156]（假设只有3个不同的flow）                │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Step 5: 批量查询 flow_id → flow_title（Redis 缓存）         │
│  MGET skylark:flow:135 skylark:flow:142 skylark:flow:156    │
│  命中：{135: "打卡成功", 142: "请假申请"}                     │
│  未命中：[156]                                               │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Step 6: 批量调用 Skylark API 获取未命中的 flow 信息         │
│  GET /api/v4/yaw/flows/156                                  │
│  结果：{id: 156, title: "报销流程", ...}                     │
│  并回写缓存：SET skylark:flow:156 "{...}" EX {TTL}          │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Step 7: 合并数据，返回完整的列表                             │
│  [{id, journey_id, flow_id, flow_title, ...}, ...]          │
└─────────────────────────────────────────────────────────────┘
```

### 性能对比

| 场景 | 优化前 | 优化后（首次） | 优化后（缓存命中） |
|------|--------|---------------|-------------------|
| 100条记录，50个journey，3个flow | 1次列表 + 50次journey查询 + 3次flow查询<br>= 54次API请求 | 1次列表 + 1次DB查询 + 3次flow查询<br>= 4次API请求 + 1次DB | 1次列表 + 1次DB查询<br>= 1次API请求 + 1次DB |
| 耗时估算 | 50×50ms + 3×100ms<br>= 2.8秒 | 1×50ms + 1×10ms + 3×100ms<br>= 360ms | 1×50ms + 1×10ms<br>= 60ms |
| **性能提升** | - | **87%** | **98%** |

### 实现位置

- **核心逻辑**: `enrichment.go` - `enrichAssignmentsWithFlowInfo()`
- **映射查询**: `enrichment.go` - `getJourneyFlowMapping()`
- **缓存查询**: `enrichment.go` - `batchGetFlowInfo()`
- **自动调用**: `handler.go:507-588` - `GetUserAssignments()` 和 `GetProposedJourneys()`

---

## 配置说明

### 缓存配置

```go
// internal/flows/config.go
type Config struct {
    // FlowInfoCacheTTL Flow信息缓存过期时间（秒）
    // 默认值：3600（1小时）
    // 建议范围：1800-7200（30分钟到2小时）
    FlowInfoCacheTTL int
}
```

### 初始化示例

```go
// 使用默认配置（TTL=3600秒）
flowRegistry := flows.NewSkylarkFlowRegistry(
    &flows.Config{},
    cacheInterface,
    getPlatformConfigFunc,
    getRemoteUserIDsFunc,
    getRemoteDBFunc,
)

// 自定义缓存TTL
flowRegistry := flows.NewSkylarkFlowRegistry(
    &flows.Config{
        FlowInfoCacheTTL: 7200, // 2小时
    },
    cacheInterface,
    getPlatformConfigFunc,
    getRemoteUserIDsFunc,
    getRemoteDBFunc,
)
```

### 依赖注入

本模块通过函数注入解耦，避免循环依赖：

```go
// 必需的依赖注入函数
type Dependencies struct {
    // 获取平台配置（由 platform.Manager 提供）
    GetPlatformConfig core.GetPlatformConfigFunc

    // 获取远程用户ID（由 user.Manager 提供，入参转换）
    GetRemoteUserIDs core.GetRemoteUserIDsFunc

    // 批量反向转换函数（由 user.Manager 提供，出参转换）
    FillLocalUserIDMap core.FillLocalUserIDMapFunc

    // 获取远程数据库连接（由 platform.Manager 提供）
    GetRemoteDB core.GetRemoteDBFunc
}
```

---

## API 调用规范

### 通用规范

1. **HTTP 客户端**: 使用 `httpc.Do` 发送请求
2. **响应解析**: 使用 `httputils.ReadJSONResponse` 解析 JSON
3. **认证**: 所有请求添加 `Authorization` 头（从平台配置获取）
4. **错误映射**: HTTP 404 自动映射为 `core.ErrJourneyNotFound` 或 `core.ErrFlowNotFound`

### 参数校验

所有接口必须校验：
- `tenantID`: 非空
- `flowID`, `journeyID`: 大于 0
- `page`: 大于 0（默认1）
- `pageSize`: 1-100 范围（默认20）

### 分页处理

```go
// 默认分页参数
const (
    DefaultPage     = 1
    DefaultPageSize = 20
    MaxPageSize     = 100
)

// 响应头解析
totalCount := resp.Header.Get("X-SLP-Total-Count")
totalPages := resp.Header.Get("X-SLP-Total-Pages")
```

---

## 缓存策略

### Flow 信息缓存

| 数据类型 | 缓存键 | TTL | 说明 |
|---------|--------|-----|------|
| Flow 信息 | `skylark:flow:{flow_id}` | 可配置（默认3600秒） | Flow 变化频率低，适合长期缓存 |
| 用户名 | `skylark:users:{tenant}:{user_id}` | 24小时 | 用户名变化频率低 |

### 容错机制

- 映射库查询失败：降级为逐个 API 调用（保证功能可用）
- 缓存查询失败：直接调用 API（记录日志，不影响主流程）
- Flow 信息获取失败：跳过该 flow（返回部分数据）

---

## 使用示例

### 创建流程

```go
// 1. 准备流程数据
data := map[string]core.TypedValue{
    "field_name": {
        Type:  "string",
        Value: "测试数据",
    },
    "amount_field": {
        Type:  "number",
        Value: 1000.5,
    },
}

// 2. 创建流程（SDK 会自动获取分布式锁）
err := flowRegistry.CreateFlow(ctx, "app.skylarkflow.com", 123, 456, "Bearer token", data)
if err != nil {
    log.Printf("创建流程失败: %v", err)
}
```

### 更新流程状态

```go
// 1. 准备更新选项（注意：NextVertexID 会从第一次请求的响应中自动获取，无需手动指定）
options := flows.UpdateJourneyStatusOptions{
    Comment:      "审批通过",
    Data: map[string]core.TypedValue{
        "approval_result": {
            Type:  "string",
            Value: "同意",
        },
    },
}

// 2. 更新流程（localUserID 会自动转换为远程用户ID）
err := flowRegistry.UpdateJourneyStatus(
    ctx,
    "tenant-001",
    123,     // flowID
    456,     // journeyID
    789,     // assignmentID
    "user-local-001", // localUserID
    core.OperationApprove,
    options,
)
```

### 查询流程记录

```go
// 根据流程编号查询
journey, err := flowRegistry.GetJourneyBySN(ctx, "tenant-001", 123, "SN20250112001")

// 获取流程详情
detail, err := flowRegistry.GetJourneyDetail(ctx, "tenant-001", 123, 456)

// 获取流程配置
flowDetail, err := flowRegistry.GetFlowDetail(ctx, "tenant-001", 123)
```

### 获取用户任务列表

```go
// 获取待处理任务（自动补充 flow_id 和 flow_title）
assignments, total, err := flowRegistry.GetUserAssignments(
    ctx,
    "tenant-001",
    "user-local-001", // 本地用户ID，自动转换
    core.AssignmentCategoryPending,
    1,  // page
    20, // pageSize
)

// 遍历任务
for _, assignment := range assignments {
    fmt.Printf("任务ID: %d, 流程: %s (ID: %d)\n",
        assignment.ID, assignment.FlowTitle, assignment.FlowID)
}
```

### 搜索流程记录

```go
// 搜索进行中的流程
req := &core.JourneySearchRequest{
    FlowID:      123,
    Status:      core.StatusProcessing, // 直接使用常量
    Keyword:     "报销",                 // 直接使用字符串
    InitiatorID: "local-user-001",      // SDK 自动转换为远程ID
    Page:        1,
    PageSize:    20,
}

journeys, total, err := flowRegistry.SearchJourneys(ctx, "tenant-001", req)
```

### 获取审批历史

```go
// 获取流程审批历史
moments, err := flowRegistry.GetJourneyMoments(ctx, "tenant-001", 456)

// 打印审批时间线
for _, moment := range moments {
    fmt.Printf("%s - %s: %s (%s)\n",
        moment.CreatedAt,
        *moment.OperatorName,
        core.TranslateStatus(moment.Status),
        *moment.Comment,
    )
}
```

### 获取当前处理人

```go
// 获取当前处理人列表
users, err := flowRegistry.GetCurrentProcessingUsers(ctx, "tenant-001", 123, 456)

for _, user := range users {
    fmt.Printf("处理人: %s (ID: %d, 手机: %s)\n",
        user.Name, user.ID, *user.Phone)
}
```

### 终止流程

```go
// 终止流程任务（不可逆操作）
err := flowRegistry.AbortJourney(ctx, "tenant-001", 123, 456)
if err != nil {
    log.Printf("终止流程失败: %v", err)
}
```

---

## 架构规范

### 三层架构

```
Engine Layer (engine/)          ← 对外接口，用户ID转换
    ↓ 依赖
Internal Layer (internal/flows) ← 业务逻辑，API调用
    ↓ 依赖
Core Layer (core/)              ← 领域模型，纯 Go 类型
```

### 数据流向

```
Skylark API 响应 → model.go (xxxResponse) → core/ (DomainModel) → 用户
```

### 关键设计

1. **函数注入解耦**: 通过 `core.GetRemoteDBFunc` 等函数类型避免循环依赖
2. **领域模型分离**: Core 层不包含任何框架类型（无 `db` 标签、无 `sql.Null*`）
3. **自动转换**: Engine 层自动处理本地用户ID → 远程用户ID 转换
4. **配置化**: 缓存 TTL、分页大小等参数支持配置
5. **容错降级**: 优化失败时自动降级为基础功能

---

## 错误处理

### 预定义错误（core/var.go）

- `ErrJourneyNotFound`: 流程记录不存在
- `ErrFlowNotFound`: 流程不存在
- `ErrHTTPRequestFailed`: HTTP 请求失败
- `ErrConfigNil`: 配置为空
- `ErrInvalidParams`: 参数无效

### 错误处理示例

```go
journey, err := flowRegistry.GetJourneyBySN(ctx, tenantID, flowID, sn)
if err != nil {
    if errors.Is(err, core.ErrJourneyNotFound) {
        // 处理不存在的情况
        return nil, fmt.Errorf("流程记录不存在: %s", sn)
    }
    // 处理其他错误
    return nil, err
}
```

---

## 注意事项

### 并发控制

`CreateFlow` 使用分布式锁（Redis）防止重复提交：
- 锁键格式: `flow:lock:{app}:{flowID}:{userID}`
- 默认过期时间: 30秒
- 支持重试机制

### 分布式事务

流程创建涉及多个步骤，通过分布式锁保证原子性：
1. 获取字段映射
2. 构建请求体
3. 上传附件（如有）
4. 调用 API 创建流程

### 性能建议

1. **合理设置缓存 TTL**: Flow 信息变化频率低，可设置较长 TTL（建议 1-2 小时）
2. **批量查询**: 使用 `GetUserAssignments` 时，SDK 自动进行批量优化
3. **避免频繁调用**: 对于列表接口，建议在前端实现分页缓存

---

## 开发指南

### 添加新接口

1. **定义领域模型**: `core/journey.go` 或 `core/flow.go`
2. **添加接口签名**: `flows.go`
3. **实现业务逻辑**: `handler.go`
4. **定义数据模型**: `model.go` - 添加响应结构体及 `ToDomain()` 方法
5. **Engine 层集成**: `engine/handler.go` - 添加包装方法
6. **更新文档**: 本文档和 `CLAUDE.md`

### 代码风格

- 所有接口必须有详细的文档注释
- 参数顺序：`ctx` → `tenantID` → 业务参数 → 分页参数
- 错误处理：优先使用预定义错误，必要时包装错误信息
- 日志记录：使用 `logx.Error` 记录错误（不使用 `Warn`）

### 测试规范

```go
func TestGetJourneyBySN(t *testing.T) {
    // 1. Mock 依赖
    mockCache := &MockCache{}
    mockGetConfig := func(ctx context.Context, tenantID string) (*core.PlatformConfig, error) {
        return &core.PlatformConfig{...}, nil
    }

    // 2. 创建测试实例
    registry := newSkylarkFlowRegistry(config, mockCache, mockGetConfig, ...)

    // 3. 执行测试
    journey, err := registry.GetJourneyBySN(ctx, "tenant-001", 123, "SN001")

    // 4. 断言结果
    assert.NoError(t, err)
    assert.Equal(t, "SN001", journey.SN)
}
```

---

## 版本历史

- **v1.1** (2025-01-14): 用户ID统一转换改造
  - 8个接口支持出参转换（远程用户ID → 本地用户ID）
  - 1个接口支持入参转换（本地抄送人ID → 远程ID）
  - SearchJourneys 接口优化（移除冗余 localUserID 参数）
  - 新增 5 个批量转换函数（helpers.go）
  - 依赖注入 FillLocalUserIDMap 函数

- **v1.0** (2025-11-12): 完成所有 11 个流程接口
  - 核心流程管理（5个接口）
  - 工作台支持（2个接口）
  - 搜索与历史（2个接口）
  - 辅助功能（2个接口）
  - 性能优化（自动补充 flow 信息）
  - 配置化支持（缓存 TTL 可配置）

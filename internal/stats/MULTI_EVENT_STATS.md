# Skylark Query - Stats Manager 统计分析管理器

## 概述

Stats Manager 是 Skylark Query Engine 的统计分析管理器，负责从远程 Skylark 数据库查询事件统计数据并提供聚合分析功能。

**包路径**: `pkg/skylarkq/internal/stats`

**职责**:
- 事件处理时长统计（平均/最短/最长时长）
- 事件状态统计（各状态数量和占比）
- 事件趋势统计（按日/周/月聚合）
- 节点统计（各节点事件数和平均时长）
- 处理人统计（Top N处理人排名）
- 组织统计（各组织事件数和平均时长）
- 待处理事件统计（实时查询，无缓存）

**特性**:
- ✅ 支持单个或多个事件配置ID的聚合统计
- ✅ Redis缓存优化（默认5分钟TTL）
- ✅ 组织权限自动过滤
- ✅ 结果智能合并（加权平均、分组聚合）
- ✅ 向后兼容单ID查询

---

## 架构设计

### Manager模式

Stats Manager 遵循 Skylark Query 统一的 Manager 模式，与 Query/Event/Platform/Mapping Manager 保持一致的设计风格。

**依赖关系**:
```
Stats Manager
  ├── Platform Manager (提供远程DB连接)
  ├── Event Manager (提供事件配置加载)
  └── Mapping Manager (提供组织映射)
```

**初始化流程**:
```go
// 1. 初始化基础Manager
platformMgr, _ := platform.NewManager(localDB)
eventMgr, _ := event.NewManager(localDB, platformMgr.GetRemoteDB)
mappingMgr, _ := mapping.NewManager(localDB, redisClient)

// 2. 初始化Stats Manager（注入依赖）
statsMgr, _ := stats.NewManager(
    statsConfig,
    localDB,
    platformMgr.GetRemoteDB,       // 远程DB连接
    eventMgr.GetWithFields,        // 事件配置加载
    mappingMgr.ListOrgMappings,    // 组织映射加载
    redisClient,                   // Redis缓存
)
```

---

## 核心功能

### 1. 多事件ID统计支持

**实现日期**: 2025-10-16

实现了统计查询支持多个事件配置ID的功能，允许在单次API调用中聚合多个事件类型的统计数据，适用于可视化大屏等场景。

#### 数据结构变更 (core/query.go)

```go
// Before:
EventConfigID string // 事件配置ID（必填）

// After:
EventConfigIDs []string // 事件配置ID列表（必填，支持单个或多个ID）
```

### 2. 统计合并逻辑 (merge.go)

创建了7个统计合并函数，每个对应一种统计类型：

#### 2.1 处理时长统计 (mergeDurationStats)
- **合并策略**: 加权平均
- **计算公式**: `AvgDuration = Σ(avg * count) / Σ(count)`
- **字段处理**:
  - `AvgDuration`: 加权平均
  - `MinDuration`: 取所有结果的最小值
  - `MaxDuration`: 取所有结果的最大值
  - `TotalCount`: 求和
  - `CompletedCount`: 求和

#### 2.2 状态统计 (mergeStatusStats)
- **合并策略**: 按StatusKey分组聚合
- **处理步骤**:
  1. 使用map按StatusKey分组，累加Count
  2. 重新计算Total
  3. 重新计算每个状态的Percentage
  4. 按Count降序排序

#### 2.3 趋势统计 (mergeTrendStats)
- **合并策略**: 按Date分组聚合
- **处理步骤**:
  1. 使用map按Date分组
  2. 累加TotalCount和CompletedCount
  3. 重新计算CompletionRate
  4. 按Date正序排序

#### 2.4 节点统计 (mergeNodeStats)
- **合并策略**: 按VertexID分组，加权平均时长
- **字段处理**:
  - `AvgDuration`: 加权平均 = `Σ(avg * count) / Σ(count)`
  - `Count`: 求和
  - `VertexName`: 保留第一个非空名称
- **排序**: 按Count降序

#### 2.5 处理人统计 (mergeUserStats)
- **合并策略**: 按UserID分组，应用TopN
- **处理步骤**:
  1. 使用map按UserID分组，累加Count
  2. 更新UserName（保留第一个非空值）
  3. 按Count降序排序
  4. 取Top N
  5. 重新计算Rank

#### 2.6 组织统计 (mergeOrgStats)
- **合并策略**: 按OrgValue分组，加权平均时长
- **字段处理**:
  - `AvgDuration`: 加权平均
  - `Count`: 求和
- **排序**: 按Count降序

#### 2.7 待处理事件统计 (mergePendingStats)
- **合并策略**: 直接求和
- **字段处理**: 所有字段直接累加
  - `PendingCount`: 求和
  - `ProcessingCount`: 求和
  - `Total`: 求和

### 3. 缓存机制优化 (cache.go)

#### 3.1 缓存Key格式

**单ID格式** (向后兼容):
```
{prefix}{tenant_id}:{event_config_id}:{params_hash}
```

**多ID格式** (新增):
```
{prefix}{tenant_id}:multi:{sorted_ids_hash}:{params_hash}
```

#### 3.2 Key生成逻辑

```go
func buildStatsCacheKey(prefix, tenantID string, eventConfigIDs []string, req *core.StatsRequest) string {
    // 计算请求参数hash
    paramsStr := fmt.Sprintf("%v|%v|%s|%s|%d",
        req.DateFrom, req.DateTo, req.Status, req.GroupBy, req.TopN)
    paramsHash := md5.Sum([]byte(paramsStr))

    // 单ID：使用原格式
    if len(eventConfigIDs) == 1 {
        return fmt.Sprintf("%s%s:%s:%s", prefix, tenantID, eventConfigIDs[0], paramsHashStr)
    }

    // 多ID：对ID排序后计算hash
    sortedIDs := make([]string, len(eventConfigIDs))
    copy(sortedIDs, eventConfigIDs)
    sort.Strings(sortedIDs)

    idsStr := ""
    for _, id := range sortedIDs {
        idsStr += id + ","
    }
    idsHash := md5.Sum([]byte(idsStr))

    return fmt.Sprintf("%s%s:multi:%s:%s", prefix, tenantID, idsHashStr, paramsHashStr)
}
```

**设计原理**:
- 对ID排序确保 `["id1", "id2"]` 和 `["id2", "id1"]` 产生相同的缓存key
- 使用MD5 hash避免key过长
- `:multi:` 标识符便于调试和监控

### 4. Handler实现 (handler.go)

所有7个统计Handler都采用统一的实现模式：

#### 4.1 统一处理流程

```go
func (m *statsManager) GetXXXStats(ctx context.Context, req *core.StatsRequest) (*core.XXXStats, error) {

    // 2. 单ID：直接调用helper（向后兼容）
    if len(req.EventConfigIDs) == 1 {
        return m.getSingleXXXStats(ctx, req)
    }

    // 3. 多ID：尝试从缓存获取合并结果
    cached, err := m.getCachedXXXStats(ctx, req)
    if err == nil && cached != nil {
        logx.Info("从缓存获取多事件XXX统计成功")
        return cached, nil
    }

    // 4. 循环查询每个ID（利用单ID缓存）
    results := make([]*core.XXXStats, 0, len(req.EventConfigIDs))
    for _, eventConfigID := range req.EventConfigIDs {
        singleReq := *req
        singleReq.EventConfigIDs = []string{eventConfigID}

        singleStats, err := m.getSingleXXXStats(ctx, &singleReq)
        if err != nil {
            return nil, fmt.Errorf("查询事件配置[%s]的XXX统计失败: %w", eventConfigID, err)
        }
        results = append(results, singleStats)
    }

    // 5. 合并结果
    merged := mergeXXXStats(results)

    // 6. 缓存合并结果
    _ = m.setCachedXXXStats(ctx, req, merged)

    // 7. 记录日志
    logx.Info("查询多事件XXX统计成功（已合并）")

    return merged, nil
}
```

#### 4.2 Helper函数

每个Handler都有对应的 `getSingleXXXStats` 辅助函数：

```go
func (m *statsManager) getSingleXXXStats(ctx context.Context, req *core.StatsRequest) (*core.XXXStats, error) {
    // 注意：此时req.EventConfigIDs应该只有1个元素

    // 1. 尝试从缓存获取
    cached, err := m.getCachedXXXStats(ctx, req)
    if err == nil && cached != nil {
        return cached, nil
    }

    eventConfigID := req.EventConfigIDs[0]

    // 2-5. 加载配置、构建SQL、执行查询...（原有逻辑）

    // 6. 写入缓存
    _ = m.setCachedXXXStats(ctx, req, &stats)

    return &stats, nil
}
```

#### 4.3 特殊处理

**GetPendingStats** - 实时查询（不缓存）:
```go
// 多ID场景也不使用缓存，直接循环查询并合并
// 日志记录source为"database_realtime_merged"
```

**GetUserStats** - TopN参数:
```go
// 合并时传入TopN参数
merged := mergeUserStats(results, req.TopN)
```

## 缓存分层策略

### 两级缓存设计

1. **单ID缓存** (第一层):
   - Key格式: `{prefix}{tenant_id}:{event_config_id}:{params_hash}`
   - TTL: 由配置决定（默认5分钟）
   - 用途: 单ID查询和多ID查询的中间缓存

2. **多ID合并缓存** (第二层):
   - Key格式: `{prefix}{tenant_id}:multi:{sorted_ids_hash}:{params_hash}`
   - TTL: 同单ID缓存
   - 用途: 直接返回多ID查询结果

### 查询流程

**单ID请求**:
```
请求 → 检查单ID缓存 → (miss) → 查询数据库 → 写入单ID缓存 → 返回
```

**多ID请求**:
```
请求 → 检查多ID缓存 → (miss) → 循环查询单ID (利用单ID缓存) → 合并结果 → 写入多ID缓存 → 返回
```

### 缓存优势

1. **避免重复查询**: 多ID请求能复用已有的单ID缓存
2. **提升性能**: 相同的多ID组合直接返回缓存结果
3. **向后兼容**: 不影响现有单ID请求的缓存行为

## 向后兼容性

### API层面

- 接受单个ID: `EventConfigIDs: ["id1"]`
- 接受多个ID: `EventConfigIDs: ["id1", "id2", "id3"]`
- 旧代码只需修改字段名即可

### 缓存层面

- 单ID请求保持原有缓存key格式
- 不会导致现有缓存失效
- 平滑过渡，无需清空缓存

### 行为层面

- 单ID请求的查询逻辑完全不变
- 日志格式保持一致
- 错误处理保持一致

## 性能优化

### 1. 缓存复用

多ID查询会优先利用已有的单ID缓存：

```
假设有3个事件ID: [A, B, C]
- 第一次查询: 3次数据库查询 + 写入3个单ID缓存 + 写入1个多ID缓存
- 第二次查询相同ID组合: 直接从多ID缓存返回
- 第三次查询部分重叠[A, B, D]: A和B从缓存读取，只查询D
```

### 2. 并发优化建议

当前实现是串行查询每个ID，后续可优化为并发查询：

```go
// TODO: 优化为并发查询
var wg sync.WaitGroup
results := make([]*core.XXXStats, len(req.EventConfigIDs))
errs := make([]error, len(req.EventConfigIDs))

for i, eventConfigID := range req.EventConfigIDs {
    wg.Add(1)
    go func(idx int, id string) {
        defer wg.Done()
        singleReq := *req
        singleReq.EventConfigIDs = []string{id}
        results[idx], errs[idx] = m.getSingleXXXStats(ctx, &singleReq)
    }(i, eventConfigID)
}

wg.Wait()
// 检查errs并合并results
```

### 3. 合并算法优化

- 使用map进行分组，时间复杂度O(n)
- 预分配切片容量，减少扩容
- 单次排序，避免重复排序

## 测试建议

### 1. 单元测试

```go
// 测试单ID查询（向后兼容）
func TestGetDurationStats_SingleID(t *testing.T) {
    req := &core.StatsRequest{
        EventConfigIDs: []string{"event1"},
        TenantID: "tenant1",
    }
    // 验证返回结果与旧逻辑一致
}

// 测试多ID查询
func TestGetDurationStats_MultiID(t *testing.T) {
    req := &core.StatsRequest{
        EventConfigIDs: []string{"event1", "event2", "event3"},
        TenantID: "tenant1",
    }
    // 验证合并逻辑正确
}

// 测试缓存key生成
func TestBuildStatsCacheKey(t *testing.T) {
    // 验证单ID和多ID生成不同格式的key
    // 验证ID顺序不影响多ID的key
}

// 测试合并函数
func TestMergeDurationStats(t *testing.T) {
    results := []*core.DurationStats{
        {AvgDuration: 100, TotalCount: 10, MinDuration: 50, MaxDuration: 150},
        {AvgDuration: 200, TotalCount: 20, MinDuration: 80, MaxDuration: 300},
    }
    merged := mergeDurationStats(results)
    // 验证加权平均: (100*10 + 200*20) / 30 = 166.67
    // 验证最小值: 50
    // 验证最大值: 300
}
```

### 2. 集成测试

```go
// 测试缓存命中
func TestMultiIDCacheHit(t *testing.T) {
    // 第一次查询，缓存miss
    // 第二次查询相同参数，验证从缓存返回
}

// 测试单ID缓存复用
func TestSingleIDCacheReuse(t *testing.T) {
    // 先查询单ID，写入单ID缓存
    // 再查询包含该ID的多ID请求，验证复用单ID缓存
}
```

### 3. 性能测试

```go
// 基准测试：单ID vs 多ID
func BenchmarkGetDurationStats_SingleID(b *testing.B) { }
func BenchmarkGetDurationStats_MultiID_3IDs(b *testing.B) { }
func BenchmarkGetDurationStats_MultiID_10IDs(b *testing.B) { }

// 验证合并函数性能
func BenchmarkMergeDurationStats(b *testing.B) { }
```

## 使用示例

### 单事件统计（向后兼容）

```go
req := &core.StatsRequest{
    EventConfigIDs: []string{"event_config_001"},
    TenantID: "tenant_001",
    UserOrgIDs: []string{"org_001"},
    DateFrom: &startTime,
    DateTo: &endTime,
}

stats, err := engine.GetDurationStats(ctx, req)
```

### 多事件统计（新功能）

```go
req := &core.StatsRequest{
    EventConfigIDs: []string{
        "event_config_001", // 设备报修
        "event_config_002", // 设备巡检
        "event_config_003", // 设备维护
    },
    TenantID: "tenant_001",
    UserOrgIDs: []string{"org_001"},
    DateFrom: &startTime,
    DateTo: &endTime,
    GroupBy: "day", // 趋势统计使用
    TopN: 10,       // 用户统计使用
}

// 获取合并后的处理时长统计
durationStats, err := engine.GetDurationStats(ctx, req)

// 获取合并后的状态统计
statusStats, err := engine.GetStatusStats(ctx, req)

// 获取合并后的趋势统计
trendStats, err := engine.GetTrendStats(ctx, req)
```

## 已知限制

1. **组织统计的特殊性**:
   - 不同事件配置可能使用不同的组织字段
   - 当前实现会合并所有组织值，可能需要前端明确展示来源

2. **节点统计的字段冲突**:
   - 不同事件配置可能有相同的VertexID但不同的VertexName
   - 当前策略：保留第一个非空名称

3. **串行查询性能**:
   - 多ID查询是串行的，大量ID时可能较慢
   - 建议：后续改为并发查询

## 后续优化方向

### 1. 并发查询

将串行查询改为并发：

```go
// 使用goroutine池并发查询多个ID
// 控制并发数避免过多连接
```

### 2. 智能缓存预热

```go
// 常用的多ID组合可以定时预热缓存
// 基于历史查询记录分析热点组合
```

### 3. 缓存失效策略

```go
// 事件数据更新时主动失效相关缓存
// 支持通配符删除：skylark:stats:*:tenant_id:*:event_config_id:*
```

### 4. 监控指标

```go
// 添加Prometheus指标
// - skylarkq_multi_id_query_total{ids_count="3"}
// - skylarkq_cache_hit_ratio{type="single_id|multi_id"}
// - skylarkq_merge_duration_seconds
```

---

## 文件结构

```
pkg/skylarkq/internal/stats/
├── stats.go              # Manager接口定义（7个统计方法）
├── handler.go            # 统计接口实现（统一的查询和合并流程）
├── config.go             # 统计配置结构
├── sql.go                # 统计SQL构建函数（7个SQL构建函数）
├── merge.go              # 结果合并函数（7个合并算法）
├── cache.go              # 统计缓存操作（Redis读写）
├── helpers.go            # 工具函数
├── internal.go           # 内部辅助函数
└── MULTI_EVENT_STATS.md  # 本文档（多事件ID统计功能说明）
```

---

## 与Query Manager的关系

**职责分离**：
- **Query Manager** (`internal/query/`): 负责事件数据查询（事件列表、事件详情、flow配置查询）
- **Stats Manager** (`internal/stats/`): 负责事件统计分析（时长、状态、趋势、节点、用户、组织统计）

**共享依赖**：
- 都依赖 Platform Manager 获取远程DB连接
- 都依赖 Event Manager 获取事件配置
- 都依赖 Mapping Manager 获取组织映射
- 都使用相同的 Redis 缓存客户端

**独立配置**：
- Query Config: 查询相关配置（FlowListCacheTTL、UserNameCacheTTL、QueryTimeout等）
- Stats Config: 统计相关配置（StatsCacheTTL、MaxEventConfigIDs等）

---

## 文件清单（历史变更）

**从 query 包迁移的文件**：
- `query/stats_handler.go` → `stats/handler.go`
- `query/stats.go` → `stats/sql.go`
- `query/stats_merge.go` → `stats/merge.go`
- `query/stats_cache.go` → `stats/cache.go`

**新增的文件**：
- `stats/stats.go` - Manager接口定义
- `stats/config.go` - 统计配置结构
- `stats/helpers.go` - 工具函数
- `stats/internal.go` - 内部辅助函数

**修改的文件**：
- `pkg/skylarkq/core/query.go` - StatsRequest结构定义（EventConfigID → EventConfigIDs）
- `pkg/skylarkq/engine/handler.go` - 添加statsMgr字段和统计接口转发

---

## 版本历史

- **v2.0.0** (2025-10-16): 架构重构 - 拆分Stats Manager
  - ✅ 将统计功能从query包独立为stats包
  - ✅ 遵循Manager模式，与其他Manager保持一致
  - ✅ 职责更清晰，易于维护和扩展

- **v1.0.0** (2025-10-16): 多事件ID统计功能
  - ✅ 支持多事件ID查询
  - ✅ 实现7种统计类型的合并逻辑
  - ✅ 优化缓存策略
  - ✅ 保持向后兼容

---

## 参考文档

- **主开发文档**: `pkg/skylarkq/DEVELOPMENT.md`
- **Query Manager**: `pkg/skylarkq/internal/query/`
- **Engine接口**: `pkg/skylarkq/engine/engine.go`
- **核心数据结构**: `pkg/skylarkq/core/query.go`

---

**最后更新**: 2025-10-16
**维护者**: Rezer

# internal/cache - 缓存模块文档

## 概述

`internal/cache` 模块提供统一的缓存抽象层,实现了 `core.CacheInterface` 接口,支持以下功能:

- **Flow 缓存**: Flow 列表、字段、详情信息缓存
- **用户名缓存**: 批量查询用户名并自动缓存
- **组织映射缓存**: 业务组织字段值映射
- **配置缓存**: 平台配置、事件配置
- **统计结果缓存**: 各类统计分析结果
- **分布式锁**: 流程创建等场景的并发控制
- **缓存监控**: 实时统计命中率和性能指标

## 文件组织

| 文件 | 职责 |
|------|-----|
| `cache.go` | 对外接口入口 |
| `handler.go` | SkylarkCache 结构定义 |
| `flow.go` | Flow 相关缓存 (列表/字段/详情) |
| `user.go` | 用户名缓存 + 批量查询 |
| `mapping.go` | 组织映射缓存 |
| `organization.go` | 组织ID双向映射缓存 |
| `platform.go` | 平台配置缓存 |
| `event.go` | 事件配置缓存 |
| `field.go` | 字段映射缓存 (API模块) |
| `lock.go` | 分布式锁实现 |
| `helpers.go` | 辅助函数 (TTL随机偏移) |
| `metrics.go` | 缓存监控指标 |
| `config.go` | 缓存配置定义 |

## 核心优化策略

### 1. 缓存数据源隔离

**问题**: API数据（实时）和数据库数据（可能有延迟）混用同一缓存键

**解决方案**: 按数据源分离缓存键
- `skylark:flow:api:{tenant}:{flow_id}` - API数据（flows模块）
- `skylark:flow:{tenant}:{flow_id}` - 数据库数据（query模块）

**收益**:
- API的新鲜数据不会被数据库过时数据覆盖
- 缓存语义清晰，易于调试
- enrichment.go 使用API缓存，保证数据实时性

### 2. 防止缓存雪崩

**问题**: 大量缓存同时过期导致数据库压力骤增

**解决方案** (`helpers.go:20-32`):
```go
func addJitter(ttl int) int {
    jitter := ttl / 20  // ±5% 偏移
    offset := rand.Intn(2*jitter+1) - jitter
    return ttl + offset
}
```

**应用**: 所有 Set 操作自动添加随机偏移

### 3. 批量查询优化

**问题**: query 和 stats 模块重复实现用户名批量查询

**解决方案** (`user.go:84-152`):
```go
func BatchGetUserNames(ctx, remoteDB, tenantID, userIDs, ttl) {
    // 1. 批量从缓存获取
    for _, userID := range userIDs {
        if cached := GetUserName(userID); cached != nil {
            metrics.UserNameHit.Add(1)  // 监控埋点
        } else {
            uncachedIDs = append(uncachedIDs, userID)
            metrics.UserNameMiss.Add(1)
        }
    }

    // 2. 批量查询数据库 (IN 查询)
    query := "SELECT id, name FROM users WHERE id = ANY($1)"
    remoteDB.QueryRowsCtx(ctx, &users, query, pq.Array(uncachedIDs))

    // 3. 自动回写缓存
    for _, user := range users {
        SetUserName(user.ID, user.Name, addJitter(ttl))
    }
}
```

**收益**: 代码复用,减少维护成本

### 4. Write-Through 缓存策略

**应用**: 组织映射 Create/Update 操作

**策略**:
- 单个映射缓存: **Write-Through** (直接写入)
- 列表缓存: **Cache-Aside** (删除,下次读取时重建)

**原因**:
- 单个查询频繁,缓存命中率高
- 列表重建成本低,避免并发问题

## 缓存监控指标

### 使用方式

```go
// 获取监控指标
cache := NewSkylarkCache(redisClient, config)
metrics := cache.GetMetrics()

// 查看特定类型的统计
stats := metrics.GetStats("flow_list")
fmt.Printf("命中率: %.2f%%\n", stats.HitRate)

// 查看所有统计
allStats := metrics.GetAllStats()
for _, s := range allStats {
    fmt.Printf("%s: 命中=%d 未命中=%d 命中率=%.2f%%\n",
        s.Type, s.Hit, s.Miss, s.HitRate)
}
```

### 监控指标类型

| 类型 | 说明 | 埋点位置 |
|------|-----|---------|
| `flow_list` | Flow列表缓存 (DB数据) | flow.go:17,27 |
| `flow_fields` | Flow字段缓存 (DB数据) | flow.go:51,60 |
| `flow_info` | Flow详情缓存 (DB数据) | flow.go:83,92 |
| `flow_info_api` | Flow详情缓存 (API数据) | flow.go:112,121 |
| `user_name` | 用户名缓存 | user.go (已实现) |
| `org_mapping` | 组织映射缓存 | mapping.go (待添加) |
| `platform_config` | 平台配置缓存 | platform.go (待添加) |
| `event_config` | 事件配置缓存 | event.go (待添加) |
| `stats` | 统计结果缓存 | (待添加) |

## 缓存键设计规范

**规则**: 所有缓存键定义在 `core/*.go`,使用常量

**示例**:
```go
// core/query.go
const (
    CacheFlowListKeyPrefix    = "skylark:flows:"           // {tenant}:{namespace} (DB数据)
    CacheFlowFieldsKeyPrefix  = "skylark:flow_fields:"     // {tenant}:{flow_id} (DB数据)
    CacheFlowInfoKeyPrefix    = "skylark:flow:"            // {tenant}:{flow_id} (DB数据)
    CacheFlowInfoAPIKeyPrefix = "skylark:flow:api:"        // {tenant}:{flow_id} (API数据)
    CacheUserNameKeyPrefix    = "skylark:users:"           // {tenant}:{user_id}
)
```

**优点**:
- 统一管理,避免冲突
- 易于调试和监控
- 支持按前缀批量删除

### 缓存键数据源隔离 ⭐

**重要原则**: **不同数据源使用不同的缓存键**

**背景**:
- API 数据: 实时性强，来自 Skylark REST API
- 数据库数据: 可能有主从延迟，来自只读数据库

**实施**:
- ✅ `skylark:flow:api:{tenant}:{flow_id}` - API 数据 (flows 模块)
- ✅ `skylark:flow:{tenant}:{flow_id}` - 数据库数据 (query 模块)

**避免的问题**:
- ❌ API 的新鲜数据被数据库的过时数据覆盖
- ❌ 数据源混用导致缓存语义不清

## TTL 配置建议

| 数据类型 | 推荐 TTL | 配置项 | 原因 |
|---------|---------|--------|-----|
| Flow列表 | 1小时 | `FlowListCacheTTL` | 变化不频繁 |
| Flow字段 | 1小时 | `FlowFieldsCacheTTL` | 变化不频繁 |
| 用户名 | 24小时 | `UserNameCacheTTL` | 基本不变 |
| 组织映射 | 7天 | `OrgMappingCacheTTL` | 很少变化 |
| 平台配置 | 30分钟 | `PlatformConfigCacheTTL` | 可能调整 |
| 事件配置 | 10分钟 | `EventConfigCacheTTL` | 较常调整 |
| 统计结果 | 5分钟 | `StatsCacheTTL` | 实时性要求 |

**注意**: 实际 TTL 会自动添加 ±5% 随机偏移

## 最佳实践

### 1. 缓存穿透防护

✅ **已实现**: 缓存空结果
```go
if len(flows) == 0 {
    flows = []*core.FlowInfo{}  // 空数组而非nil
}
cache.SetFlowList(..., flows, ttl)  // 缓存空结果
```

### 2. 缓存雪崩防护

✅ **已实现**: TTL随机偏移
```go
ttlWithJitter := addJitter(ttl)  // 自动添加±5%偏移
cache.SetFlowList(..., ttl Withjitter)
```

### 3. 缓存击穿防护

✅ **已实现**: 分布式锁
```go
// flows/handler.go
lockKey := fmt.Sprintf("flow:lock:%s:%d", tenantID, flowID)
if err := cache.AcquireLockWithRetry(ctx, lockKey, uuid, expiry); err != nil {
    return core.ErrFlowCreationInProgress
}
defer cache.ReleaseLock(ctx, lockKey, uuid)
```

### 4. 异步写入

✅ **已实现**: 非阻塞缓存回写
```go
go func() {
    _ = cache.SetFlowInfo(context.Background(), flowInfo, ttl)
}()
```

## 常见问题

### Q: 为什么需要 BatchGetUserNames?

A: 批量查询避免 N+1 问题:
- ❌ 逐个查询: 100个用户 = 100次Redis + 可能100次DB
- ✅ 批量查询: 100个用户 = 100次Redis + 1次DB (IN查询)

### Q: 缓存失效策略是什么?

A: 根据场景选择:
- **单个实体**: Write-Through (Create/Update时直接写入)
- **列表聚合**: Cache-Aside (Create/Update/Delete时删除,读取时重建)

### Q: 如何调试缓存问题?

A:
1. 查看监控指标: `cache.GetMetrics().GetStats("flow_list")`
2. 检查 Redis 键: `redis-cli KEYS "skylark:*"`
3. 查看日志: 缓存操作失败会记录日志 (非致命错误)

### Q: 缓存监控指标线程安全吗?

A: ✅ 是的,使用 `atomic.Int64` 保证并发安全

## 性能优化数据

### enrichment.go 优化效果

**场景**: 100个 assignment → 50个 journey → 3个 flow

**优化前**:
- API 调用: 100+ 次 (每个 assignment 查询一次)
- 总耗时: ~5000ms

**优化后**:
- API 调用: 1-5 次 (未命中时)
- 缓存命中: ~60ms
- 缓存未命中: ~600ms
- **性能提升: 8-80倍**

## 相关文档

- [DEVELOPMENT.md](../../DEVELOPMENT.md) - 完整开发规范
- [CLAUDE.md](../../CLAUDE.md) - 快速参考
- [core/cache.go](../../core/cache.go) - 缓存接口定义

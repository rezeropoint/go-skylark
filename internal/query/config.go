// Package query 远程查询管理器
//
// 职责：
//   - 查询远程Skylark数据库（events数据、flows列表、字段列表）
//   - 组织权限计算和过滤（集成原 permission 功能）
//   - 动态SQL构建（参数化查询，防注入）
//   - 用户名批量转换（优先从Redis缓存获取）
//   - 结果集格式化
//
// 核心特性：
//   - Journey聚合：使用DISTINCT ON返回每个Journey的最新Assignment
//   - 权限强制验证：如果配置了组织字段但没有组织映射，返回错误而不是查询所有数据
//   - 参数化查询：所有变量使用占位符（$1, $2, ...），字段名白名单验证
//   - 批量优化：用户名查询优先从Redis批量获取（MGET），减少网络往返
//   - Redis缓存：支持缓存flows列表、字段列表、用户名（可选，传nil则降级为直接查询）
//
// 权限过滤机制（集成功能）：
//  1. 输入：UserOrgIDs（用户所属组织列表，含子组织）
//  2. 通过 OrgMappings 映射：LocalOrgID → RemoteOrgValue
//  3. 计算：用户有权访问的远程组织值列表
//  4. 构建 SQL WHERE 条件：org_field = ANY($1)
//  5. 使用 pq.Array(allowedValues) 传递参数
//
// SQL构建安全原则：
//  1. 字段白名单：只查询配置的字段，拒绝任意字段查询
//  2. 参数化查询：所有变量使用占位符，防止SQL注入
//  3. 字段名验证：正则 ^[a-zA-Z_][a-zA-Z0-9_]*$（只允许字母、数字、下划线）
//  4. 只读操作：引擎只执行SELECT语句，拒绝INSERT/UPDATE/DELETE
//
// Journey聚合查询示例：
//
//	-- 内层查询：使用DISTINCT ON获取每个Journey的最新Assignment
//	SELECT * FROM (
//	    SELECT DISTINCT ON (slp_journey_id)
//	        slp_journey_id,
//	        slp_assignment_id,
//	        slp_status,
//	        slp_vertex_id,
//	        slp_created_at,
//	        username,  -- 业务字段
//	        type       -- 业务字段
//	    FROM assignments_191
//	    WHERE username = ANY($1)  -- 组织权限过滤
//	      AND username::text ILIKE $2  -- 关键词搜索
//	    ORDER BY slp_journey_id, slp_created_at DESC  -- 关键：每个journey取最新
//	) t
//	ORDER BY slp_created_at DESC  -- 外层排序
//	LIMIT $3 OFFSET $4;
//
// 事件详情查询示例：
//
//	-- 查询Journey的所有Assignment（完整流转历史）
//	SELECT
//	    a.slp_assignment_id,
//	    a.slp_journey_id,
//	    a.slp_status,
//	    a.slp_vertex_id,
//	    v.name as vertex_name,
//	    v.alias_name as vertex_alias,
//	    a.slp_user_id,              -- 处理人ID（需要转换为用户名）
//	    a.slp_created_at,
//	    a.slp_updated_at,
//	    a.username,                 -- 业务字段示例
//	    a.type                      -- 业务字段示例
//	FROM assignments_191 a
//	LEFT JOIN vertices v ON a.slp_vertex_id = v.id
//	WHERE a.slp_journey_id = $1
//	ORDER BY a.slp_created_at ASC;  -- 按时间正序，展示流转过程
//
// 用户名转换机制：
//   - 远程Skylark users表：id 字段存储用户ID，name 字段存储用户姓名
//   - 缓存策略：Redis key = "skylark:users:{tenant_id}:{user_id}"，TTL 24小时
//   - 批量查询：先 MGET 批量获取缓存，未命中的再查数据库并回写缓存
//   - 自动填充：引擎内部自动填充 FlowNode.UserNames（数组，支持多人处理），REST层无需额外处理
//
// Redis缓存策略：
//   - flows列表：key = "skylark:flows:{tenant_id}:{namespace_id}"，TTL 1小时
//   - flow字段：key = "skylark:flow_fields:{tenant_id}:{flow_id}"，TTL 1小时
//   - 用户名：key = "skylark:users:{tenant_id}:{user_id}"，TTL 24小时
//   - 缓存更新：依赖过期时间自动清理，不主动失效（如需立即更新，手动删除key）
//
// 注意事项：
//  1. 所有查询必须包含租户ID验证，防止跨租户访问
//  2. 字段名必须通过白名单验证，拒绝任意字段查询
//  3. 远程连接只用于SELECT操作，拒绝任何修改语句
//  4. Redis缓存可选（rdb参数可为nil，降级为直接查询数据库）
//  5. 事件详情查询必须先验证权限，防止越权访问
//  6. 如果配置了组织字段但没有组织映射，必须返回错误（不能查询所有数据）
//
// 依赖注入：
//   - 本地数据库连接（sqlx.SqlConn）：查询本地配置数据（事件配置、字段配置、组织映射）
//   - 远程数据库连接函数（core.GetRemoteDBFunc）：由 Platform Manager 提供，获取远程Skylark连接
//   - Redis客户端（redis.Client）：可选，用于缓存远程数据
//
// 错误处理：
//   - 配置相关：core.ErrEventConfigNotFound、core.ErrRemoteTableNotFound
//   - 权限相关：core.ErrNoOrgPermission、core.ErrAccessDenied
//   - 查询相关：core.ErrInvalidQueryParams、core.ErrQueryTimeout
//   - 字段相关：core.ErrInvalidFieldName、core.ErrNoVisibleFields
package query

import "time"

// Config 远程查询管理器配置
type Config struct {
	// ========== Redis缓存TTL配置 ==========

	// FlowListCacheTTL flows列表缓存时间
	// key格式: skylark:flows:{tenant_id}:{namespace_id}
	// 默认值: 1小时
	FlowListCacheTTL time.Duration

	// FlowFieldsCacheTTL flow字段列表缓存时间
	// key格式: skylark:flow_fields:{tenant_id}:{flow_id}
	// 默认值: 1小时
	FlowFieldsCacheTTL time.Duration

	// UserNameCacheTTL 用户名缓存时间
	// key格式: skylark:users:{tenant_id}:{user_id}
	// 默认值: 24小时
	UserNameCacheTTL time.Duration

	// ========== 查询限制配置 ==========

	// MaxPageSize 每页最大记录数
	// 防止单次查询数据量过大
	// 默认值: 1000
	MaxPageSize int

	// DefaultPageSize 默认每页记录数
	// 当请求未指定PageSize时使用
	// 默认值: 20
	DefaultPageSize int

	// QueryTimeout 单次查询超时时间
	// 防止慢查询阻塞
	// 默认值: 30秒
	QueryTimeout time.Duration
}

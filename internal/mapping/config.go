// Package mapping 组织映射管理器
//
// 职责：管理组织映射配置（将远程业务字段值映射到本地组织ID）
//
// 核心特性：
//   - 平台级别共享：映射关系是租户级别全局配置，多个事件类型可共享
//   - 映射关系：(TenantID + RemoteOrgValue) → LocalOrgID
//   - 唯一约束：同一租户下，远程组织值唯一映射到一个本地组织
//
// 注意事项：
//   1. 创建时禁止前端传递ID，系统自动生成UUID
//   2. 创建/更新前会验证 local_org_id 是否存在于 organizations 表（status='active' AND deleted_at IS NULL）
//   3. 映射表不存储 local_org_name，查询时通过 LEFT JOIN organizations 表动态获取组织名称
//   4. 删除前会检查是否被事件配置使用（org_field_name IS NOT NULL）
//   5. 批量创建使用事务，要么全成功，要么全失败
//   6. UpdateOrgMapping 不允许修改 tenant_id
//
// 前端交互：
//   - 前端表单通过调用 backend 微服务获取组织列表（下拉选项）
//   - 用户选择组织后，只需传递 local_org_id 到 skylarkq
//   - 后端自动验证 local_org_id 并在查询时返回 local_org_name
//
package mapping

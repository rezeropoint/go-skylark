package query

import (
	"context"

	"github.com/rezeropoint/go-skylark/v2/core"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// Manager 远程查询管理器接口
// 职责：查询远程Skylark数据库 + 组织权限过滤 + SQL构建 + 用户名转换 + 统计分析
type Manager interface {
	QueryEventData(ctx context.Context, req *core.QueryRequest) (*core.QueryResponse, error)         // QueryEventData 查询事件数据列表（Journey聚合 + 权限过滤）
	GetEventDetail(ctx context.Context, req *core.DetailRequest) (*core.DetailResponse, error)       // GetEventDetail 获取事件详情（完整流转历史 + 用户名转换）
	GetFlowList(ctx context.Context, tenantID string, configuredOnly bool) ([]*core.FlowInfo, error) // GetFlowList 获取远程flows列表（configuredOnly=true时只返回已配置事件的流程）
	GetFlowFields(ctx context.Context, tenantID string, flowID int) ([]*core.FieldMetadata, error)   // GetFlowFields 获取远程flow字段列表（供前端配置）
}

// NewManager 创建远程查询管理器
// 参数：
//   - config: 查询管理器配置
//   - db: 本地数据库连接
//   - getRemoteDB: 获取远程数据库连接的函数（由 Platform Manager 提供）
//   - getEventConfig: 获取事件配置（含字段）的函数（由 Event Manager 提供）
//   - listOrgMappings: 获取组织映射列表的函数（由 Mapping Manager 提供）
//   - fillLocalUserIDMap: 批量反向转换远程用户ID为本地用户ID的函数（由 User Manager 提供）
//   - listConfiguredFlowIDs: 获取已配置事件的flow_id列表的函数（由 Event Manager 提供）
//   - cache: 缓存接口（必须提供，用于缓存）
//   - convertBusinessDataAttachments: 转换业务数据中附件为Base64的函数（由 Attachment Manager 提供）
func NewManager(
	config Config,
	db sqlx.SqlConn,
	getRemoteDB core.GetRemoteDBFunc,
	getEventConfig core.GetEventConfigWithFieldsFunc,
	listOrgMappings core.ListOrgMappingsFunc,
	fillLocalUserIDMap core.FillLocalUserIDMapFunc,
	listConfiguredFlowIDs core.ListConfiguredFlowIDsFunc,
	cache core.CacheInterface,
	convertBusinessDataAttachments core.ConvertBusinessDataAttachmentsFunc,
) (Manager, error) {
	return newQueryManager(config, db, getRemoteDB, getEventConfig, listOrgMappings, fillLocalUserIDMap, listConfiguredFlowIDs, cache, convertBusinessDataAttachments)
}

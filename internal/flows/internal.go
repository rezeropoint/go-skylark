package flows

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/httputils"
	"github.com/rezeropoint/go-skylark/v2/internal/images"

	"github.com/zeromicro/go-zero/core/collection"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpc"
)

// buildEntriesFromData 将原始数据转换为 entries 数组
// 这是一个公共函数，用于处理字段映射、图片上传、选项字段等逻辑
func (f *skylarkFlowRegistry) buildEntriesFromData(
	ctx context.Context,
	skylarkFlowAddress core.SkylarkAPIContext,
	originalData map[string]core.TypedValue,
	fieldMappings map[string]core.FieldMapping,
) ([]map[string]any, error) {
	entries := make([]map[string]any, 0, len(originalData))

	// 处理字段数据
	for key, TypedValue := range originalData {
		// 先判断 key 是否存在于 fieldMappings 中
		if fieldMapping, ok := fieldMappings[key]; ok {
			fieldId := fieldMapping.ID

			switch {
			case TypedValue.Type == string(core.FieldImage):
				// 提取图片URL
				imageURL, ok := TypedValue.Value.(string)
				if !ok {
					return nil, fmt.Errorf("%w: 图片字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
				}

				// 创建图片字段的 entry
				id, name, err := images.CreateImageEntryFromURL(ctx, skylarkFlowAddress, imageURL)
				if err != nil {
					return nil, err
				}
				// 创建 entry 并添加到 entries
				entries = append(
					entries,
					map[string]any{
						"field_id": fieldId,
						"value":    name,
						"value_id": id,
					})
			case TypedValue.Type == string(core.FieldImageBase64):
				// 提取 base64 数据
				base64Data, ok := TypedValue.Value.(string)
				if !ok {
					return nil, fmt.Errorf("%w: Base64 图片字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
				}

				// 上传 base64 图片
				id, name, err := images.CreateImageEntryFromBase64(ctx, skylarkFlowAddress, base64Data)
				if err != nil {
					return nil, err
				}
				// 创建 entry 并添加到 entries
				entries = append(
					entries,
					map[string]any{
						"field_id": fieldId,
						"value":    name,
						"value_id": id,
					})
			case TypedValue.Type == string(core.FieldString) && core.IsOptionField(fieldMapping.Type):
				// 处理选项字段(仅限字符串类型)
				valueStr, ok := TypedValue.Value.(string)
				if !ok {
					return nil, fmt.Errorf("%w: 选项字段 %s 的值应为字符串类型", core.ErrInvalidFieldValue, key)
				}

				// 查找匹配的选项
				optionId := 0
				for _, option := range fieldMapping.Options {
					if option.Value == valueStr {
						optionId = option.ID
						break
					}
				}

				if optionId == 0 {
					return nil, fmt.Errorf("%w: 选项字段 %s 的值 %s 不存在", core.ErrOptionNotFound, key, valueStr)
				}

				// 找到匹配的选项，添加option_id
				entries = append(
					entries,
					map[string]any{
						"field_id":  fieldId,
						"value":     valueStr,
						"option_id": optionId,
					})
			default:
				// 不需要特殊处理，直接添加。因为any类型可以包含所有类型
				entries = append(
					entries,
					map[string]any{
						"field_id": fieldId,
						"value":    TypedValue.Value,
					})
			}
		}
	}

	return entries, nil
}

// getFlowFieldMappings 获取流程字段映射
// 先尝试从缓存获取，缓存未命中则从API获取
func (f *skylarkFlowRegistry) getFlowFieldMappings(ctx context.Context, skylarkFlowAddress core.SkylarkAPIContext, flowID int64) (map[string]core.FieldMapping, error) {
	// 生成缓存键
	cacheKey := fmt.Sprintf("%s%s:%d", core.CacheFieldMappingKeyPrefix, skylarkFlowAddress.App, flowID)

	// 尝试从缓存中获取
	fieldMappings, found, err := f.cache.GetFieldMappingsFromCache(ctx, cacheKey)
	if err != nil {
		// 缓存查询出错，记录错误但继续执行，不影响正常流程
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "flows_field_mappings"),
			logx.Field("cache_key", cacheKey),
			logx.Field("error", err.Error()),
		).Error("从缓存获取字段映射时发生错误")
	} else if found {
		// 缓存命中，直接返回
		return fieldMappings, nil
	}

	// 缓存未命中或已过期，发起请求
	flowAPIURL := core.BuildFlowAPIURL(skylarkFlowAddress, flowID)
	resp, err := httpc.Do(ctx, http.MethodGet, flowAPIURL, core.AuthHeader{Token: skylarkFlowAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 使用 httputils 统一处理响应
	var respField core.Field
	if err := httputils.ReadJSONResponse(resp, &respField); err != nil {
		return nil, err
	}

	fieldMappings = make(map[string]core.FieldMapping, len(respField.Fields))
	for _, field := range respField.Fields {
		fieldMappings[field.IdentityKey] = field
	}

	// 将结果存入缓存
	if cacheErr := f.cache.SaveFieldMappingsToCache(ctx, cacheKey, fieldMappings); cacheErr != nil {
		// 缓存保存失败只记录错误，不影响正常流程
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "flows_field_mappings"),
			logx.Field("cache_key", cacheKey),
			logx.Field("error", cacheErr.Error()),
		).Error("保存字段映射到缓存时发生错误")
	}

	return fieldMappings, nil
}

// buildFlowRouteRequest 构建流程路由请求
// 根据原始数据和字段映射构建请求体
func (f *skylarkFlowRegistry) buildFlowRouteRequest(ctx context.Context, skylarkFlowAddress core.SkylarkAPIContext, flowID int64, originalData map[string]core.TypedValue, fieldMappings map[string]core.FieldMapping) (FlowRouteRequest, error) {
	// 使用公共函数处理字段数据
	entries, err := f.buildEntriesFromData(ctx, skylarkFlowAddress, originalData, fieldMappings)
	if err != nil {
		return FlowRouteRequest{}, err
	}
	if len(entries) == 0 {
		// 字段映射为空，可能是缓存没有更新，清除字段映射缓存
		cacheKey := fmt.Sprintf("%s%s:%d", core.CacheFieldMappingKeyPrefix, skylarkFlowAddress.App, flowID)
		if cacheErr := f.cache.ClearFieldMappingsCache(ctx, cacheKey); cacheErr != nil {
			// 缓存清除失败，返回包含缓存清除失败信息的错误
			return FlowRouteRequest{}, fmt.Errorf("%w: 字段映射为: %+v, 缓存清除错误为: %v", core.ErrFieldMappingEmptyAndCacheClearFailed, fieldMappings, cacheErr)
		}
		// 缓存清除成功，返回正常的字段映射为空错误
		return FlowRouteRequest{}, fmt.Errorf("%w: 字段映射为: %+v", core.ErrFieldMappingEmpty, fieldMappings)
	}
	userID, err := strconv.Atoi(skylarkFlowAddress.UserID)
	if err != nil {
		return FlowRouteRequest{}, fmt.Errorf("%w: %v", core.ErrUserIDConversionFailed, err)
	}
	// 创建符合 Request 结构体的数据
	return FlowRouteRequest{
		Assignment: RouteAssignment{
			Operation: string(core.OperationRoute),
			ResponseAttributes: map[string]any{
				"entries_attributes": entries,
			},
		},
		UserID: userID,
		Webhook: Webhook{
			PayloadURL:       "",
			SubscribedEvents: []string{core.EventJourneyStatus},
		},
		Token: skylarkFlowAddress.AuthHeader,
	}, nil
}

// buildRouteRequestForUpdate 构建第一次请求：修改数据（route操作）
// 根据原始数据和字段映射构建请求体
func (f *skylarkFlowRegistry) buildRouteRequestForUpdate(
	ctx context.Context,
	skylarkFlowAddress core.SkylarkAPIContext,
	remoteUserID int,
	originalData map[string]core.TypedValue,
	fieldMappings map[string]core.FieldMapping,
) (UpdateJourneyStatusRequest, error) {
	// 使用公共函数处理字段数据
	entries, err := f.buildEntriesFromData(ctx, skylarkFlowAddress, originalData, fieldMappings)
	if err != nil {
		return UpdateJourneyStatusRequest{}, err
	}

	// 创建符合第一次请求的数据结构
	return UpdateJourneyStatusRequest{
		Assignment: UpdateAssignment{
			Operation: string(core.OperationRoute),
			ResponseAttributes: map[string]any{
				"entries_attributes": entries,
			},
		},
		UserID: remoteUserID,
		Token:  skylarkFlowAddress.AuthHeader,
	}, nil
}

// buildOperationRequest 构建第二次请求：执行操作（approve/refuse/transfer/cancel）
func (f *skylarkFlowRegistry) buildOperationRequest(
	skylarkFlowAddress core.SkylarkAPIContext,
	remoteUserID int,
	operation string,
	nextVertexID int,
	comment string,
	carbonCopyUserIDs []int,
) (UpdateJourneyStatusRequest, error) {
	// 如果 carbonCopyUserIDs 为 nil，初始化为空数组
	if carbonCopyUserIDs == nil {
		carbonCopyUserIDs = []int{}
	}

	// 创建符合第二次请求的数据结构
	return UpdateJourneyStatusRequest{
		Assignment: UpdateAssignment{
			ResponseAttributes: map[string]any{
				"entries_attributes": []any{},
			},
			Operation:         operation,
			NextVertexID:      nextVertexID,
			Comment:           comment,
			CarbonCopyUserIDs: carbonCopyUserIDs,
		},
		UserID: remoteUserID,
		Token:  skylarkFlowAddress.AuthHeader,
	}, nil
}

// extractPendingNodes 从 assignments 中提取待处理节点（内部方法）
// 说明：
//   - 筛选条件：category='processed' AND status='processing'
//   - 自动补充节点名称（通过 vertices 映射）
//   - 自动补充处理人姓名（通过用户映射）
//
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID
//   - flowID: 流程ID
//   - assignments: 任务列表
//   - vertices: 节点信息映射（用于补充节点名称）
//
// 返回:
//   - []*core.PendingNode: 待处理节点列表
//   - error: 错误信息
func (f *skylarkFlowRegistry) extractPendingNodes(
	ctx context.Context,
	tenantID string,
	flowID int64,
	assignments []*core.Assignment,
	vertices map[int64]*core.FlowVertex,
) ([]*core.PendingNode, error) {
	// 1. 筛选待处理任务
	// 条件：category='processed' AND status='processing'
	pendingAssignments := make([]*core.Assignment, 0)
	for _, a := range assignments {
		if a.Category == core.AssignmentCategoryProcessed && a.Status == core.AssignmentStatusProcessing {
			pendingAssignments = append(pendingAssignments, a)
		}
	}

	// 2. 快速返回：没有待处理任务
	if len(pendingAssignments) == 0 {
		return []*core.PendingNode{}, nil
	}

	// 3. 按节点聚合（一个节点可能有多个待处理人）
	// 键：vertexID，值：任务列表
	vertexAssignmentsMap := make(map[int64][]*core.Assignment)
	for _, a := range pendingAssignments {
		vertexAssignmentsMap[a.VertexID] = append(vertexAssignmentsMap[a.VertexID], a)
	}

	// 4. 批量查询用户名
	// 4.1 收集所有唯一的用户ID
	uniqueUserIDs := collection.NewSet[string]()
	for _, nodeAssignments := range vertexAssignmentsMap {
		for _, a := range nodeAssignments {
			uniqueUserIDs.Add(a.AssigneeID)
		}
	}

	// 4.2 转换为数组
	userIDList := uniqueUserIDs.Keys()

	// 4.3 批量查询用户名（优先从缓存获取，未命中时查询远程数据库）
	remoteDB, err := f.getRemoteDB(ctx, tenantID)
	if err != nil {
		// 远程数据库连接失败不影响主流程，只记录日志
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "flows_pending_nodes"),
			logx.Field("tenant_id", tenantID),
			logx.Field("error", err.Error()),
		).Error("获取远程数据库连接失败，无法查询用户名")
	}

	var userNameMap map[string]string
	if remoteDB != nil {
		userNameMap, err = f.cache.BatchGetUserNames(ctx, remoteDB, tenantID, userIDList, 86400) // TTL: 24小时
		if err != nil {
			// 查询用户名失败不影响主流程，只记录日志
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "flows_pending_nodes"),
				logx.Field("tenant_id", tenantID),
				logx.Field("user_count", len(userIDList)),
				logx.Field("error", err.Error()),
			).Error("批量查询用户名失败")
			userNameMap = make(map[string]string) // 使用空映射
		}
	} else {
		userNameMap = make(map[string]string) // 使用空映射
	}

	// 5. 构建待处理节点列表
	pendingNodes := make([]*core.PendingNode, 0, len(vertexAssignmentsMap))
	for vertexID, nodeAssignments := range vertexAssignmentsMap {
		// 提取处理人ID列表
		assigneeIDs := make([]string, len(nodeAssignments))
		for i, a := range nodeAssignments {
			assigneeIDs[i] = a.AssigneeID
		}

		// 构建处理人姓名列表
		assigneeNames := make([]string, len(nodeAssignments))
		for i, a := range nodeAssignments {
			if name, ok := userNameMap[a.AssigneeID]; ok {
				assigneeNames[i] = name // 使用真实姓名
			} else {
				assigneeNames[i] = a.AssigneeID // 降级：使用用户ID
			}
		}

		// 获取节点名称（从传入的 vertices 映射）
		vertexName := ""
		if vertex, ok := vertices[vertexID]; ok {
			vertexName = vertex.Name
		}

		// 构建待处理节点
		pendingNode := &core.PendingNode{
			VertexID:      vertexID,
			VertexName:    vertexName,
			AssigneeIDs:   assigneeIDs,
			AssigneeNames: assigneeNames, // 使用批量查询的用户名
			CreatedAt:     nodeAssignments[0].CreatedAt,
		}

		pendingNodes = append(pendingNodes, pendingNode)
	}

	return pendingNodes, nil
}

// getJourneyAssignments 获取流程节点处理信息列表（内部方法）
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - journeyID: 流程记录ID
//
// 返回:
//   - []*core.Assignment: 任务列表
//   - error: 错误信息
func (f *skylarkFlowRegistry) getJourneyAssignments(
	ctx context.Context,
	tenantID string,
	journeyID int64,
) ([]*core.Assignment, error) {
	// 1. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. 构建 SkylarkAddress
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     "",
		AuthHeader: apiCfg.Token,
	}

	// 3. 构建 API URL: /api/v4/yaw/journeys/:journey_id/assignments
	apiURL := core.BuildJourneyAPIURL(skylarkAddress, journeyID, "assignments")

	// 4. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 5. 解析响应
	var assignmentResponses []AssignmentResponse
	if err := httputils.ReadJSONResponse(resp, &assignmentResponses); err != nil {
		return nil, err
	}

	// 6. 提取所有唯一的远程用户ID（使用通用函数）
	userIDMapping := core.ExtractUserIDsToMap(assignmentResponses, func(ar AssignmentResponse) int {
		return int(ar.AssigneeID)
	})

	// 7. 批量转换（远程ID → 本地ID），填充映射
	if len(userIDMapping) > 0 {
		if err := f.fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
			return nil, fmt.Errorf("批量转换用户ID失败: %w", err)
		}
	}

	// 8. 使用映射转换为领域模型
	assignments := make([]*core.Assignment, len(assignmentResponses))
	for i, ar := range assignmentResponses {
		assignments[i] = ar.ToDomain(userIDMapping)
	}

	return assignments, nil
}

// getJourneyDetail 获取流程记录详情（内部方法）
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - journeyID: 流程记录ID
//
// 返回:
//   - *core.JourneyDetail: 流程记录详情（包含字段值和附件）
//   - error: 错误信息（如果不存在返回 core.ErrJourneyNotFound）
func (f *skylarkFlowRegistry) getJourneyDetail(
	ctx context.Context,
	tenantID string,
	flowID int64,
	journeyID int64,
) (*core.JourneyDetail, error) {
	// 1. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. 构建 SkylarkAddress
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     "",
		AuthHeader: apiCfg.Token,
	}

	// 3. 构建 API URL: /api/v4/yaw/flows/:flow_id/journeys/:journey_id
	apiURL := core.BuildFlowAPIURL(skylarkAddress, flowID, "journeys", fmt.Sprintf("%d", journeyID))

	// 4. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 5. 解析响应
	var journeyDetailResp JourneyDetailResponse
	if err := httputils.ReadJSONResponse(resp, &journeyDetailResp); err != nil {
		// 如果是 404 错误，转换为 ErrJourneyNotFound
		if errors.Is(err, core.ErrSkylarkAPINotFound) {
			return nil, core.ErrJourneyNotFound
		}
		return nil, err
	}

	// 6. 获取字段映射（用于将字段ID转换为字段名）
	fieldMappings, err := f.getFlowFieldMappings(ctx, skylarkAddress, flowID)
	if err != nil {
		return nil, fmt.Errorf("获取字段映射失败: %w", err)
	}

	// 7. 提取远程用户ID（发起人）
	userIDMapping := map[int]string{
		int(journeyDetailResp.User.ID): "",
	}

	// 8. 批量转换（远程ID → 本地ID），填充映射
	if err := f.fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
		return nil, fmt.Errorf("批量转换用户ID失败: %w", err)
	}

	// 9. 使用映射转换为领域模型（传入字段映射）
	journeyDetail := journeyDetailResp.ToDomainWithFieldNames(userIDMapping, fieldMappings)

	return journeyDetail, nil
}

// getFlowDetail 获取流程详情（内部方法）
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//
// 返回:
//   - *core.FlowDetail: 流程详情（包含字段、节点、边信息）
//   - error: 错误信息（如果不存在返回 core.ErrFlowNotFound）
func (f *skylarkFlowRegistry) getFlowDetail(
	ctx context.Context,
	tenantID string,
	flowID int64,
) (*core.FlowDetail, error) {
	// 1. 尝试从缓存获取
	flowDetail, err := f.cache.GetFlowDetailAPI(ctx, tenantID, flowID)
	if err == nil && flowDetail != nil {
		// 缓存命中，直接返回
		return flowDetail, nil
	}
	// 如果缓存返回 ErrFlowNotFound，说明缓存中已有空值标记，直接返回错误（缓存穿透防护）
	if errors.Is(err, core.ErrFlowNotFound) {
		return nil, core.ErrFlowNotFound
	}
	// 缓存未命中或出错，继续执行，不影响正常流程
	if err != nil {
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "flows_flow_detail"),
			logx.Field("tenant_id", tenantID),
			logx.Field("flow_id", flowID),
			logx.Field("error", err.Error()),
		).Error("从缓存获取流程详情时发生错误")
	}

	// 2. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 3. 构建 SkylarkAddress
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     "",
		AuthHeader: apiCfg.Token,
	}

	// 4. 构建 API URL: /api/v4/yaw/flows/:flow_id
	apiURL := core.BuildFlowAPIURL(skylarkAddress, flowID)

	// 5. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 6. 解析响应
	var flowDetailResp FlowDetailResponse
	if err := httputils.ReadJSONResponse(resp, &flowDetailResp); err != nil {
		// 如果是 404 错误，转换为 ErrFlowNotFound 并缓存空值（缓存穿透防护）
		if errors.Is(err, core.ErrSkylarkAPINotFound) {
			// 异步缓存空值标记（不阻塞主流程）
			go func() {
				if cacheErr := f.cache.SetFlowDetailAPINull(context.Background(), tenantID, flowID); cacheErr != nil {
					logx.WithContext(context.Background()).WithFields(
						logx.Field("module", "flows_flow_detail"),
						logx.Field("tenant_id", tenantID),
						logx.Field("flow_id", flowID),
						logx.Field("error", cacheErr.Error()),
					).Error("缓存空值标记失败")
				}
			}()
			return nil, core.ErrFlowNotFound
		}
		return nil, err
	}

	// 7. 转换为领域模型
	flowDetail = flowDetailResp.ToDomain()

	// 8. 异步回写缓存（不阻塞主流程）
	go func() {
		ttl := f.config.FlowInfoCacheTTL
		if cacheErr := f.cache.SetFlowDetailAPI(context.Background(), tenantID, flowDetail, ttl); cacheErr != nil {
			logx.WithContext(context.Background()).WithFields(
				logx.Field("module", "flows_flow_detail"),
				logx.Field("tenant_id", tenantID),
				logx.Field("flow_id", flowID),
				logx.Field("error", cacheErr.Error()),
			).Error("保存流程详情到缓存时发生错误")
		}
	}()

	return flowDetail, nil
}

// getJourneyMoments 获取流程审批历史（内部方法）
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - journeyID: 流程记录ID
//
// 返回:
//   - []*core.Moment: 审批历史列表
//   - error: 错误信息
func (f *skylarkFlowRegistry) getJourneyMoments(
	ctx context.Context,
	tenantID string,
	journeyID int64,
) ([]*core.Moment, error) {
	// 1. 参数校验
	if journeyID <= 0 {
		return nil, fmt.Errorf("journeyID 必须大于 0")
	}

	// 2. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 3. 构建 SkylarkAPIContext
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     "",
		AuthHeader: apiCfg.Token,
	}

	// 4. 构建 API URL: GET /api/v4/yaw/journeys/:journey_id/moments
	apiURL := core.BuildJourneyMomentsAPIURL(skylarkAddress, journeyID)

	// 5. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 6. 处理 404 错误（流程记录不存在）
	if resp.StatusCode == http.StatusNotFound {
		return nil, core.ErrJourneyNotFound
	}

	// 7. 解析响应
	var momentResponses []MomentResponse
	if err := httputils.ReadJSONResponse(resp, &momentResponses); err != nil {
		return nil, err
	}

	// 8. 提取所有唯一的远程用户ID（使用通用函数）
	userIDMapping := core.ExtractUserIDsToMap(momentResponses, func(mr MomentResponse) int {
		if mr.User != nil {
			return int(mr.User.ID)
		}
		return 0
	})

	// 9. 批量转换（远程ID → 本地ID），填充映射
	if len(userIDMapping) > 0 {
		if err := f.fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
			return nil, fmt.Errorf("批量转换用户ID失败: %w", err)
		}
	}

	// 10. 使用映射转换为领域模型
	moments := make([]*core.Moment, len(momentResponses))
	for i, mr := range momentResponses {
		moments[i] = mr.ToDomain(userIDMapping)
	}

	// 10. 返回结果
	return moments, nil
}

// extractUniqueVertexIDs 从待处理节点列表中提取唯一的 vertexID
// 参数：
//   - pendingNodes: 待处理节点列表
//
// 返回：
//   - []int64: 唯一的 vertexID 列表
func extractUniqueVertexIDs(pendingNodes []*core.PendingNode) []int64 {
	if len(pendingNodes) == 0 {
		return nil
	}

	// 使用 TypedSet 去重
	vertexIDSet := collection.NewSet[int64]()
	for _, node := range pendingNodes {
		vertexIDSet.Add(node.VertexID)
	}

	return vertexIDSet.Keys()
}

// batchGetVertexDetails 批量获取节点详情（包含字段信息）
// 说明：优先从 Redis 缓存获取，未命中则并发调用 API，并异步回写缓存
// 参数：
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - vertexIDs: 节点ID列表
//
// 返回：
//   - map[int64][]*core.VertexField: 节点ID → 字段列表的映射
//   - error: 错误信息（部分失败不影响整体流程，只记录日志）
func (f *skylarkFlowRegistry) batchGetVertexDetails(
	ctx context.Context,
	tenantID string,
	flowID int64,
	vertexIDs []int64,
) (map[int64][]*core.VertexField, error) {
	if len(vertexIDs) == 0 {
		return make(map[int64][]*core.VertexField), nil
	}

	// 获取平台配置
	platformConfig, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取平台配置失败: %w", err)
	}

	// 结果映射
	result := make(map[int64][]*core.VertexField, len(vertexIDs))
	missedIDs := make([]int64, 0)

	// 1. 批量查询 Redis 缓存
	for _, vertexID := range vertexIDs {
		cacheKey := fmt.Sprintf("%s%s:%d:%d", core.CacheVertexKeyPrefix, tenantID, flowID, vertexID)

		fields, found, err := f.cache.GetVertexFieldsFromCache(ctx, cacheKey)
		if err != nil {
			// 缓存查询失败，记录日志并加入未命中列表
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "flows_vertex_fields"),
				logx.Field("cache_key", cacheKey),
				logx.Field("error", err.Error()),
			).Error("从缓存获取节点字段时发生错误")
			missedIDs = append(missedIDs, vertexID)
		} else if found {
			// 缓存命中
			result[vertexID] = fields
		} else {
			// 缓存未命中
			missedIDs = append(missedIDs, vertexID)
		}
	}

	// 2. 并发查询未命中的节点详情（使用 errgroup）
	if len(missedIDs) > 0 {
		type vertexResult struct {
			vertexID int64
			fields   []*core.VertexField
			err      error
		}

		resultChan := make(chan vertexResult, len(missedIDs))

		// 并发查询（限制最大并发数为 10）
		maxConcurrent := 10
		if len(missedIDs) < maxConcurrent {
			maxConcurrent = len(missedIDs)
		}

		semaphore := make(chan struct{}, maxConcurrent)
		for _, vertexID := range missedIDs {
			semaphore <- struct{}{} // 获取信号量
			go func(vID int64) {
				defer func() { <-semaphore }() // 释放信号量

				fields, err := f.getVertexDetail(ctx, platformConfig, flowID, vID)
				resultChan <- vertexResult{vertexID: vID, fields: fields, err: err}
			}(vertexID)
		}

		// 等待所有查询完成
		for i := 0; i < len(missedIDs); i++ {
			res := <-resultChan
			if res.err != nil {
				// 查询失败，记录日志但不影响整体流程
				logx.WithContext(ctx).WithFields(
					logx.Field("module", "flows_vertex_fields"),
					logx.Field("flow_id", flowID),
					logx.Field("vertex_id", res.vertexID),
					logx.Field("error", res.err.Error()),
				).Error("查询节点详情失败")
			} else {
				// 查询成功，添加到结果
				result[res.vertexID] = res.fields

				// 3. 异步回写缓存（使用 context.Background() 避免主请求取消影响缓存）
				go func(vID int64, fields []*core.VertexField) {
					cacheKey := fmt.Sprintf("%s%s:%d:%d", core.CacheVertexKeyPrefix, tenantID, flowID, vID)
					ttl := f.config.VertexFieldCacheTTL
					if err := f.cache.SaveVertexFieldsToCache(context.Background(), cacheKey, fields, ttl); err != nil {
						logx.WithContext(context.Background()).WithFields(
							logx.Field("module", "flows_vertex_fields"),
							logx.Field("cache_key", cacheKey),
							logx.Field("error", err.Error()),
						).Error("保存节点字段到缓存时发生错误")
					}
				}(res.vertexID, res.fields)
			}
		}

		close(resultChan)
	}

	return result, nil
}

// getVertexDetail 获取单个节点详情（调用 Skylark API）
// 参数：
//   - ctx: 上下文
//   - apiConfig: API 配置（包含认证信息）
//   - flowID: 流程ID
//   - vertexID: 节点ID
//
// 返回：
//   - []*core.VertexField: 节点字段列表
//   - error: 错误信息
func (f *skylarkFlowRegistry) getVertexDetail(
	ctx context.Context,
	apiConfig *core.SkylarkAPIConfig,
	flowID int64,
	vertexID int64,
) ([]*core.VertexField, error) {
	// 构建 SkylarkAPIContext
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiConfig.App,
		UserID:     "",
		AuthHeader: apiConfig.Token,
	}

	// 构建 API URL: /api/v4/yaw/flows/:flow_id/vertices/:id
	apiURL := core.BuildFlowAPIURL(skylarkAddress, flowID, "vertices", fmt.Sprintf("%d", vertexID))

	// 发起 GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 解析响应
	var vertexDetail VertexDetailResponse
	if err := httputils.ReadJSONResponse(resp, &vertexDetail); err != nil {
		return nil, err
	}

	// 转换为领域模型
	return vertexDetail.ToDomain(), nil
}

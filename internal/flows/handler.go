package flows

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/httputils"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpc"
)

// skylarkFlowRegistry 流程注册表结构
type skylarkFlowRegistry struct {
	config                Config                         // 配置
	cache                 core.CacheInterface            // 缓存接口
	getPlatformConfig     core.GetPlatformConfigFunc     // 获取平台配置的函数（依赖注入）
	getRemoteUserIDs      core.GetRemoteUserIDsFunc      // 获取远程用户ID的函数（依赖注入，用于入参转换）
	fillLocalUserIDMap    core.FillLocalUserIDMapFunc    // 批量反向转换函数（依赖注入，用于出参转换）
	getRemoteDB           core.GetRemoteDBFunc           // 获取远程数据库连接的函数（依赖注入，用于性能优化）
	listConfiguredFlowIDs core.ListConfiguredFlowIDsFunc // 获取已配置事件的flow_id列表的函数（依赖注入，用于筛选流程实例）
}

// newSkylarkFlowRegistry 创建新的流程注册表
func newSkylarkFlowRegistry(config Config, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc, getRemoteUserIDs core.GetRemoteUserIDsFunc, fillLocalUserIDMap core.FillLocalUserIDMapFunc, getRemoteDB core.GetRemoteDBFunc, listConfiguredFlowIDs core.ListConfiguredFlowIDsFunc) (*skylarkFlowRegistry, error) {
	if getPlatformConfig == nil {
		return nil, fmt.Errorf("getPlatformConfig 不能为 nil")
	}

	if getRemoteUserIDs == nil {
		return nil, fmt.Errorf("getRemoteUserIDs 不能为 nil")
	}

	if fillLocalUserIDMap == nil {
		return nil, fmt.Errorf("fillLocalUserIDMap 不能为 nil")
	}

	if getRemoteDB == nil {
		return nil, fmt.Errorf("getRemoteDB 不能为 nil")
	}

	if listConfiguredFlowIDs == nil {
		return nil, fmt.Errorf("listConfiguredFlowIDs 不能为 nil")
	}

	// 设置缓存配置的默认值
	if config.FlowInfoCacheTTL <= 0 {
		config.FlowInfoCacheTTL = 3600 // 默认1小时
	}
	if config.VertexInfoCacheTTL <= 0 {
		config.VertexInfoCacheTTL = 3600 // 默认1小时
	}
	if config.VertexFieldCacheTTL <= 0 {
		config.VertexFieldCacheTTL = 3600 // 默认1小时
	}

	return &skylarkFlowRegistry{
		config:                config,
		cache:                 cache,
		getPlatformConfig:     getPlatformConfig,
		getRemoteUserIDs:      getRemoteUserIDs,
		fillLocalUserIDMap:    fillLocalUserIDMap,
		getRemoteDB:           getRemoteDB,
		listConfiguredFlowIDs: listConfiguredFlowIDs,
	}, nil
}

// CreateFlow 创建并启动一个流程
// 参数:
//   - ctx: 上下文
//   - app: 应用名称
//   - flowID: 流程ID
//   - userID: 用户ID
//   - authHeader: 认证头信息
//   - data: 流程数据
func (f *skylarkFlowRegistry) CreateFlow(ctx context.Context, app string, flowID int64, userID int64, authHeader string, data map[string]core.TypedValue) error {
	// 构建流程地址信息
	skylarkFlowAddress := core.SkylarkAPIContext{
		App:        app,
		UserID:     strconv.FormatInt(userID, 10),
		AuthHeader: authHeader,
	}

	// 获取字段映射
	FlowFieldMappings, err := f.getFlowFieldMappings(ctx, skylarkFlowAddress, flowID)
	if err != nil {
		return err
	}

	// 构建 Skylark 流程路由请求体
	flowRouteRequest, err := f.buildFlowRouteRequest(ctx, skylarkFlowAddress, flowID, data, FlowFieldMappings)
	if err != nil {
		return err
	}

	// 构建API请求URL
	apiURL := core.BuildFlowAPIURL(skylarkFlowAddress, flowID, "journeys")

	// 准备工作完成，现在获取分布式锁
	lockKey := fmt.Sprintf("flow:lock:%s:%d:%d", app, flowID, userID)
	lockValue := uuid.New().String()
	lockExpiry := 30 // 默认30秒过期时间

	// 获取分布式锁，支持重试
	if err = f.cache.AcquireLockWithRetry(ctx, lockKey, lockValue, lockExpiry); err != nil {
		return err
	}

	// 确保释放锁
	defer func() {
		if releaseErr := f.cache.ReleaseLock(ctx, lockKey, lockValue); releaseErr != nil {
			// 记录释放锁失败的错误，但不影响主流程的返回
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "flows_create"),
				logx.Field("lock_key", lockKey),
				logx.Field("app", app),
				logx.Field("flow_id", flowID),
				logx.Field("user_id", userID),
				logx.Field("error", releaseErr.Error()),
			).Error("释放分布式锁失败")
		}
	}()

	// 发送 Skylark 流程路由请求
	routeFlowResult, err := httpc.Do(ctx, http.MethodPost, apiURL, flowRouteRequest)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer routeFlowResult.Body.Close()

	// 使用 httputils 统一处理响应并解析为 FlowRouteResponse
	var routeResp FlowRouteResponse
	if err := httputils.ReadJSONResponse(routeFlowResult, &routeResp); err != nil {
		return err
	}

	// 检查 NextVertices 是否为空
	if len(routeResp.NextVertices) == 0 {
		return fmt.Errorf("%w: 响应为: %+v", core.ErrNoNextVertices, routeResp)
	}

	// 构建提议请求
	userIDInt, err := strconv.Atoi(skylarkFlowAddress.UserID)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrUserIDConversionFailed, err)
	}
	flowProposeRequest := FlowProposeRequest{
		Assignment: ProposeAssignment{
			Operation:          string(core.OperationPropose),
			NextVertexID:       routeResp.NextVertices[0].NextVerticesID,
			DurationThresholds: []map[string]string{},
		},
		UserID: userIDInt,
		Webhook: Webhook{
			PayloadURL:       "",
			SubscribedEvents: []string{core.EventJourneyStatus},
		},
		Token: skylarkFlowAddress.AuthHeader,
	}

	// 发送 Skylark 流程提议请求
	proposeFlowResult, err := httpc.Do(ctx, http.MethodPost, apiURL, flowProposeRequest)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer proposeFlowResult.Body.Close()

	// 使用 httputils 统一处理响应（propose 请求无需解析响应体）
	if err := httputils.ReadJSONResponse(proposeFlowResult, nil); err != nil {
		return err
	}

	return nil
}

// UpdateJourneyStatus 更新流程任务状态
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID（用于获取字段映射）
//   - journeyID: 流程记录ID
//   - assignmentID: 任务ID
//   - localUserID: 本地用户ID（操作人，SDK自动转换为远程用户ID）
//   - operation: 操作类型（使用 core.OperationApprove 等常量）
//   - options: 可选参数
func (f *skylarkFlowRegistry) UpdateJourneyStatus(
	ctx context.Context,
	tenantID string,
	flowID int64,
	journeyID int64,
	assignmentID int64,
	localUserID string,
	operation core.JourneyOperation,
	options core.UpdateJourneyStatusOptions,
) error {
	// 1. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return err
	}

	// 2. 转换本地用户ID为远程用户ID
	remoteUserIDs, err := f.getRemoteUserIDs(ctx, tenantID, []string{localUserID})
	if err != nil {
		return fmt.Errorf("转换用户ID失败: %w", err)
	}
	if len(remoteUserIDs) == 0 {
		return fmt.Errorf("本地用户ID %s 未找到对应的远程用户ID", localUserID)
	}
	remoteUserID := remoteUserIDs[0]

	// 3. 从配置中构建 SkylarkAddress
	skylarkFlowAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     strconv.Itoa(remoteUserID), // 操作人ID（远程）
		AuthHeader: apiCfg.Token,
	}

	// 构建API请求URL
	apiURL := core.BuildJourneyAssignmentAPIURL(skylarkFlowAddress, journeyID, assignmentID)

	// 准备工作完成，现在获取分布式锁
	lockKey := fmt.Sprintf("journey:lock:%s:%d:%d:%d", tenantID, journeyID, assignmentID, remoteUserID)
	lockValue := uuid.New().String()
	lockExpiry := 30 // 默认30秒过期时间

	// 获取分布式锁，支持重试
	if err := f.cache.AcquireLockWithRetry(ctx, lockKey, lockValue, lockExpiry); err != nil {
		return err
	}

	// 确保释放锁
	defer func() {
		if releaseErr := f.cache.ReleaseLock(ctx, lockKey, lockValue); releaseErr != nil {
			// 记录释放锁失败的错误，但不影响主流程的返回
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "flows_update_status"),
				logx.Field("lock_key", lockKey),
				logx.Field("tenant_id", tenantID),
				logx.Field("journey_id", journeyID),
				logx.Field("assignment_id", assignmentID),
				logx.Field("operation", string(operation)),
				logx.Field("error", releaseErr.Error()),
			).Error("释放分布式锁失败")
		}
	}()

	// 第一次请求：修改数据（route操作）
	// 初始化字段映射（如果有数据需要处理）
	var fieldMappings map[string]core.FieldMapping
	if len(options.Data) > 0 {
		var err error
		fieldMappings, err = f.getFlowFieldMappings(ctx, skylarkFlowAddress, flowID)
		if err != nil {
			return err
		}
	}

	// 构建第一次请求：修改数据
	routeRequest, err := f.buildRouteRequestForUpdate(
		ctx,
		skylarkFlowAddress,
		remoteUserID,
		options.Data,
		fieldMappings,
	)
	if err != nil {
		return err
	}

	// 发送第一次请求
	routeResult, err := httpc.Do(ctx, http.MethodPut, apiURL, routeRequest)
	if err != nil {
		return fmt.Errorf("%w: 第一次请求失败: %v", core.ErrHTTPRequestFailed, err)
	}
	defer routeResult.Body.Close()

	// 解析第一次请求的响应，获取 next_vertices
	var routeResp FlowRouteResponse
	if err := httputils.ReadJSONResponse(routeResult, &routeResp); err != nil {
		return fmt.Errorf("第一次请求失败: %w", err)
	}

	// 检查 NextVertices 是否为空
	if len(routeResp.NextVertices) == 0 {
		return fmt.Errorf("%w: 响应为: %+v", core.ErrNoNextVertices, routeResp)
	}

	// 从响应中获取下一个节点ID
	nextVertexID := routeResp.NextVertices[0].NextVerticesID

	// 第二次请求：执行操作（approve/refuse/transfer/cancel）
	// 转换抄送者本地用户ID为远程用户ID
	var carbonCopyRemoteUserIDs []int
	if len(options.CarbonCopyUserIDs) > 0 {
		remoteIDs, err := f.getRemoteUserIDs(ctx, tenantID, options.CarbonCopyUserIDs)
		if err != nil {
			return fmt.Errorf("转换抄送者用户ID失败: %w", err)
		}
		carbonCopyRemoteUserIDs = remoteIDs
	}

	operationRequest, err := f.buildOperationRequest(
		skylarkFlowAddress,
		remoteUserID,      // 操作人的远程用户ID
		string(operation), // 转换为字符串
		nextVertexID,      // 使用第一次响应中的 next_vertex_id
		options.Comment,
		carbonCopyRemoteUserIDs, // 使用远程用户ID
	)
	if err != nil {
		return err
	}

	// 发送第二次请求
	operationResult, err := httpc.Do(ctx, http.MethodPut, apiURL, operationRequest)
	if err != nil {
		return fmt.Errorf("%w: 第二次请求失败: %v", core.ErrHTTPRequestFailed, err)
	}
	defer operationResult.Body.Close()

	// 使用 httputils 统一处理第二次请求的响应（operation 请求无需解析响应体）
	if err := httputils.ReadJSONResponse(operationResult, nil); err != nil {
		return fmt.Errorf("第二次请求失败: %w", err)
	}

	return nil
}

// GetUserAssignments 获取用户处理的任务列表
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - localUserID: 本地用户ID（SDK内部自动转换为远程用户ID）
//   - category: 任务类别（使用 core.AssignmentCategoryXXX 常量）
//   - page: 页码（从1开始）
//   - pageSize: 每页数量
//
// 返回:
//   - []*core.Assignment: 任务列表（已补充 flow_id 和 flow_title）
//   - int: 总数
//   - error: 错误信息
func (f *skylarkFlowRegistry) GetUserAssignments(ctx context.Context, tenantID string, localUserID string, category string, page, pageSize int) ([]*core.Assignment, int, error) {
	// 1. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	// 2. 转换本地用户ID为远程用户ID
	remoteUserIDs, err := f.getRemoteUserIDs(ctx, tenantID, []string{localUserID})
	if err != nil {
		return nil, 0, fmt.Errorf("获取用户任务失败: 查询用户映射时发生错误 (%w)", err)
	}
	if len(remoteUserIDs) == 0 {
		return nil, 0, fmt.Errorf("获取用户任务失败: 用户 %s 未同步到 Skylark，请先调用 UserManager.CreateUser() 创建用户映射后重试", localUserID)
	}
	remoteUserID := remoteUserIDs[0]

	// 3. 构建 SkylarkAPIContext
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     "",
		AuthHeader: apiCfg.Token,
	}

	// 4. 构建 API URL: /api/v4/yaw/flows/user_assignments.json
	apiURL := core.BuildUserAssignmentsURL(skylarkAddress)

	// 5. 构建查询参数
	apiURL = fmt.Sprintf("%s?user_id=%d&category=%s&page=%d&per_page=%d",
		apiURL, remoteUserID, category, page, pageSize)

	// 5. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 6. 解析响应（直接解析为数组）
	var assignmentsResp UserAssignmentsResponse
	if err := httputils.ReadJSONResponse(resp, &assignmentsResp); err != nil {
		return nil, 0, err
	}

	// 7. 提取所有唯一的远程用户ID（使用通用函数）
	userIDMapping := core.ExtractUserIDsToMap(assignmentsResp, func(ar AssignmentResponse) int {
		return int(ar.AssigneeID)
	})

	// 8. 批量转换（远程ID → 本地ID），填充映射
	if len(userIDMapping) > 0 {
		if err := f.fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
			return nil, 0, fmt.Errorf("获取用户任务失败: 任务中存在未同步的 Skylark 用户，无法转换为本地用户ID (%w)", err)
		}
	}

	// 9. 使用映射转换为领域模型
	assignments := make([]*core.Assignment, len(assignmentsResp))
	for i, assignmentResp := range assignmentsResp {
		assignments[i] = assignmentResp.ToDomain(userIDMapping)
	}

	// 9. 性能优化：自动补充 flow_id 和 flow_title（注：筛选后总数无需从响应头获取）
	if err := f.enrichAssignmentsWithFlowInfo(ctx, tenantID, assignments); err != nil {
		return nil, 0, err
	}

	// 11. 筛选：只保留已配置事件的任务
	trueVal := true
	configuredFlowIDs, err := f.listConfiguredFlowIDs(ctx, tenantID, &trueVal) // 只获取已启用的事件
	if err != nil {
		return nil, 0, fmt.Errorf("获取已配置流程列表失败: %w", err)
	}

	// 构建快速查找集合
	flowIDSet := make(map[int64]bool, len(configuredFlowIDs))
	for _, flowID := range configuredFlowIDs {
		flowIDSet[int64(flowID)] = true
	}

	// 过滤 assignments（只保留已配置事件的任务）
	filteredAssignments := make([]*core.Assignment, 0, len(assignments))
	for _, a := range assignments {
		if a.FlowID != nil && flowIDSet[*a.FlowID] {
			filteredAssignments = append(filteredAssignments, a)
		}
	}

	// 12. 返回筛选后的结果
	return filteredAssignments, len(filteredAssignments), nil
}

// GetProposedJourneys 获取用户发起的流程列表
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - localUserID: 本地用户ID（SDK内部自动转换为远程用户ID）
//   - page: 页码（从1开始）
//   - pageSize: 每页数量
//
// 返回:
//   - []*core.Journey: 流程列表
//   - int: 总数
//   - error: 错误信息
func (f *skylarkFlowRegistry) GetProposedJourneys(ctx context.Context, tenantID string, flowID int64, localUserID string, page, pageSize int) ([]*core.Journey, int, error) {
	// 1. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	// 2. 转换本地用户ID为远程用户ID
	remoteUserIDs, err := f.getRemoteUserIDs(ctx, tenantID, []string{localUserID})
	if err != nil {
		return nil, 0, fmt.Errorf("获取用户发起的流程失败: 查询用户映射时发生错误 (%w)", err)
	}
	if len(remoteUserIDs) == 0 {
		return nil, 0, fmt.Errorf("获取用户发起的流程失败: 用户 %s 未同步到 Skylark，请先调用 UserManager.CreateUser() 创建用户映射后重试", localUserID)
	}
	remoteUserID := remoteUserIDs[0]

	// 3. 构建 SkylarkAPIContext
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     "",
		AuthHeader: apiCfg.Token,
	}

	// 4. 构建 API URL: /api/v4/yaw/flows/:flow_id/journeys/proposed_journeys
	apiURL := core.BuildProposedJourneysURL(skylarkAddress, flowID)

	// 5. 构建查询参数
	apiURL = fmt.Sprintf("%s?user_id=%d&page=%d&per_page=%d",
		apiURL, remoteUserID, page, pageSize)

	// 5. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 6. 解析响应（直接解析为数组）
	var journeysResp ProposedJourneysResponse
	if err := httputils.ReadJSONResponse(resp, &journeysResp); err != nil {
		return nil, 0, err
	}

	// 7. 转换为领域模型（包含批量用户ID转换）
	journeys, err := convertJourneyResponsesToDomain(ctx, journeysResp, f.fillLocalUserIDMap, tenantID)
	if err != nil {
		return nil, 0, err
	}

	// 8. 筛选：只保留已配置事件的流程（注：筛选后总数无需从响应头获取）
	trueVal := true
	configuredFlowIDs, err := f.listConfiguredFlowIDs(ctx, tenantID, &trueVal) // 只获取已启用的事件
	if err != nil {
		return nil, 0, fmt.Errorf("获取已配置流程列表失败: %w", err)
	}

	// 构建快速查找集合
	flowIDSet := make(map[int64]bool, len(configuredFlowIDs))
	for _, flowID := range configuredFlowIDs {
		flowIDSet[int64(flowID)] = true
	}

	// 过滤 journeys（只保留已配置事件的流程）
	filteredJourneys := make([]*core.Journey, 0, len(journeys))
	for _, j := range journeys {
		if flowIDSet[j.FlowID] {
			filteredJourneys = append(filteredJourneys, j)
		}
	}

	// 10. 返回筛选后的结果
	return filteredJourneys, len(filteredJourneys), nil
}

// SearchJourneys 搜索流程记录
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - req: 搜索请求（req.InitiatorID 为本地用户ID，SDK内部自动转换为远程用户ID）
//
// 返回:
//   - []*core.Journey: 流程列表
//   - int: 总数
//   - error: 错误信息
func (f *skylarkFlowRegistry) SearchJourneys(ctx context.Context, tenantID string, req *core.JourneySearchRequest) ([]*core.Journey, int, error) {
	// 1. 转换发起人ID（本地用户ID → 远程用户ID）
	var remoteInitiatorID *int64
	if req.InitiatorID != "" {
		remoteUserIDs, err := f.getRemoteUserIDs(ctx, tenantID, []string{req.InitiatorID})
		if err != nil {
			return nil, 0, fmt.Errorf("转换发起人ID失败: %w", err)
		}
		if len(remoteUserIDs) == 0 {
			return nil, 0, fmt.Errorf("本地用户ID %s 未找到对应的远程用户ID", req.InitiatorID)
		}
		remoteID := int64(remoteUserIDs[0])
		remoteInitiatorID = &remoteID
	}

	// 2. 参数校验
	if req.Page < 1 {
		return nil, 0, fmt.Errorf("page 必须大于等于 1")
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		return nil, 0, fmt.Errorf("pageSize 必须在 1-100 之间")
	}
	if req.FlowID <= 0 {
		return nil, 0, fmt.Errorf("flowID 必须大于 0")
	}

	// 3. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, 0, err
	}

	// 4. 构建 SkylarkAPIContext
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     "",
		AuthHeader: apiCfg.Token,
	}

	// 5. 构建 API URL: POST /api/v4/yaw/flows/:id/journeys/search
	apiURL := core.BuildJourneySearchURL(skylarkAddress, req.FlowID)

	// 6. 构建请求体
	searchBody := SearchJourneysRequest{
		Page:        req.Page,
		PerPage:     req.PageSize,
		Status:      req.Status,
		Keyword:     req.Keyword,
		InitiatorID: remoteInitiatorID,
		CreatedFrom: req.CreatedFrom,
		CreatedTo:   req.CreatedTo,
		Token:       skylarkAddress.AuthHeader,
	}

	// 6. 发送 HTTP POST 请求
	resp, err := httpc.Do(ctx, http.MethodPost, apiURL, searchBody)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 7. 处理 404 错误（流程不存在）
	if resp.StatusCode == http.StatusNotFound {
		return nil, 0, core.ErrFlowNotFound
	}

	// 8. 解析响应（直接解析为数组）
	var searchResp JourneySearchAPIResponse
	if err := httputils.ReadJSONResponse(resp, &searchResp); err != nil {
		return nil, 0, err
	}

	// 9. 转换为领域模型（包含批量用户ID转换）
	journeys, err := convertJourneyResponsesToDomain(ctx, []JourneyResponse(searchResp), f.fillLocalUserIDMap, tenantID)
	if err != nil {
		return nil, 0, err
	}

	// 10. 筛选：只保留已配置事件的流程（注：筛选后总数无需从响应头获取）
	trueVal := true
	configuredFlowIDs, err := f.listConfiguredFlowIDs(ctx, tenantID, &trueVal) // 只获取已启用的事件
	if err != nil {
		return nil, 0, fmt.Errorf("获取已配置流程列表失败: %w", err)
	}

	// 构建快速查找集合
	flowIDSet := make(map[int64]bool, len(configuredFlowIDs))
	for _, flowID := range configuredFlowIDs {
		flowIDSet[int64(flowID)] = true
	}

	// 过滤 journeys（只保留已配置事件的流程）
	filteredJourneys := make([]*core.Journey, 0, len(journeys))
	for _, j := range journeys {
		if flowIDSet[j.FlowID] {
			filteredJourneys = append(filteredJourneys, j)
		}
	}

	// 12. 返回筛选后的结果
	return filteredJourneys, len(filteredJourneys), nil
}

// AbortJourney 终止流程任务
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - journeyID: 流程记录ID
//
// 返回:
//   - error: 错误信息
func (f *skylarkFlowRegistry) AbortJourney(
	ctx context.Context,
	tenantID string,
	flowID int64,
	journeyID int64,
) error {
	// 1. 参数校验
	if flowID <= 0 {
		return fmt.Errorf("flowID 必须大于 0")
	}
	if journeyID <= 0 {
		return fmt.Errorf("journeyID 必须大于 0")
	}

	// 2. 获取API配置（已验证APIBaseURL、APIToken）
	apiCfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return err
	}

	// 3. 构建 SkylarkAPIContext
	skylarkAddress := core.SkylarkAPIContext{
		App:        apiCfg.App,
		UserID:     "",
		AuthHeader: apiCfg.Token,
	}

	// 4. 构建 API URL: PUT /api/v4/yaw/flows/:flow_id/journeys/:id
	apiURL := core.BuildAbortJourneyURL(skylarkAddress, flowID, journeyID)

	// 5. 构建请求体
	requestBody := AbortJourneyRequest{
		Status: core.StatusAborted,
		Token:  skylarkAddress.AuthHeader,
	}

	// 6. 发送 HTTP PUT 请求
	resp, err := httpc.Do(ctx, http.MethodPut, apiURL, requestBody)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 7. 处理 404 错误（流程或流程记录不存在）
	if resp.StatusCode == http.StatusNotFound {
		return core.ErrJourneyNotFound
	}

	// 8. 解析响应（验证终止成功）
	var abortResp AbortJourneyResponse
	if err := httputils.ReadJSONResponse(resp, &abortResp); err != nil {
		return err
	}

	// 9. 验证状态是否已更新为 aborted
	if abortResp.Status != core.StatusAborted {
		return fmt.Errorf("终止流程失败: 期望状态为 aborted，实际为 %s", abortResp.Status)
	}

	// 10. 返回成功
	return nil
}

// GetJourneyFullDetail 获取流程完整详情（一站式接口）
// 功能：
//   - 聚合基础信息、业务数据、审批历史、待处理节点、节点信息
//   - 减少前端调用次数（4次 → 1次）
//   - 自动补充节点名称
//   - 使用并发查询优化性能（耗时从 ~310ms 降至 ~100-120ms）
//
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - journeyID: 流程记录ID
//
// 返回:
//   - *core.JourneyFullDetail: 流程完整详情（包含所有维度信息）
//   - error: 错误信息
func (f *skylarkFlowRegistry) GetJourneyFullDetail(
	ctx context.Context,
	tenantID string,
	flowID int64,
	journeyID int64,
) (*core.JourneyFullDetail, error) {
	// 1. 参数校验
	if flowID <= 0 {
		return nil, fmt.Errorf("flowID 必须大于 0")
	}
	if journeyID <= 0 {
		return nil, fmt.Errorf("journeyID 必须大于 0")
	}

	// 2. 并发组1：获取基础信息 + 流程详情（必需）
	// 这两个接口独立且都是必需的，可以并发执行
	var (
		basicInfo  *core.JourneyDetail
		flowDetail *core.FlowDetail
		err1, err2 error
	)

	// 使用 channel 实现并发查询
	done := make(chan struct{})

	// 并发查询基础信息
	go func() {
		basicInfo, err1 = f.getJourneyDetail(ctx, tenantID, flowID, journeyID)
		done <- struct{}{}
	}()

	// 并发查询流程详情
	go func() {
		flowDetail, err2 = f.getFlowDetail(ctx, tenantID, flowID)
		done <- struct{}{}
	}()

	// 等待两个查询完成
	<-done
	<-done

	// 检查错误（基础信息和流程详情都是必需的）
	if err1 != nil {
		return nil, fmt.Errorf("获取流程基础信息失败: %w", err1)
	}
	if err2 != nil {
		return nil, fmt.Errorf("获取流程详情失败: %w", err2)
	}

	// 3. 构建节点信息映射
	vertices := make(map[int64]*core.FlowVertex, len(flowDetail.Vertices))
	for _, vertex := range flowDetail.Vertices {
		vertices[vertex.ID] = vertex
	}

	// 4. 并发组2：获取审批历史 + 任务列表
	// 审批历史（可选）和任务列表（必需）可以并发执行
	var (
		history     []*core.Moment
		assignments []*core.Assignment
		err3, err4  error
	)

	done2 := make(chan struct{})

	// 并发查询审批历史（失败不影响主流程）
	go func() {
		history, err3 = f.getJourneyMoments(ctx, tenantID, journeyID)
		if err3 != nil {
			// 审批历史失败不影响整体流程，只记录日志
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "flows_full_detail"),
				logx.Field("journey_id", journeyID),
				logx.Field("error", err3.Error()),
			).Error("获取审批历史失败")
			history = []*core.Moment{} // 返回空列表
		}
		done2 <- struct{}{}
	}()

	// 并发查询任务列表（必需）
	go func() {
		assignments, err4 = f.getJourneyAssignments(ctx, tenantID, journeyID)
		done2 <- struct{}{}
	}()

	// 等待两个查询完成
	<-done2
	<-done2

	// 检查任务列表错误（必需）
	if err4 != nil {
		return nil, fmt.Errorf("获取任务列表失败: %w", err4)
	}

	// 5. 过滤并补充审批历史
	// 说明：
	//   - 只保留已处理的记录（排除 status="processing" 的待处理节点）
	//   - 自动补充节点名称（vertexName）
	filteredHistory := make([]*core.Moment, 0, len(history))
	for _, moment := range history {
		// 过滤：只保留已处理的记录
		if moment.StatusKey == core.StatusProcessing {
			continue // 跳过待处理节点（这些节点应该在 PendingNodes 中）
		}

		// 补充节点名称
		if vertex, ok := vertices[moment.VertexID]; ok {
			moment.VertexName = &vertex.Name
		}

		filteredHistory = append(filteredHistory, moment)
	}

	// 6. 提取待处理节点
	pendingNodes, err := f.extractPendingNodes(assignments, vertices)
	if err != nil {
		// 待处理节点提取失败不影响整体流程，只记录日志
		logx.WithContext(ctx).WithFields(
			logx.Field("module", "flows_full_detail"),
			logx.Field("journey_id", journeyID),
			logx.Field("error", err.Error()),
		).Error("提取待处理节点失败")
		pendingNodes = []*core.PendingNode{} // 返回空列表
	}

	// 7. 填充待处理节点的字段信息（批量查询优化）
	// 说明：
	//   - 从 Skylark API 获取节点字段列表（缓存优先）
	//   - 用于前端动态渲染表单，构造 UpdateJourneyStatus 的 Data 参数
	//   - 查询失败不影响主流程（Fields 为 nil）
	if len(pendingNodes) > 0 {
		// 7.1 提取唯一的 vertexID
		vertexIDs := extractUniqueVertexIDs(pendingNodes)

		// 7.2 批量查询节点详情
		vertexFieldsMap, err := f.batchGetVertexDetails(ctx, tenantID, flowID, vertexIDs)
		if err != nil {
			// 批量查询失败，记录日志但不影响主流程
			logx.WithContext(ctx).WithFields(
				logx.Field("module", "flows_full_detail"),
				logx.Field("flow_id", flowID),
				logx.Field("journey_id", journeyID),
				logx.Field("error", err.Error()),
			).Error("批量查询节点字段失败")
		} else {
			// 7.3 填充每个待处理节点的字段信息
			for _, node := range pendingNodes {
				if fields, ok := vertexFieldsMap[node.VertexID]; ok {
					node.Fields = fields
				}
			}
		}
	}

	// 8. 返回完整详情
	return &core.JourneyFullDetail{
		BasicInfo:    basicInfo,
		History:      filteredHistory, // 使用过滤并补充后的历史记录
		PendingNodes: pendingNodes,
		Vertices:     vertices,
	}, nil
}

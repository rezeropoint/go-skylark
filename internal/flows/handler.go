package flows

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/httputils"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// skylarkFlowRegistry 流程注册表结构
type skylarkFlowRegistry struct {
	config               *Config                          // 配置
	cache                core.CacheInterface              // 缓存接口
	getPlatformConfig    core.GetPlatformConfigFunc       // 获取平台配置的函数（依赖注入）
	getRemoteUserIDs     core.GetRemoteUserIDsFunc        // 获取远程用户ID的函数（依赖注入，用于入参转换）
	fillLocalUserIDMap   core.FillLocalUserIDMapFunc      // 批量反向转换函数（依赖注入，用于出参转换）
	getRemoteDB          core.GetRemoteDBFunc             // 获取远程数据库连接的函数（依赖注入，用于性能优化）
	listConfiguredFlowIDs core.ListConfiguredFlowIDsFunc  // 获取已配置事件的flow_id列表的函数（依赖注入，用于筛选流程实例）
}

// newSkylarkFlowRegistry 创建新的流程注册表
func newSkylarkFlowRegistry(config *Config, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc, getRemoteUserIDs core.GetRemoteUserIDsFunc, fillLocalUserIDMap core.FillLocalUserIDMapFunc, getRemoteDB core.GetRemoteDBFunc, listConfiguredFlowIDs core.ListConfiguredFlowIDsFunc) (*skylarkFlowRegistry, error) {
	if config == nil {
		return nil, core.ErrConfigNil
	}

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

	return &skylarkFlowRegistry{
		config:               config,
		cache:                cache,
		getPlatformConfig:    getPlatformConfig,
		getRemoteUserIDs:     getRemoteUserIDs,
		fillLocalUserIDMap:   fillLocalUserIDMap,
		getRemoteDB:          getRemoteDB,
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
			// 这里可以添加日志记录
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
			// 这里可以添加日志记录
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
		options.Data,
		fieldMappings,
	)
	if err != nil {
		return err
	}

	// 发送第一次请求
	routeResult, err := httpc.Do(ctx, http.MethodPost, apiURL, routeRequest)
	if err != nil {
		return fmt.Errorf("%w: 第一次请求失败: %v", core.ErrHTTPRequestFailed, err)
	}
	defer routeResult.Body.Close()

	// 使用 httputils 统一处理第一次请求的响应（route 请求无需解析响应体）
	if err := httputils.ReadJSONResponse(routeResult, nil); err != nil {
		return fmt.Errorf("第一次请求失败: %w", err)
	}

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
		string(operation), // 转换为字符串
		options.NextVertexID,
		options.Comment,
		carbonCopyRemoteUserIDs, // 使用远程用户ID
	)
	if err != nil {
		return err
	}

	// 发送第二次请求
	operationResult, err := httpc.Do(ctx, http.MethodPost, apiURL, operationRequest)
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

// GetJourneyBySN 根据流程编号查询流程记录
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - sn: 流程编号
//
// 返回:
//   - *core.Journey: 流程记录信息
//   - error: 错误信息（如果不存在返回 core.ErrJourneyNotFound）
func (f *skylarkFlowRegistry) GetJourneyBySN(
	ctx context.Context,
	tenantID string,
	flowID int64,
	sn string,
) (*core.Journey, error) {
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

	// 3. 构建 API URL: /api/v4/yaw/flows/:flow_id/journeys/find_by_sn?sn=xxx
	apiURL := core.BuildFlowAPIURL(skylarkAddress, flowID, "journeys", "find_by_sn")
	apiURL = fmt.Sprintf("%s?sn=%s", apiURL, sn)

	// 4. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 5. 解析响应
	var journeyResp JourneyResponse
	if err := httputils.ReadJSONResponse(resp, &journeyResp); err != nil {
		// 如果是 404 错误，转换为 ErrJourneyNotFound
		if errors.Is(err, core.ErrSkylarkAPINotFound) {
			return nil, core.ErrJourneyNotFound
		}
		return nil, err
	}

	// 6. 转换为领域模型（包含用户ID转换）
	journey, err := convertJourneyUserID(ctx, &journeyResp, f.fillLocalUserIDMap, tenantID)
	if err != nil {
		return nil, err
	}

	return journey, nil
}

// GetJourneyAssignments 获取流程节点处理信息列表
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - journeyID: 流程记录ID
//
// 返回:
//   - []*core.Assignment: 任务列表
//   - error: 错误信息
func (f *skylarkFlowRegistry) GetJourneyAssignments(
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

// GetJourneyDetail 获取流程记录详情
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - journeyID: 流程记录ID
//
// 返回:
//   - *core.JourneyDetail: 流程记录详情（包含字段值和附件）
//   - error: 错误信息（如果不存在返回 core.ErrJourneyNotFound）
func (f *skylarkFlowRegistry) GetJourneyDetail(
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

	// 6. 提取远程用户ID（发起人）
	userIDMapping := map[int]string{
		int(journeyDetailResp.User.ID): "",
	}

	// 7. 批量转换（远程ID → 本地ID），填充映射
	if err := f.fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
		return nil, fmt.Errorf("批量转换用户ID失败: %w", err)
	}

	// 8. 使用映射转换为领域模型
	journeyDetail := journeyDetailResp.ToDomain(userIDMapping)

	return journeyDetail, nil
}

// GetFlowDetail 获取流程详情
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//
// 返回:
//   - *core.FlowDetail: 流程详情（包含字段、节点、边信息）
//   - error: 错误信息（如果不存在返回 core.ErrFlowNotFound）
func (f *skylarkFlowRegistry) GetFlowDetail(
	ctx context.Context,
	tenantID string,
	flowID int64,
) (*core.FlowDetail, error) {
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

	// 3. 构建 API URL: /api/v4/yaw/flows/:flow_id
	apiURL := core.BuildFlowAPIURL(skylarkAddress, flowID)

	// 4. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 5. 解析响应
	var flowDetailResp FlowDetailResponse
	if err := httputils.ReadJSONResponse(resp, &flowDetailResp); err != nil {
		// 如果是 404 错误，转换为 ErrFlowNotFound
		if errors.Is(err, core.ErrSkylarkAPINotFound) {
			return nil, core.ErrFlowNotFound
		}
		return nil, err
	}

	// 6. 转换为领域模型并返回
	return flowDetailResp.ToDomain(), nil
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

// GetJourneyMoments 获取流程审批历史
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - journeyID: 流程记录ID
//
// 返回:
//   - []*core.Moment: 审批历史列表
//   - error: 错误信息
func (f *skylarkFlowRegistry) GetJourneyMoments(
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
		return int(mr.OperatorID)
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

// GetCurrentProcessingUsers 获取当前流程任务的处理者
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户ID（用于获取平台配置）
//   - flowID: 流程ID
//   - journeyID: 流程记录ID
//
// 返回:
//   - []*core.ProcessingUser: 当前处理人列表
//   - error: 错误信息
func (f *skylarkFlowRegistry) GetCurrentProcessingUsers(
	ctx context.Context,
	tenantID string,
	flowID int64,
	journeyID int64,
) ([]*core.ProcessingUser, error) {
	// 1. 参数校验
	if flowID <= 0 {
		return nil, fmt.Errorf("flowID 必须大于 0")
	}
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

	// 4. 构建 API URL: GET /api/v4/yaw/flows/:flow_id/journeys/:id/current_processing_users
	apiURL := core.BuildCurrentProcessingUsersURL(skylarkAddress, flowID, journeyID)

	// 5. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: skylarkAddress.AuthHeader})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 6. 处理 404 错误（流程或流程记录不存在）
	if resp.StatusCode == http.StatusNotFound {
		return nil, core.ErrJourneyNotFound
	}

	// 7. 解析响应
	var userResponses []ProcessingUserResponse
	if err := httputils.ReadJSONResponse(resp, &userResponses); err != nil {
		return nil, err
	}

	// 8. 提取所有唯一的远程用户ID（使用通用函数）
	userIDMapping := core.ExtractUserIDsToMap(userResponses, func(ur ProcessingUserResponse) int {
		return int(ur.ID)
	})

	// 9. 批量转换（远程ID → 本地ID），填充映射
	if len(userIDMapping) > 0 {
		if err := f.fillLocalUserIDMap(ctx, tenantID, &userIDMapping); err != nil {
			return nil, fmt.Errorf("批量转换用户ID失败: %w", err)
		}
	}

	// 10. 使用映射转换为领域模型
	users := make([]*core.ProcessingUser, len(userResponses))
	for i, ur := range userResponses {
		users[i] = ur.ToDomain(userIDMapping)
	}

	// 10. 返回结果
	return users, nil
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

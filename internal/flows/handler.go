package flows

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/rezeropoint/go-skylark/core"
	"github.com/rezeropoint/go-skylark/internal/httputils"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// skylarkFlowRegistry 流程注册表结构
type skylarkFlowRegistry struct {
	cache             core.CacheInterface
	getPlatformConfig core.GetPlatformConfigFunc // 获取平台配置的函数（依赖注入）
	getRemoteUserIDs  core.GetRemoteUserIDsFunc  // 获取远程用户ID的函数（依赖注入）
}

// newSkylarkFlowRegistry 创建新的流程注册表
func newSkylarkFlowRegistry(config *Config, cache core.CacheInterface, getPlatformConfig core.GetPlatformConfigFunc, getRemoteUserIDs core.GetRemoteUserIDsFunc) (*skylarkFlowRegistry, error) {
	if config == nil {
		return nil, core.ErrConfigNil
	}

	if getPlatformConfig == nil {
		return nil, fmt.Errorf("getPlatformConfig 不能为 nil")
	}

	if getRemoteUserIDs == nil {
		return nil, fmt.Errorf("getRemoteUserIDs 不能为 nil")
	}

	return &skylarkFlowRegistry{
		cache:             cache,
		getPlatformConfig: getPlatformConfig,
		getRemoteUserIDs:  getRemoteUserIDs,
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
	options UpdateJourneyStatusOptions,
) error {
	// 1. 通过 PlatformManager 获取租户配置
	cfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("获取租户配置失败: %w", err)
	}

	// 2. 验证 APIBaseURL 和 APIToken 是否配置
	if cfg.APIBaseURL == nil || *cfg.APIBaseURL == "" {
		return fmt.Errorf("租户 %s 的 APIBaseURL 未配置", tenantID)
	}
	if cfg.APIToken == nil || *cfg.APIToken == "" {
		return fmt.Errorf("租户 %s 的 APIToken 未配置", tenantID)
	}

	// 3. 转换本地用户ID为远程用户ID
	remoteUserIDs, err := f.getRemoteUserIDs(ctx, tenantID, []string{localUserID})
	if err != nil {
		return fmt.Errorf("转换用户ID失败: %w", err)
	}
	if len(remoteUserIDs) == 0 {
		return fmt.Errorf("本地用户ID %s 未找到对应的远程用户ID", localUserID)
	}
	remoteUserID := remoteUserIDs[0]

	// 4. 从配置中构建 SkylarkAddress
	skylarkFlowAddress := core.SkylarkAPIContext{
		App:        *cfg.APIBaseURL,
		UserID:     strconv.Itoa(remoteUserID), // 操作人ID（远程）
		AuthHeader: *cfg.APIToken,
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
	operationRequest, err := f.buildOperationRequest(
		skylarkFlowAddress,
		string(operation), // 转换为字符串
		options.NextVertexID,
		options.Comment,
		options.CarbonCopyUserIDs,
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
	// 1. 通过 PlatformManager 获取租户配置
	cfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取租户配置失败: %w", err)
	}

	// 2. 验证 APIBaseURL 和 APIToken 是否配置
	if cfg.APIBaseURL == nil || *cfg.APIBaseURL == "" {
		return nil, fmt.Errorf("租户 %s 的 APIBaseURL 未配置", tenantID)
	}
	if cfg.APIToken == nil || *cfg.APIToken == "" {
		return nil, fmt.Errorf("租户 %s 的 APIToken 未配置", tenantID)
	}

	// 3. 从配置中构建 SkylarkAddress
	skylarkAddress := core.SkylarkAPIContext{
		App:        *cfg.APIBaseURL, // 使用配置中的域名
		UserID:     "",               // 查询操作不需要 UserID
		AuthHeader: *cfg.APIToken,   // 使用配置中的 Token
	}

	// 4. 构建 API URL: /api/v4/yaw/flows/:flow_id/journeys/find_by_sn
	apiURL := core.BuildFlowAPIURL(skylarkAddress, flowID, "journeys", "find_by_sn")

	// 5. 添加查询参数: ?sn=xxx
	apiURL = fmt.Sprintf("%s?sn=%s", apiURL, sn)

	// 6. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 7. 解析响应
	var journeyResp JourneyResponse
	if err := httputils.ReadJSONResponse(resp, &journeyResp); err != nil {
		// 如果是 404 错误，转换为 ErrJourneyNotFound
		if errors.Is(err, core.ErrSkylarkAPINotFound) {
			return nil, core.ErrJourneyNotFound
		}
		return nil, err
	}

	// 8. 转换为领域模型并返回
	return journeyResp.ToDomain(), nil
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
	// 1. 通过 PlatformManager 获取租户配置
	cfg, err := f.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("获取租户配置失败: %w", err)
	}

	// 2. 验证 APIBaseURL 和 APIToken 是否配置
	if cfg.APIBaseURL == nil || *cfg.APIBaseURL == "" {
		return nil, fmt.Errorf("租户 %s 的 APIBaseURL 未配置", tenantID)
	}
	if cfg.APIToken == nil || *cfg.APIToken == "" {
		return nil, fmt.Errorf("租户 %s 的 APIToken 未配置", tenantID)
	}

	// 3. 从配置中构建 SkylarkAddress
	skylarkAddress := core.SkylarkAPIContext{
		App:        *cfg.APIBaseURL, // 使用配置中的域名
		UserID:     "",               // 查询操作不需要 UserID
		AuthHeader: *cfg.APIToken,   // 使用配置中的 Token
	}

	// 4. 构建 API URL: /api/v4/yaw/journeys/:journey_id/assignments
	apiURL := core.BuildJourneyAPIURL(skylarkAddress, journeyID, "assignments")

	// 5. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 6. 解析响应
	var assignmentResponses []AssignmentResponse
	if err := httputils.ReadJSONResponse(resp, &assignmentResponses); err != nil {
		return nil, err
	}

	// 7. 转换为领域模型
	assignments := make([]*core.Assignment, len(assignmentResponses))
	for i, ar := range assignmentResponses {
		assignments[i] = ar.ToDomain()
	}

	return assignments, nil
}

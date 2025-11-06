package flows

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// skylarkFlowRegistry 流程注册表结构
type skylarkFlowRegistry struct {
	cache core.CacheInterface
}

// newSkylarkFlowRegistry 创建新的流程注册表
func newSkylarkFlowRegistry(config *Config, cache core.CacheInterface) (*skylarkFlowRegistry, error) {
	if config == nil {
		return nil, core.ErrConfigNil
	}

	return &skylarkFlowRegistry{
		cache: cache,
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
	skylarkFlowAddress := core.BasicSkylarkAddress{
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

	// 读取响应体
	routeFlowBody, err := io.ReadAll(routeFlowResult.Body)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrResponseBodyReadFailed, err)
	}

	if routeFlowResult.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: 状态码: %d，响应体: %s", core.ErrHTTPRequestFailed, routeFlowResult.StatusCode, routeFlowBody)
	}

	// 解析 Skylark 流程路由响应，获取下一个节点 ID
	flowProposeRequest, err := f.buildFlowProposeRequest(skylarkFlowAddress, routeFlowBody)
	if err != nil {
		return err
	}

	// 发送 Skylark 流程提议请求
	proposeFlowResult, err := httpc.Do(ctx, http.MethodPost, apiURL, flowProposeRequest)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer proposeFlowResult.Body.Close()

	// 读取响应体
	proposeFlowBody, err := io.ReadAll(proposeFlowResult.Body)
	if err != nil {
		return fmt.Errorf("%w: %v", core.ErrResponseBodyReadFailed, err)
	}

	if proposeFlowResult.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: 状态码: %d，响应体: %s", core.ErrHTTPRequestFailed, proposeFlowResult.StatusCode, proposeFlowBody)
	}

	return nil
}

// UpdateJourneyStatus 更新流程任务状态
// 参数:
//   - ctx: 上下文
//   - app: 应用名称
//   - flowID: 流程ID（用于获取字段映射）
//   - journeyID: 流程记录ID
//   - assignmentID: 任务ID
//   - userID: 用户ID
//   - authHeader: 认证头信息
//   - operation: 操作类型 (approve/refuse/transfer/cancel)
//   - options: 可选参数
func (f *skylarkFlowRegistry) UpdateJourneyStatus(
	ctx context.Context,
	app string,
	flowID int64,
	journeyID int64,
	assignmentID int64,
	userID int64,
	authHeader string,
	operation string,
	options UpdateJourneyStatusOptions,
) error {
	// 构建流程地址信息
	skylarkFlowAddress := core.BasicSkylarkAddress{
		App:        app,
		UserID:     strconv.FormatInt(userID, 10),
		AuthHeader: authHeader,
	}

	// 构建API请求URL
	apiURL := core.BuildJourneyAssignmentAPIURL(skylarkFlowAddress, journeyID, assignmentID)

	// 准备工作完成，现在获取分布式锁
	lockKey := fmt.Sprintf("journey:lock:%s:%d:%d:%d", app, journeyID, assignmentID, userID)
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

	// 读取第一次请求的响应体
	routeBody, err := io.ReadAll(routeResult.Body)
	if err != nil {
		return fmt.Errorf("%w: 读取第一次请求响应失败: %v", core.ErrResponseBodyReadFailed, err)
	}

	if routeResult.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: 第一次请求失败，状态码: %d，响应体: %s", core.ErrHTTPRequestFailed, routeResult.StatusCode, routeBody)
	}

	// 第二次请求：执行操作（approve/refuse/transfer/cancel）
	operationRequest, err := f.buildOperationRequest(
		skylarkFlowAddress,
		operation,
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

	// 读取第二次请求的响应体
	operationBody, err := io.ReadAll(operationResult.Body)
	if err != nil {
		return fmt.Errorf("%w: 读取第二次请求响应失败: %v", core.ErrResponseBodyReadFailed, err)
	}

	if operationResult.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: 第二次请求失败，状态码: %d，响应体: %s", core.ErrHTTPRequestFailed, operationResult.StatusCode, operationBody)
	}

	return nil
}

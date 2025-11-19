// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🔵 API 请求
// 说明：本文件定义 Skylark REST API 的地址值对象和 URL 构建器
package core

import (
	"strconv"
	"strings"
)

// SkylarkAPIContext Skylark API 调用上下文（DDD值对象）
// 职责：封装 Skylark REST API 调用所需的认证信息和基础参数
// 使用：配合包级 URL 构建函数（BuildFlowAPIURL、BuildFormAPIURL 等）构建完整 API 请求
type SkylarkAPIContext struct {
	App        string // 应用域名（如 "app.skylark.com"）
	UserID     string // 用户ID（调用者身份，查询操作可为空）
	AuthHeader string // 认证令牌（用于 HTTP Authorization 头）
}

// BuildAPIURL 构建通用 API URL
// 说明：组合协议 + 应用域名 + 路径 + 参数
// 参数:
//   - app: 应用名称
//   - path: API路径
//   - pathParams: 路径参数
func BuildAPIURL(app string, path string, pathParams ...string) string {
	var fullPath strings.Builder
	fullPath.WriteString(SchemeHTTPS)
	fullPath.WriteString(app)
	fullPath.WriteString(path)

	for _, param := range pathParams {
		if param != "" {
			if !strings.HasSuffix(fullPath.String(), "/") {
				fullPath.WriteString("/")
			}
			fullPath.WriteString(param)
		}
	}

	return fullPath.String()
}

// BuildFlowAPIURL 构建流程API URL
// 示例：https://app.skylark.com/flows/123/journeys
// 参数:
//   - apiCtx: API 调用上下文
//   - flowID: 流程ID
//   - additionalPath: 额外路径（如 "journeys"）
func BuildFlowAPIURL(apiCtx SkylarkAPIContext, flowID int64, additionalPath ...string) string {
	params := []string{strconv.FormatInt(flowID, 10)}
	params = append(params, additionalPath...)
	return BuildAPIURL(apiCtx.App, APIFlowsPath, params...)
}

// BuildFormAPIURL 构建表单API URL
// 示例：https://app.skylark.com/forms/456/rows
// 参数:
//   - apiCtx: API 调用上下文
//   - formID: 表单ID
//   - additionalPath: 额外路径（如 "rows"）
func BuildFormAPIURL(apiCtx SkylarkAPIContext, formID int64, additionalPath ...string) string {
	params := []string{strconv.FormatInt(formID, 10)}
	params = append(params, additionalPath...)
	return BuildAPIURL(apiCtx.App, APIFormsPath, params...)
}

// BuildJourneyAssignmentAPIURL 构建流程记录任务API URL
// 示例：https://app.skylark.com/journeys/789/assignments/101
// 参数:
//   - apiCtx: API 调用上下文
//   - journeyID: 流程记录ID
//   - assignmentID: 任务ID
func BuildJourneyAssignmentAPIURL(apiCtx SkylarkAPIContext, journeyID int64, assignmentID int64) string {
	params := []string{
		strconv.FormatInt(journeyID, 10),
		"assignments",
		strconv.FormatInt(assignmentID, 10),
	}
	return BuildAPIURL(apiCtx.App, APIJourneysPath, params...)
}

// BuildUserAssignmentsURL 构建用户任务列表 API URL
// 示例：https://app.skylark.com/api/v4/yaw/flows/user_assignments.json
// 参数:
//   - apiCtx: API 调用上下文
func BuildUserAssignmentsURL(apiCtx SkylarkAPIContext) string {
	return BuildAPIURL(apiCtx.App, APIFlowsPath, "user_assignments.json")
}

// BuildProposedJourneysURL 构建用户发起的流程列表 API URL
// 示例：https://app.skylark.com/api/v4/yaw/flows/123/journeys/proposed_journeys
// 参数:
//   - apiCtx: API 调用上下文
//   - flowID: 流程ID
func BuildProposedJourneysURL(apiCtx SkylarkAPIContext, flowID int64) string {
	return BuildFlowAPIURL(apiCtx, flowID, "journeys", "proposed_journeys")
}

// BuildJourneySearchURL 构建流程记录搜索 API URL
// 示例：https://app.skylark.com/api/v4/yaw/flows/123/journeys/search
// 参数:
//   - apiCtx: API 调用上下文
//   - flowID: 流程ID
func BuildJourneySearchURL(apiCtx SkylarkAPIContext, flowID int64) string {
	return BuildFlowAPIURL(apiCtx, flowID, "journeys", "search")
}

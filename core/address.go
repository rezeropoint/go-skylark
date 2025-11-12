// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🔵 API 请求
// 说明：本文件定义 Skylark REST API 的地址值对象和 URL 构建器
package core

import (
	"strconv"
	"strings"
)

// BasicSkylarkAddress Skylark API 认证上下文（DDD值对象）
// 职责：
// 1. 存储 Skylark REST API 认证信息
// 2. 提供 URL 构建能力（通过 BuildXxxAPIURL 函数）
type BasicSkylarkAddress struct {
	App        string // 应用域名（如 "app.skylark.com"）
	UserID     string // 用户ID（调用者身份）
	AuthHeader string // 认证头信息（用于 HTTP Authorization）
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
//   - address: 流程地址信息
//   - flowID: 流程ID
//   - additionalPath: 额外路径（如 "journeys"）
func BuildFlowAPIURL(address BasicSkylarkAddress, flowID int64, additionalPath ...string) string {
	params := []string{strconv.FormatInt(flowID, 10)}
	params = append(params, additionalPath...)
	return BuildAPIURL(address.App, APIFlowsPath, params...)
}

// BuildFormAPIURL 构建表单API URL
// 示例：https://app.skylark.com/forms/456/rows
// 参数:
//   - address: 表单地址信息
//   - formID: 表单ID
//   - additionalPath: 额外路径（如 "rows"）
func BuildFormAPIURL(address BasicSkylarkAddress, formID int64, additionalPath ...string) string {
	params := []string{strconv.FormatInt(formID, 10)}
	params = append(params, additionalPath...)
	return BuildAPIURL(address.App, APIFormsPath, params...)
}

// BuildJourneyAssignmentAPIURL 构建流程记录任务API URL
// 示例：https://app.skylark.com/journeys/789/assignments/101
// 参数:
//   - address: 流程地址信息
//   - journeyID: 流程记录ID
//   - assignmentID: 任务ID
func BuildJourneyAssignmentAPIURL(address BasicSkylarkAddress, journeyID int64, assignmentID int64) string {
	params := []string{
		strconv.FormatInt(journeyID, 10),
		"assignments",
		strconv.FormatInt(assignmentID, 10),
	}
	return BuildAPIURL(address.App, APIJourneysPath, params...)
}

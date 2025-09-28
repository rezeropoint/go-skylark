package core

import (
	"strconv"
	"strings"
)

// buildAPIURL 构建API URL
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
// 参数:
//   - address: 流程地址信息
//   - additionalPath: 额外路径
func BuildFlowAPIURL(address BasicSkylarkAddress, flowID int64, additionalPath ...string) string {
	params := []string{strconv.FormatInt(flowID, 10)}
	params = append(params, additionalPath...)
	return BuildAPIURL(address.App, APIFlowsPath, params...)
}

func BuildFormAPIURL(address BasicSkylarkAddress, formID int64, additionalPath ...string) string {
	params := []string{strconv.FormatInt(formID, 10)}
	params = append(params, additionalPath...)
	return BuildAPIURL(address.App, APIFormsPath, params...)
}

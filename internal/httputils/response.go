package httputils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ReadJSONResponse 读取HTTP响应体并解析为JSON
//
// 功能：
//  1. 先检查状态码，如果>=400则读取响应体并返回细粒度错误
//  2. 如果状态码正常且target不为nil，读取响应体并解析JSON
//
// 性能优化：
//   - 错误响应时只读取少量响应体用于错误提示
//   - 成功响应时才完整解析JSON
//
// 用途：
//   - POST/GET请求：传入target解析响应
//   - DELETE请求：传入nil只验证状态码
//
// 参数：
//   - resp: HTTP响应对象（调用方负责关闭Body）
//   - target: JSON解析目标对象，nil表示不需要解析响应体
//
// 返回：
//   - error: 如果状态码>=400返回细粒度错误，如果JSON解析失败返回解析错误
//
// 示例：
//
//	// 解析响应
//	var result UserResponse
//	if err := ReadJSONResponse(resp, &result); err != nil {
//	    return nil, err
//	}
//
//	// 不解析响应（DELETE等）
//	if err := ReadJSONResponse(resp, nil); err != nil {
//	    return err
//	}
func ReadJSONResponse(resp *http.Response, target any) error {
	// 1. 先检查状态码（避免不必要的读取）
	if resp.StatusCode >= 400 {
		// 只有错误时才读取响应体（用于错误详情）
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("读取错误响应失败: %w", err)
		}
		return ParseHTTPError(resp.StatusCode, body)
	}

	// 2. 状态码正常，如果需要解析响应体
	if target != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("读取响应失败: %w", err)
		}

		// 解析JSON响应
		if len(body) > 0 {
			if err := json.Unmarshal(body, target); err != nil {
				return fmt.Errorf("解析响应失败: %w, 响应体: %s", err, string(body))
			}
		}
	}

	return nil
}

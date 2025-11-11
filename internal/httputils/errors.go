package httputils

import (
	"fmt"
	"net/http"

	"github.com/rezeropoint/go-skylark/core"
)

// ParseHTTPError 解析HTTP错误状态码并返回细粒度错误
// 根据HTTP状态码返回对应的预定义错误（core包中定义）
//
// 支持的状态码映射：
//   - 401 Unauthorized    → core.ErrSkylarkAPIUnauthorized
//   - 404 Not Found       → core.ErrSkylarkAPINotFound
//   - 400 Bad Request     → core.ErrSkylarkAPIBadRequest
//   - 其他 (>=400)        → core.ErrSkylarkAPIServerError
//
// 参数：
//   - statusCode: HTTP状态码
//   - body: 响应体字节数组
//
// 返回：
//   - error: 包含状态码和响应体的细粒度错误
func ParseHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", core.ErrSkylarkAPIUnauthorized, string(body))
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", core.ErrSkylarkAPINotFound, string(body))
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", core.ErrSkylarkAPIBadRequest, string(body))
	default:
		return fmt.Errorf("%w: HTTP %d, %s", core.ErrSkylarkAPIServerError, statusCode, string(body))
	}
}

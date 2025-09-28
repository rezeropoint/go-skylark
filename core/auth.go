package core

// AuthHeader 用于HTTP请求的认证头
type AuthHeader struct {
	Token string `header:"Authorization"` // JWT token
}

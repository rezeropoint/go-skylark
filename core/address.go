// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🔵 API 请求
// 说明：本文件定义的类型用于调用 Skylark REST API（写操作：创建流程、创建表单、更新任务）
package core

// BasicSkylarkAddress Skylark API 认证上下文（DDD值对象）
// 说明：包含调用 Skylark REST API 所需的认证信息
type BasicSkylarkAddress struct {
	App        string // 应用域名（如 "app.skylark.com"）
	UserID     string // 用户ID（调用者身份）
	AuthHeader string // 认证头信息（用于 HTTP Authorization）
}

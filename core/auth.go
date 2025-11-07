// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🔵 API 请求
// 说明：本文件定义的类型用于调用 Skylark REST API（写操作：创建流程、创建表单、更新任务）
package core

// AuthHeader 用于HTTP请求的认证头
type AuthHeader struct {
	Token string `header:"Authorization"` // JWT token
}

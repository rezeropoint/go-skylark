// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🔵 API 请求
// 说明：本文件定义的类型用于调用 Skylark REST API（写操作：创建流程、创建表单、更新任务）
package core

type SkylarkAddress interface {
	GetApp() string
	GetUserID() string
	GetAuthHeader() string
}

// BasicSkylarkAddress 流程地址信息
type BasicSkylarkAddress struct {
	App        string // 应用名称
	UserID     string // 用户ID
	AuthHeader string // 认证头信息
}

func (s *BasicSkylarkAddress) GetApp() string        { return s.App }
func (s *BasicSkylarkAddress) GetUserID() string     { return s.UserID }
func (s *BasicSkylarkAddress) GetAuthHeader() string { return s.AuthHeader }

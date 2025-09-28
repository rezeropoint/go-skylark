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

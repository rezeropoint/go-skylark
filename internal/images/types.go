package images

// uploadResponse 表示上传响应
type uploadResponse struct {
	ID   int    `json:"id"`   // 上传文件ID
	Name string `json:"name"` // 上传文件名称
}

// tokenResponse 表示Token响应
type tokenResponse struct {
	UpToken string `json:"uptoken"` // 上传Token
}

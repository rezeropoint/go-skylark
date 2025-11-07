// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：常量定义
// 说明：统一管理所有常量定义，按用途分类
package core

// ================================================================
// 🔵 API 请求常量（调用 Skylark REST API 使用）
// ================================================================

// HTTP 协议常量
const (
	SchemeHTTPS = "https://" // HTTPS协议
)

// API 路径常量
const (
	APIFlowsPath       = "/api/v4/yaw/flows/"          // 流程API路径
	APIFormsPath       = "/api/v4/forms/"              // 表单API路径
	APIJourneysPath    = "/api/v4/yaw/journeys/"       // 流程记录API路径
	APIAttachmentsPath = "/api/v4/attachments/uptoken" // 附件API路径
	APIUploadPath      = "https://up.qbox.me/"         // 七牛云上传API路径
)

// 操作类型常量
const (
	OperationRoute    = "route"    // 路由操作（修改数据）
	OperationPropose  = "propose"  // 提议操作（创建流程）
	OperationApprove  = "approve"  // 通过操作
	OperationRefuse   = "refuse"   // 回退操作
	OperationTransfer = "transfer" // 转交操作
	OperationCancel   = "cancel"   // 撤销操作
)

// 事件类型常量
const (
	EventJourneyStatus = "JourneyStatusEvent" // 旅程状态事件
)

// 七牛云上传常量
const (
	QiniuXKeyValue        = "1593586993541" // 七牛云上传x:key字段的值
	DefaultUserIDForToken = "1"             // 获取上传令牌时使用的默认用户ID
)

// 字段后缀常量（用于识别图片字段）
const (
	SuffixImage                = "_Img"       // 图片字段后缀
	SuffixBase64Image          = "_Base64Img" // Base64图片字段后缀
	MinSuffixImageLength       = 4            // 图片后缀最小长度
	MinSuffixBase64ImageLength = 10           // Base64图片后缀最小长度
)

// 字段类型常量（Skylark API 返回的字段类型）
const (
	FieldTypeRadioButton         = "Field::RadioButton"         // 字段类型: 单选按钮
	FieldTypeCheckbox            = "Field::Checkbox"            // 字段类型: 复选框
	FieldTypeSelectField         = "Field::SelectField"         // 字段类型: 下拉选择
	FieldTypeMultipleSelectField = "Field::MultipleSelectField" // 字段类型: 多选下拉
)

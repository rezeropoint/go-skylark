package core

import "errors"

// 预定义错误
var (
	ErrConfigNil                            = errors.New("配置不能为空")
	ErrFieldMappingEmpty                    = errors.New("字段映射为空，已清除字段缓存，请再次尝试")
	ErrFieldMappingEmptyAndCacheClearFailed = errors.New("字段映射为空且缓存清除失败")
	ErrInvalidFieldValue                    = errors.New("字段值类型无效")
	ErrOptionNotFound                       = errors.New("选项值不存在")
	ErrNoNextVertices                       = errors.New("没有下一个节点")
	ErrUserIDConversionFailed               = errors.New("用户ID转换失败")
	ErrHTTPRequestFailed                    = errors.New("HTTP请求失败")
	ErrResponseBodyReadFailed               = errors.New("读取响应体失败")
	ErrJSONUnmarshalFailed                  = errors.New("JSON解析失败")
	ErrCacheOperationFailed                 = errors.New("缓存操作失败")
	ErrImageOperationFailed                 = errors.New("图片操作失败")
	ErrBase64ImageNotSupported              = errors.New("不支持Base64格式图片")
	ErrUnsupportedFieldType                 = errors.New("不支持的字段类型")
	ErrFlowLockAcquireFailed                = errors.New("获取流程锁失败")
	ErrFlowLockReleaseFailed                = errors.New("释放流程锁失败")
	ErrFlowAlreadyProcessing                = errors.New("流程正在处理中，请稍后再试")
)

// HTTP协议常量
const (
	SchemeHTTPS = "https://" // HTTPS协议
)

const (
	APIFlowsPath    = "/api/v4/yaw/flows/"    // 流程API路径
	APIFormsPath    = "/api/v4/forms/"        // 表单API路径
	APIJourneysPath = "/api/v4/yaw/journeys/" // 流程记录API路径
)

// 操作类型常量
const (
	OperationRoute    = "route"    // 路由操作
	OperationPropose  = "propose"  // 提议操作
	OperationApprove  = "approve"  // 通过操作
	OperationRefuse   = "refuse"   // 回退操作
	OperationTransfer = "transfer" // 转交操作
	OperationCancel   = "cancel"   // 撤销操作
)

// 事件类型常量
const (
	EventJourneyStatus = "JourneyStatusEvent" // 旅程状态事件
)

// 其他常量
const (
	QiniuXKeyValue        = "1593586993541" // 七牛云上传x:key字段的值
	DefaultUserIDForToken = "1"             // 获取上传令牌时使用的默认用户ID
)

const (
	APIUploadPath = "https://up.qbox.me/" // 七牛云上传API路径
)

// 字段后缀常量
const (
	SuffixImage                = "_Img"       // 图片字段后缀
	SuffixBase64Image          = "_Base64Img" // Base64图片字段后缀
	MinSuffixImageLength       = 4            // 图片后缀最小长度
	MinSuffixBase64ImageLength = 10           // Base64图片后缀最小长度
)

// 字段类型常量
const (
	FieldTypeRadioButton         = "Field::RadioButton"         // 字段类型: 单选按钮
	FieldTypeCheckbox            = "Field::Checkbox"            // 字段类型: 复选框
	FieldTypeSelectField         = "Field::SelectField"         // 字段类型: 下拉选择
	FieldTypeMultipleSelectField = "Field::MultipleSelectField" // 字段类型: 多选下拉
)

const (
	APIAttachmentsPath = "/api/v4/attachments/uptoken" // 附件API路径
)

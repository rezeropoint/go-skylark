// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：错误定义
// 说明：统一管理所有错误定义，按用途分类
package core

import "errors"

// API 请求错误（调用 Skylark REST API 时可能发生的错误）

var (
	// 配置错误
	ErrConfigNil  = errors.New("配置不能为空")
	ErrLocalDBNil = errors.New("本地数据库连接不能为空")

	// 字段映射错误
	ErrFieldMappingEmpty                    = errors.New("字段映射为空，已清除字段缓存，请再次尝试")
	ErrFieldMappingEmptyAndCacheClearFailed = errors.New("字段映射为空且缓存清除失败")
	ErrInvalidFieldValue                    = errors.New("字段值类型无效")
	ErrOptionNotFound                       = errors.New("选项值不存在")
	ErrUnsupportedFieldType                 = errors.New("不支持的字段类型")

	// HTTP 请求错误
	ErrHTTPRequestFailed      = errors.New("HTTP请求失败")
	ErrResponseBodyReadFailed = errors.New("读取响应体失败")
	ErrJSONUnmarshalFailed    = errors.New("JSON解析失败")
	ErrNoNextVertices         = errors.New("没有下一个节点")

	// 图片操作错误
	ErrImageOperationFailed    = errors.New("图片操作失败")
	ErrBase64ImageNotSupported = errors.New("不支持Base64格式图片")

	// 分布式锁错误
	ErrFlowLockAcquireFailed = errors.New("获取流程锁失败")
	ErrFlowLockReleaseFailed = errors.New("释放流程锁失败")
	ErrFlowAlreadyProcessing = errors.New("流程正在处理中，请稍后再试")

	// 其他错误
	ErrUserIDConversionFailed = errors.New("用户ID转换失败")
	ErrCacheOperationFailed   = errors.New("缓存操作失败")
)

// 数据库查询错误（查询 Skylark PostgreSQL 时可能发生的错误）

var (
	// 配置相关错误
	ErrPlatformConfigNotFound = errors.New("平台配置不存在")
	ErrInvalidPlatformConfig  = errors.New("平台配置无效")
	ErrEventConfigNotFound    = errors.New("事件配置不存在")
	ErrInvalidEventConfig     = errors.New("事件配置无效")

	// 数据库连接错误
	ErrDatabaseConnection  = errors.New("数据库连接失败")
	ErrRemoteTableNotFound = errors.New("远程表不存在")

	// 字段相关错误
	ErrInvalidFieldName = errors.New("字段名无效")
	ErrNoVisibleFields  = errors.New("没有可见字段")
	ErrInvalidFieldType = errors.New("字段类型无效")

	// 权限相关错误
	ErrNoOrgAccess           = errors.New("没有组织访问权限")
	ErrAccessDenied          = errors.New("访问被拒绝")
	ErrOrgFieldNotConfigured = errors.New("组织字段未配置")

	// 查询相关错误
	ErrInvalidQueryParam = errors.New("查询参数无效")
	ErrQueryTimeout      = errors.New("查询超时")
	ErrJourneyNotFound   = errors.New("Journey不存在")
	ErrInvalidPageParam  = errors.New("分页参数无效")

	// Flow 相关错误
	ErrFlowNotFound  = errors.New("流程不存在")
	ErrInvalidFlowID = errors.New("流程ID无效")

	// 组织映射相关错误（业务字段值映射，用于 Query/Stats）
	ErrOrgMappingNotFound = errors.New("组织映射不存在")
	ErrOrgMappingExists   = errors.New("组织映射已存在")
	ErrInvalidLocalOrgID  = errors.New("本地组织ID无效")
	ErrOrgMappingInUse    = errors.New("组织映射正在使用中")
	ErrLocalOrgIDInUse    = errors.New("本地组织ID已被映射")
)

// 组织管理错误（组织ID映射及 Skylark 组织 CRUD 操作）

var (
	// 组织相关错误
	ErrOrgNotFound                       = errors.New("组织在 Skylark 中不存在")
	ErrOrgCreateFailed                   = errors.New("在 Skylark 创建组织失败")
	ErrOrgDeleteFailed                   = errors.New("在 Skylark 删除组织失败")
	ErrOrgUpdateFailed                   = errors.New("在 Skylark 更新组织失败")
	ErrOrganizationUpdateFailed          = errors.New("更新组织失败")
	ErrOrganizationManagerUpdateFailed   = errors.New("更新组织管理员失败")
	ErrOrganizationManagerUpdateConflict = errors.New("组织管理员更新冲突，请重试")
	ErrParentOrgNotFound                 = errors.New("父组织尚未同步到 Skylark，请先同步或绑定父组织")
	ErrOrgIDMappingExists                = errors.New("组织ID映射已存在")

	// 组织成员相关错误
	ErrMemberNotFound     = errors.New("成员在组织中不存在")
	ErrMemberAddFailed    = errors.New("添加组织成员失败")
	ErrMemberRemoveFailed = errors.New("移除组织成员失败")
	ErrInvalidMemberID    = errors.New("成员ID无效")
	ErrGetMembersFailed   = errors.New("获取组织成员列表失败")

	// 组织管理员相关错误
	ErrAdministratorNotFound     = errors.New("管理员在组织中不存在")
	ErrAdministratorAddFailed    = errors.New("添加组织管理员失败")
	ErrAdministratorUpdateFailed = errors.New("更新组织管理员失败")
	ErrAdministratorDeleteFailed = errors.New("删除组织管理员失败")
	ErrGetAdministratorsFailed   = errors.New("获取组织管理员列表失败")
	ErrInvalidAccessID           = errors.New("访问ID无效")

	// 用户相关错误
	ErrUserNotFound        = errors.New("用户在 Skylark 中不存在")
	ErrUserMappingNotFound = errors.New("用户映射不存在")
	ErrUserMappingExists   = errors.New("用户映射已存在")
	ErrUserCreateFailed    = errors.New("在 Skylark 创建用户失败")
)

// Skylark API 错误（调用 Skylark REST API 时的 HTTP 错误）

var (
	ErrSkylarkAPIUnauthorized = errors.New("Skylark API 认证失败")
	ErrSkylarkAPINotFound     = errors.New("Skylark API 资源不存在")
	ErrSkylarkAPIBadRequest   = errors.New("Skylark API 请求参数错误")
	ErrSkylarkAPIServerError  = errors.New("Skylark API 服务器错误")
)

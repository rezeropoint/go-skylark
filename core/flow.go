// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 数据库查询
// 说明：本文件定义的类型用于查询 Skylark PostgreSQL 数据库（读操作：Flow 信息查询、字段元数据）
package core

// FlowInfo 远程流程信息（来自flows表）
type FlowInfo struct {
	ID          int    `db:"id"`           // 流程ID
	Title       string `db:"title"`        // 流程名称
	NamespaceID int    `db:"namespace_id"` // 命名空间ID
}

// FieldMetadata 远程表字段元数据（用于前端字段选择）
type FieldMetadata struct {
	FieldName string `db:"column_name"` // 字段名
	DataType  string `db:"data_type"`   // 数据类型（PostgreSQL类型）
	IsSystem  bool   `db:"-"`           // 是否为系统字段（slp_前缀），通过IsSystemField()计算，不从数据库扫描
}

// IsSystemField 判断字段是否为系统字段
// 系统字段以 slp_ 前缀标识
func IsSystemField(fieldName string) bool {
	return len(fieldName) >= 4 && fieldName[:4] == "slp_"
}

// StatusTranslationMap 流程状态映射
var StatusTranslationMap = map[string]string{
	"stashed":     "编写中",
	"pending":     "待处理", // 虚拟状态：只有1个节点的未完成事件
	"processing":  "处理中", // 虚拟状态：有多个节点的未完成事件
	"approved":    "已通过",
	"refused":     "已回退",
	"transferred": "已转交",
	"skipped":     "已跳过",
	"cancelled":   "已撤销",
	"receding":    "回退中",
	"suspended":   "已暂停",
	"finished":    "已完成",
	"aborted":     "已终止",
}

// TranslateStatus 将流程状态翻译为中文，如果未知则返回原始状态
func TranslateStatus(status string) string {
	if translated, ok := StatusTranslationMap[status]; ok {
		return translated
	}
	return status
}

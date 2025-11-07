package query

// query 模块不需要 model.go
//
// 原因：
// - core/query.go 中的结构体（QueryRequest、QueryResponse 等）都是 API 请求/响应类型
// - 这些结构体不包含 db 标签，不需要 ORM 映射
// - query 模块直接使用动态 SQL 查询，返回 map[string]interface{} 类型
// - 不需要额外的数据库模型转换

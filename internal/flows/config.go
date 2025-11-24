package flows

// Config Flows Manager 配置
type Config struct {
	// 缓存配置

	// FlowInfoCacheTTL Flow信息缓存过期时间（秒）
	// 说明：用于 enrichment 性能优化时缓存 flow_id 和 flow_title
	// 默认值：3600（1小时）
	// 建议范围：1800-7200（30分钟到2小时）
	FlowInfoCacheTTL int

	// VertexInfoCacheTTL 节点信息缓存过期时间（秒）
	// 说明：用于缓存节点名称、类型等信息（用于 GetJourneyFullDetail）
	// 默认值：3600（1小时）
	// 建议范围：3600-86400（1小时到1天，节点信息变化频率极低）
	VertexInfoCacheTTL int
}

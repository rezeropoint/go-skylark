// Package core 提供 go-skylark SDK 的核心类型定义
//
// 本文件用途：🟢 流程完整详情
// 说明：本文件定义的类型用于表示流程记录的完整信息（一站式接口）
package core

// JourneyFullDetail 流程完整详情（领域模型）
// 说明：聚合流程记录的所有维度信息，减少前端调用次数
// 功能：
//   - 包含基础信息和业务数据（BasicInfo）
//   - 包含审批历史记录（History）
//   - 包含待处理节点信息（PendingNodes）
//   - 包含节点信息映射（Vertices）
//
// 用途：GetJourneyFullDetail 接口返回，前端一次调用即可获取完整信息
// 优势：
//   - 减少网络往返（4次调用 → 1次调用）
//   - 数据结构统一，前端无需自行合并
//   - 自动补充节点名称、处理人姓名
type JourneyFullDetail struct {
	// 基础信息（来自 GetJourneyDetail）
	// 包含流程编号、状态、发起人、业务数据等
	BasicInfo *JourneyDetail

	// 审批历史（来自 GetJourneyMoments）
	// 已处理节点的操作记录（按时间排序）
	History []*Moment

	// 待处理节点（来自 GetJourneyAssignments，经过筛选）
	// 当前流程中尚未处理的节点信息
	PendingNodes []*PendingNode

	// 节点信息映射（来自 GetFlowDetail）
	// 键：节点ID，值：节点详情
	// 用于快速查找节点名称、类型等信息
	Vertices map[int64]*FlowVertex
}

// PendingNode 待处理节点（领域模型）
// 说明：表示流程中待处理的节点信息
// 特点：
//   - 包含节点基础信息（ID、名称）
//   - 包含处理人列表（支持多人审批）
//   - 自动转换用户ID（远程ID → 本地ID）
//   - 自动补充用户姓名（通过用户映射查询）
//   - 包含节点字段列表（用于前端构造 UpdateJourneyStatus 的 Data 参数）
//
// 用途：在 JourneyFullDetail 中表示待处理节点
// 与 Moment 的区别：
//   - Moment 表示已处理的历史记录（有操作人、处理意见、处理时间）
//   - PendingNode 表示待处理节点（只有待处理人列表，无处理结果）
type PendingNode struct {
	// 节点信息
	VertexID   int64  // 节点ID
	VertexName string // 节点名称（自动补充）

	// 处理人信息
	// 说明：支持多人审批场景（如会签、或签）
	// 用途：调用方可根据本地用户ID自行查询用户信息
	AssigneeIDs []string // 待处理人ID列表（本地用户ID）

	// 时间信息
	CreatedAt string // 任务创建时间（ISO 8601格式）

	// 节点字段列表（自动补充）
	// 说明：当前节点要求填写的字段信息（从 Skylark API /vertices/:id 获取）
	// 用途：前端根据字段元数据动态渲染表单，构造 UpdateJourneyStatus 的 Data 参数
	// 容错：查询失败时为 nil，前端需检查并使用默认行为
	Fields []*VertexField
}

// VertexField 节点字段（领域模型）
// 说明：表示节点中的单个字段信息（用于前端动态渲染表单）
// 特点：
//   - 包含字段基础信息（ID、标识键、标题、类型）
//   - 包含字段权限信息（是否必填、是否可编辑）
//   - 包含选项列表（适用于选项类型字段）
//
// 用途：在 PendingNode 中描述节点的字段要求
// 使用场景：前端根据 Type/Options/Required 动态渲染表单控件
type VertexField struct {
	// 基础信息
	ID          int64  // 字段ID
	IdentityKey string // 字段唯一标识键（用于 UpdateJourneyStatus 的 Data 参数）
	Title       string // 字段标题（展示名称）
	Type        string // 字段类型（如 Field::RadioButton, Field::TextField）

	// 权限信息
	Required bool // 是否必填（前端校验）
	Editable bool // 是否可编辑（前端控制）

	// 选项列表（仅选项类型字段有值）
	// 说明：适用于 RadioButton、Checkbox、SelectField、MultipleSelectField
	// 用途：前端根据 Options 渲染下拉框/单选框等
	Options []FieldOption // 字段可选项（空数组表示非选项字段）
}

package query

import (
	"database/sql"
	"time"
)

// assignmentRow 用于扫描 Assignment 查询结果（数据库模型）
// 对应 Skylark 远程表：assignments_{flow_id}
type assignmentRow struct {
	AssignmentID int            `db:"slp_assignment_id"` // Assignment ID
	JourneyID    int            `db:"slp_journey_id"`    // Journey ID（流程实例）
	Status       string         `db:"slp_status"`        // 状态
	VertexID     int            `db:"slp_vertex_id"`     // 节点ID
	VertexName   sql.NullString `db:"vertex_name"`       // 节点名称（JOIN vertices表）
	VertexAlias  sql.NullString `db:"vertex_alias"`      // 节点别名（JOIN vertices表）
	UserID       sql.NullString `db:"slp_user_id"`       // 处理人ID
	CreatedAt    time.Time      `db:"slp_created_at"`    // 创建时间
	UpdatedAt    time.Time      `db:"slp_updated_at"`    // 更新时间
	BusinessData string         `db:"business_data"`     // 业务数据（JSON字符串）
}

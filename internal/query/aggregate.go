package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// assignmentRowForQuery 用于查询事件列表的 assignment 行（包含完整业务数据）
type assignmentRowForQuery struct {
	JourneyID    int    `db:"slp_journey_id"`
	AssignmentID int    `db:"slp_assignment_id"`
	Status       string `db:"slp_status"`
	VertexID     int    `db:"slp_vertex_id"`
	VertexName   string `db:"vertex_name"`
	CreatedAt    string `db:"slp_created_at"` // 时间字符串
	BusinessData string `db:"business_data"`  // JSON 字符串
}

// executeQueryAndAggregate 执行查询并聚合业务数据
// 逻辑：查询所有 assignment → 按 journey_id 分组 → 合并业务数据（后面的非空值覆盖前面的）
func (m *queryManager) executeQueryAndAggregate(ctx context.Context, remoteDB sqlx.SqlConn, querySQL string, queryArgs []interface{}, visibleFields []*core.FieldConfig) ([]map[string]interface{}, error) {
	// 1. 执行查询获取所有 assignment
	var rows []*assignmentRowForQuery
	err := remoteDB.QueryRowsCtx(ctx, &rows, querySQL, queryArgs...)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}

	if len(rows) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 2. 按 journey_id 分组
	journeyMap := make(map[int][]*assignmentRowForQuery)
	journeyOrder := []int{} // 保持顺序（SQL已排序）
	for _, row := range rows {
		if _, exists := journeyMap[row.JourneyID]; !exists {
			journeyOrder = append(journeyOrder, row.JourneyID)
		}
		journeyMap[row.JourneyID] = append(journeyMap[row.JourneyID], row)
	}

	// 3. 为每个 journey 聚合数据
	records := make([]map[string]interface{}, 0, len(journeyOrder))
	for _, journeyID := range journeyOrder {
		assignments := journeyMap[journeyID]
		aggregated := m.aggregateJourneyData(assignments, visibleFields)
		if aggregated != nil {
			records = append(records, aggregated)
		}
	}

	logx.WithContext(ctx).Infof("[executeQueryAndAggregate] 聚合完成: %d个journey, %d个assignment", len(records), len(rows))

	return records, nil
}

// aggregateJourneyData 聚合单个 journey 的所有 assignment 数据
// 逻辑：遍历所有 assignment（按时间顺序），合并业务数据，后面的非空值覆盖前面的
func (m *queryManager) aggregateJourneyData(assignments []*assignmentRowForQuery, visibleFields []*core.FieldConfig) map[string]interface{} {
	if len(assignments) == 0 {
		return nil
	}

	// 获取最新的 assignment（用于系统字段）
	latest := assignments[len(assignments)-1]

	// 合并所有 assignment 的业务数据
	allBusinessData := make(map[string]interface{})
	for _, assignment := range assignments {
		if assignment.BusinessData != "" {
			var tempData map[string]interface{}
			if err := json.Unmarshal([]byte(assignment.BusinessData), &tempData); err != nil {
				logx.Error("解析业务数据JSON失败:", err)
				continue
			}

			// 合并数据：后面的非空值覆盖前面的（类似 GetEventDetail）
			for key, value := range tempData {
				if value != nil {
					allBusinessData[key] = value
				}
			}
		}
	}

	// 构建最终记录（系统字段 + 可见业务字段）
	record := make(map[string]interface{})

	// 系统字段（来自最新的 assignment）
	record["slp_journey_id"] = latest.JourneyID
	record["slp_status"] = core.TranslateStatus(latest.Status) // 翻译状态
	record["vertex_name"] = latest.VertexName

	// 只保留配置的可见业务字段
	for _, field := range visibleFields {
		if field.IsVisible {
			if value, exists := allBusinessData[field.FieldName]; exists {
				record[field.FieldName] = value
			} else {
				record[field.FieldName] = nil // 字段不存在时返回 null
			}
		}
	}

	return record
}

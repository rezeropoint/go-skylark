package attachment

import (
	"strings"

	"github.com/rezeropoint/go-skylark/v2/core"
)

// isSkylarkAttachment 判断值是否为 Skylark 附件
// 返回：
//   - bool: 是否为附件
//   - []core.AttachmentValue: 解析出的附件列表
func isSkylarkAttachment(v interface{}) (bool, []core.AttachmentValue) {
	switch val := v.(type) {
	case []interface{}:
		if len(val) == 0 {
			return false, nil
		}

		attachments := make([]core.AttachmentValue, 0, len(val))
		for _, item := range val {
			if att, ok := parseAttachmentItem(item); ok {
				attachments = append(attachments, att)
			}
		}

		if len(attachments) > 0 {
			return true, attachments
		}
		return false, nil

	case map[string]interface{}:
		if att, ok := parseAttachmentItem(val); ok {
			return true, []core.AttachmentValue{att}
		}
		return false, nil

	default:
		return false, nil
	}
}

// parseAttachmentItem 解析单个附件对象
// 参数：
//   - item: 可能是附件的对象
//
// 返回：
//   - core.AttachmentValue: 解析出的附件值
//   - bool: 是否解析成功
func parseAttachmentItem(item interface{}) (core.AttachmentValue, bool) {
	m, ok := item.(map[string]interface{})
	if !ok {
		return core.AttachmentValue{}, false
	}

	// 检查 gid 字段是否符合 Skylark 附件格式
	gid, ok := m["gid"].(string)
	if !ok || !strings.HasPrefix(gid, core.SkylarkAttachmentGIDPrefix) {
		return core.AttachmentValue{}, false
	}

	// 解析 id 字段（支持多种数字类型）
	var id int64
	switch v := m["id"].(type) {
	case float64:
		id = int64(v)
	case int64:
		id = v
	case int:
		id = int64(v)
	default:
		return core.AttachmentValue{}, false
	}

	// 解析 value 字段（文件名，可选）
	value, _ := m["value"].(string)

	return core.AttachmentValue{
		GID:   gid,
		ID:    id,
		Value: value,
	}, true
}

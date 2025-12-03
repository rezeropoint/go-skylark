package attachment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/rezeropoint/go-skylark/v2/core"
	"github.com/rezeropoint/go-skylark/v2/internal/httputils"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpc"
	"golang.org/x/sync/errgroup"
)

// attachmentManager 附件管理器实现
type attachmentManager struct {
	getPlatformConfig core.GetPlatformConfigFunc
}

// GetBase64 获取单个附件的 Base64 内容
func (m *attachmentManager) GetBase64(ctx context.Context, tenantID string, attachmentID int64) (string, error) {
	// 1. 获取平台 API 配置
	apiCfg, err := m.getPlatformConfig(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("获取平台配置失败: %w", err)
	}

	// 2. 构建 API URL: GET /api/v4/attachments/:id/base64_file
	apiURL := core.BuildAPIURL(apiCfg.App, core.APIAttachmentBase64Path,
		strconv.FormatInt(attachmentID, 10), "base64_file")

	// 3. 发送 HTTP GET 请求
	resp, err := httpc.Do(ctx, http.MethodGet, apiURL, core.AuthHeader{Token: apiCfg.Token})
	if err != nil {
		return "", fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	// 4. 检查状态码
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", httputils.ParseHTTPError(resp.StatusCode, body)
	}

	// 5. 读取响应体（Base64 字符串）
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", core.ErrResponseBodyReadFailed, err)
	}

	return string(body), nil
}

// ConvertBusinessData 转换业务数据中的附件为 Base64（并发获取）
func (m *attachmentManager) ConvertBusinessData(ctx context.Context, tenantID string, data map[string]interface{}) error {
	if data == nil {
		return nil
	}

	// 1. 收集所有附件任务
	type attachmentTask struct {
		fieldKey string
		index    int // 在该字段附件数组中的索引
		att      core.AttachmentValue
	}
	var tasks []attachmentTask
	fieldAttachments := make(map[string][]core.AttachmentValue)

	for key, value := range data {
		isAttachment, attachments := isSkylarkAttachment(value)
		if !isAttachment {
			continue
		}
		fieldAttachments[key] = attachments
		for i, att := range attachments {
			tasks = append(tasks, attachmentTask{fieldKey: key, index: i, att: att})
		}
	}

	if len(tasks) == 0 {
		return nil
	}

	// 2. 并发获取所有附件
	results := make([]string, len(tasks))
	errors := make([]error, len(tasks))
	var mu sync.Mutex

	g, gCtx := errgroup.WithContext(ctx)
	for i, task := range tasks {
		i, task := i, task // 捕获循环变量
		g.Go(func() error {
			base64, err := m.GetBase64(gCtx, tenantID, task.att.ID)
			mu.Lock()
			results[i] = base64
			errors[i] = err
			mu.Unlock()
			if err != nil {
				logx.WithContext(gCtx).WithFields(
					logx.Field("module", "attachment_manager"),
					logx.Field("attachment_id", task.att.ID),
					logx.Field("field", task.fieldKey),
					logx.Field("error", err.Error()),
				).Error("获取附件 Base64 失败")
			}
			return nil // 不中断其他任务
		})
	}
	g.Wait()

	// 3. 按字段回填结果
	taskIndex := 0
	for key, attachments := range fieldAttachments {
		base64Values := make([]string, 0, len(attachments))
		hasError := false

		for range attachments {
			if errors[taskIndex] != nil {
				hasError = true
			} else {
				base64Values = append(base64Values, results[taskIndex])
			}
			taskIndex++
		}

		// 设置结果
		if hasError {
			data[key] = nil
		} else if len(base64Values) == 1 {
			// 单个附件返回字符串
			data[key] = base64Values[0]
		} else {
			// 多个附件返回数组
			data[key] = base64Values
		}
	}

	return nil
}

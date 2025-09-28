package images

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/rezeropoint/go-skylark/core"

	"github.com/zeromicro/go-zero/rest/httpc"
)

// CreateImageEntryFromURL 创建图片条目（通过图片 URL）
// 先获取上传令牌，然后下载图片并上传
func CreateImageEntryFromURL(ctx context.Context, skylarkAddress core.BasicSkylarkAddress, imageURL string) (int, string, error) {
	upToken, err := getUPToken(ctx, skylarkAddress)
	if err != nil {
		return 0, "", err
	}

	// 下载图片数据
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return 0, "", fmt.Errorf("%w: 创建请求失败: %v, URL: %s", core.ErrImageOperationFailed, err, imageURL)
	}

	resp, err := httpc.DoRequest(req)
	if err != nil {
		return 0, "", fmt.Errorf("%w: 下载图片失败: %v, URL: %s", core.ErrImageOperationFailed, err, imageURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("%w: 下载图片请求失败，状态码: %d, URL: %s", core.ErrHTTPRequestFailed, resp.StatusCode, imageURL)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %v", core.ErrResponseBodyReadFailed, err)
	}

	return uploadImageData(ctx, skylarkAddress, upToken, imageData)
}

// CreateImageEntryFromBase64 创建图片条目（通过 base64 数据）
// 先获取上传令牌，然后解码 base64 并上传
func CreateImageEntryFromBase64(ctx context.Context, skylarkAddress core.BasicSkylarkAddress, base64Data string) (int, string, error) {
	upToken, err := getUPToken(ctx, skylarkAddress)
	if err != nil {
		return 0, "", err
	}

	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return 0, "", fmt.Errorf("Base64 解码失败: %w", err)
	}

	return uploadImageData(ctx, skylarkAddress, upToken, imageData)
}

// getUPToken 获取上传令牌
func getUPToken(ctx context.Context, skylarkAddress core.BasicSkylarkAddress) (string, error) {
	// 获取Token
	attachmentsURL := buildAttachmentsAPIURL(skylarkAddress)
	resp, err := httpc.Do(ctx, http.MethodGet, attachmentsURL, core.AuthHeader{Token: skylarkAddress.GetAuthHeader()})
	if err != nil {
		return "", fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	// 关闭资源，避免作用域连接池
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", core.ErrResponseBodyReadFailed, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: 状态码: %d，响应体: %s", core.ErrHTTPRequestFailed, resp.StatusCode, string(body))
	}

	var response tokenResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("%w: %v", core.ErrJSONUnmarshalFailed, err)
	}

	return response.UpToken, nil
}

// uploadImageData 抽取上传二进制图片的公共逻辑
func uploadImageData(ctx context.Context, skylarkAddress core.BasicSkylarkAddress, upToken string, imageData []byte) (int, string, error) {
	// 创建一个新的buffer用于multipart/form-data请求
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// 写入token字段和x:key字段
	fields := map[string]string{
		"token": upToken,
		"x:key": core.QiniuXKeyValue,
	}

	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			return 0, "", fmt.Errorf("%w: 写入%s失败: %v", core.ErrImageOperationFailed, key, err)
		}
	}

	// 创建文件字段
	fileWriter, err := w.CreateFormFile("file", "image.jpeg")
	if err != nil {
		return 0, "", fmt.Errorf("%w: 创建文件字段失败: %v", core.ErrImageOperationFailed, err)
	}

	// 写入图片数据到文件字段
	if _, err := io.Copy(fileWriter, bytes.NewReader(imageData)); err != nil {
		return 0, "", fmt.Errorf("%w: 写入文件数据失败: %v", core.ErrImageOperationFailed, err)
	}

	// 关闭multipart writer
	if err := w.Close(); err != nil {
		return 0, "", fmt.Errorf("%w: 关闭multipart writer失败: %v", core.ErrImageOperationFailed, err)
	}

	// 构造HTTP请求
	req, err := http.NewRequest(http.MethodPost, core.APIUploadPath, &b)
	if err != nil {
		return 0, "", fmt.Errorf("%w: 创建HTTP请求失败: %v", core.ErrHTTPRequestFailed, err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", skylarkAddress.GetAuthHeader())

	// 使用httpc.DoRequest发送请求
	resp, err := httpc.DoRequest(req)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %v", core.ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("%w: 上传图片请求失败，状态码: %d", core.ErrHTTPRequestFailed, resp.StatusCode)
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %v", core.ErrResponseBodyReadFailed, err)
	}

	// 解析JSON响应
	var response uploadResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, "", fmt.Errorf("%w: %v", core.ErrJSONUnmarshalFailed, err)
	}

	// 返回ID和Name
	return response.ID, response.Name, nil
}

// buildAttachmentsAPIURL 构建附件API URL
// 参数:
//   - address: 流程地址信息
func buildAttachmentsAPIURL(address core.BasicSkylarkAddress) string {
	return core.BuildAPIURL(address.App, core.APIAttachmentsPath) +
		fmt.Sprintf("?purpose=create_responses&user_id=%s", core.DefaultUserIDForToken)
}

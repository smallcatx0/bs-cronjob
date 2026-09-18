package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpPayload http 任务的 payload 定义
// 示例: {"method":"GET","url":"http://x/api","headers":{"Token":"t"},"body":"","expect_status":200}
type httpPayload struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers"`
	Body         string            `json:"body"`
	ExpectStatus int               `json:"expect_status"` // 0表示 2xx/3xx 即成功
}

// RunHTTP 定时请求 http 接口
func RunHTTP(ctx context.Context, payload string) (string, error) {
	var p httpPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return "", fmt.Errorf("payload解析失败: %w", err)
	}
	if p.URL == "" {
		return "", fmt.Errorf("url 不能为空")
	}
	if p.Method == "" {
		p.Method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(p.Method), p.URL, strings.NewReader(p.Body))
	if err != nil {
		return "", err
	}
	for k, v := range p.Headers {
		req.Header.Set(k, v)
	}
	if p.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	cli := &http.Client{}
	start := time.Now()
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	output := fmt.Sprintf("HTTP %s %s -> %d (%s)\n%s",
		p.Method, p.URL, resp.StatusCode, time.Since(start).Truncate(time.Millisecond), string(body))

	if p.ExpectStatus > 0 {
		if resp.StatusCode != p.ExpectStatus {
			return output, fmt.Errorf("期望状态码 %d, 实际 %d", p.ExpectStatus, resp.StatusCode)
		}
		return output, nil
	}
	if resp.StatusCode >= 400 {
		return output, fmt.Errorf("http 状态码 %d", resp.StatusCode)
	}
	return output, nil
}

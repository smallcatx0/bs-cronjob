package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// GoFunc go func 任务函数签名: 入参为 payload 中的 args 原始JSON,返回输出与错误
type GoFunc func(ctx context.Context, args json.RawMessage) (string, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]GoFunc{}
)

// Register 注册一个 go func 任务(在 main init 阶段调用)
// payload 示例: {"func":"check_v3_domain_list","args":{}}
func Register(name string, fn GoFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = fn
}

// RegisteredFuncs 已注册的 go func 名称(供前端下拉选择)
func RegisteredFuncs() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

type gofuncPayload struct {
	Func string          `json:"func"`
	Args json.RawMessage `json:"args"`
}

// RunGoFunc 执行已注册的 go 函数
func RunGoFunc(ctx context.Context, payload string) (string, error) {
	var p gofuncPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return "", fmt.Errorf("payload解析失败: %w", err)
	}
	registryMu.RLock()
	fn, ok := registry[p.Func]
	registryMu.RUnlock()
	if !ok {
		return "", fmt.Errorf("go func %q 未注册", p.Func)
	}
	return fn(ctx, p.Args)
}

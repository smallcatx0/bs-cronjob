package tasks

import (
	"context"
	"encoding/json"
	"time"
)

// init 注册示例 go func 任务
func init() {
	Register("demo.hello", func(ctx context.Context, args json.RawMessage) (string, error) {
		var p struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(args, &p)
		if p.Name == "" {
			p.Name = "world"
		}
		return "hello " + p.Name + " @ " + time.Now().Format(time.DateTime), nil
	})
}

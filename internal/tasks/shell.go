package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"cron-job/internal/conf"
)

// shellPayload shell 任务的 payload 定义
// 示例: {"cmd":"/data/scripts/backup.sh","args":["-full"],"dir":"/data"}
type shellPayload struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args"`
	Dir  string   `json:"dir"`
}

// shellWhitelist 白名单: 允许执行的 shell 脚本或可执行程序(基名或全路径)
func shellWhitelist() []string {
	return conf.AppConf.GetStringSlice("shell.whitelist")
}

func inWhitelist(cmd string) bool {
	if cmd == "" {
		return false
	}
	base := filepath.Base(cmd)
	for _, w := range shellWhitelist() {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}
		if w == cmd || w == base {
			return true
		}
	}
	return false
}

// RunShell 执行白名单内的 shell 脚本或可执行程序
func RunShell(ctx context.Context, payload string) (string, error) {
	var p shellPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return "", fmt.Errorf("payload解析失败: %w", err)
	}
	if p.Cmd == "" {
		return "", fmt.Errorf("cmd 不能为空")
	}
	if !inWhitelist(p.Cmd) {
		return "", fmt.Errorf("命令 %s 不在白名单内", p.Cmd)
	}

	cmd := exec.CommandContext(ctx, p.Cmd, p.Args...)
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}
	out, err := cmd.CombinedOutput()
	output := truncateOut(string(out))
	if err != nil {
		return output, fmt.Errorf("执行失败: %w, output: %s", err, output)
	}
	return output, nil
}

func truncateOut(s string) string {
	const max = 8192
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

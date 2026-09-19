package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cron-job/internal/conf"
	"cron-job/models/dao/rds"

	"github.com/spf13/viper"
)

// setupConf 为依赖配置的测试初始化一个干净的 viper 实例
func setupConf() {
	conf.AppConf = viper.New()
}

func TestOnceTaskID(t *testing.T) {
	if got, want := OnceTaskID(42), "job:once:42"; got != want {
		t.Fatalf("OnceTaskID(42) = %q, want %q", got, want)
	}
}

func TestNewTask(t *testing.T) {
	task := newTask(7, rds.TriggerCron)
	if task.Type() != TypeJobExec {
		t.Fatalf("task type = %q, want %q", task.Type(), TypeJobExec)
	}
	var p Payload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		t.Fatalf("unmarshal payload fail: %v", err)
	}
	if p.JobID != 7 || p.TriggerType != rds.TriggerCron {
		t.Fatalf("payload = %+v, want JobID=7 TriggerType=%q", p, rds.TriggerCron)
	}
}

func TestQueue(t *testing.T) {
	setupConf()
	if got := Queue(); got != "default" {
		t.Fatalf("Queue() default = %q, want %q", got, "default")
	}
	conf.AppConf.Set("asynq.queue", "myqueue")
	if got := Queue(); got != "myqueue" {
		t.Fatalf("Queue() = %q, want %q", got, "myqueue")
	}
}

func TestTruncateOut(t *testing.T) {
	short := "hello"
	if got := truncateOut(short); got != short {
		t.Fatalf("truncateOut(short) = %q, want %q", got, short)
	}
	long := strings.Repeat("a", 8193)
	got := truncateOut(long)
	if !strings.HasSuffix(got, "...(truncated)") {
		t.Fatalf("truncateOut(long) should end with truncated mark, got tail %q", got[len(got)-20:])
	}
	if len(got) != 8192+len("...(truncated)") {
		t.Fatalf("truncateOut(long) len = %d, want %d", len(got), 8192+len("...(truncated)"))
	}
}

func TestInWhitelist(t *testing.T) {
	setupConf()
	conf.AppConf.Set("shell.whitelist", []string{"/data/scripts/backup.sh", "echo"})

	cases := []struct {
		cmd  string
		want bool
	}{
		{"/data/scripts/backup.sh", true}, // 全路径命中
		{"backup.sh", false},              // 仅基名不命中(白名单项为全路径)
		{"echo", true},
		{"rm", false},         // 不在白名单
		{"/other/echo", true}, /* 输入基名 echo 命中白名单项 */
		{"", false},           // 空命令
	}
	for _, c := range cases {
		if got := inWhitelist(c.cmd); got != c.want {
			t.Errorf("inWhitelist(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

func TestRunHTTP_InvalidPayload(t *testing.T) {
	if _, err := RunHTTP(context.Background(), "not-a-json"); err == nil ||
		!strings.Contains(err.Error(), "payload解析失败") {
		t.Fatalf("expect payload parse error, got %v", err)
	}
}

func TestRunHTTP_EmptyURL(t *testing.T) {
	if _, err := RunHTTP(context.Background(), `{"method":"GET","url":""}`); err == nil ||
		!strings.Contains(err.Error(), "url 不能为空") {
		t.Fatalf("expect url empty error, got %v", err)
	}
}

func TestRunHTTP_SuccessAndHeaders(t *testing.T) {
	var gotHeader, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Token")
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}))
	defer srv.Close()

	payload := fmt.Sprintf(`{"method":"post","url":%q,"headers":{"Token":"abc"},"body":"hi","expect_status":200}`, srv.URL)
	out, err := RunHTTP(context.Background(), payload)
	if err != nil {
		t.Fatalf("RunHTTP success err = %v", err)
	}
	if !strings.Contains(out, "200") || !strings.Contains(out, "pong") {
		t.Fatalf("RunHTTP output unexpected: %q", out)
	}
	if gotHeader != "abc" {
		t.Fatalf("header Token = %q, want %q", gotHeader, "abc")
	}
	if gotBody != "hi" {
		t.Fatalf("body = %q, want %q", gotBody, "hi")
	}
}

func TestRunHTTP_DefaultMethodGet(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := fmt.Sprintf(`{"url":%q}`, srv.URL)
	if _, err := RunHTTP(context.Background(), payload); err != nil {
		t.Fatalf("RunHTTP err = %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("default method = %q, want %q", gotMethod, http.MethodGet)
	}
}

func TestRunHTTP_ExpectStatusMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := fmt.Sprintf(`{"url":%q,"expect_status":500}`, srv.URL)
	if _, err := RunHTTP(context.Background(), payload); err == nil ||
		!strings.Contains(err.Error(), "期望状态码") {
		t.Fatalf("expect status mismatch error, got %v", err)
	}
}

func TestRunHTTP_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	payload := fmt.Sprintf(`{"url":%q}`, srv.URL)
	if _, err := RunHTTP(context.Background(), payload); err == nil ||
		!strings.Contains(err.Error(), "http 状态码") {
		t.Fatalf("expect http status error, got %v", err)
	}
}

func TestRegisteredFuncs(t *testing.T) {
	names := RegisteredFuncs()
	found := false
	for _, n := range names {
		if n == "demo.hello" {
			found = true
		}
	}
	if !found {
		t.Fatalf("RegisteredFuncs() should contain demo.hello, got %v", names)
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("RegisteredFuncs() not sorted: %v", names)
		}
	}
}

func TestRunGoFunc(t *testing.T) {
	out, err := RunGoFunc(context.Background(), `{"func":"demo.hello","args":{"name":"kui"}}`)
	if err != nil {
		t.Fatalf("RunGoFunc err = %v", err)
	}
	if !strings.Contains(out, "hello kui") {
		t.Fatalf("RunGoFunc output = %q, want contains 'hello kui'", out)
	}

	// 无 name 时回退 world
	out, err = RunGoFunc(context.Background(), `{"func":"demo.hello"}`)
	if err != nil || !strings.Contains(out, "hello world") {
		t.Fatalf("RunGoFunc default = (%q, %v), want 'hello world'", out, err)
	}
}

func TestRunGoFunc_Unregistered(t *testing.T) {
	if _, err := RunGoFunc(context.Background(), `{"func":"not.exist"}`); err == nil ||
		!strings.Contains(err.Error(), "未注册") {
		t.Fatalf("expect unregistered error, got %v", err)
	}
}

func TestRunGoFunc_InvalidPayload(t *testing.T) {
	if _, err := RunGoFunc(context.Background(), "bad-json"); err == nil ||
		!strings.Contains(err.Error(), "payload解析失败") {
		t.Fatalf("expect payload parse error, got %v", err)
	}
}

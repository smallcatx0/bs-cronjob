package rds

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMaskAlarm(t *testing.T) {
	tests := []struct {
		name      string
		alarm     string
		wantEq    string // 非空则要求结果与之完全相等
		notIn     string // 结果不应包含(用于校验敏感信息已脱敏)
		wantInStr string // 结果应包含
	}{
		{name: "空", alarm: "", wantEq: ""},
		{name: "非法JSON兜底", alarm: "{bad", wantEq: "******"},
		{
			name:   "预定义机器人原样返回",
			alarm:  `{"type":"ding_alarm","name":"运维告警群"}`,
			wantEq: `{"type":"ding_alarm","name":"运维告警群"}`,
		},
		{
			name:      "自定义机器人脱敏token与secret",
			alarm:     `{"type":"ding_alarm","webhook":"https://oapi.dingtalk.com/robot/send?access_token=abc123secret","secret":"SECxyz"}`,
			notIn:     "abc123secret",
			wantInStr: "access_token=******",
		},
		{
			name:  "无access_token参数整体打码",
			alarm: `{"type":"ding_alarm","webhook":"https://example.com/hook","secret":"SECxyz"}`,
			notIn: "example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskAlarm(tt.alarm)
			if tt.wantEq != "" || tt.alarm == "" {
				if got != tt.wantEq {
					t.Fatalf("MaskAlarm(%q) = %q, want %q", tt.alarm, got, tt.wantEq)
				}
			}
			if tt.notIn != "" && strings.Contains(got, tt.notIn) {
				t.Fatalf("MaskAlarm(%q) = %q, should not contain %q", tt.alarm, got, tt.notIn)
			}
			if tt.wantInStr != "" && !strings.Contains(got, tt.wantInStr) {
				t.Fatalf("MaskAlarm(%q) = %q, should contain %q", tt.alarm, got, tt.wantInStr)
			}
		})
	}
}

func TestParseAlarm(t *testing.T) {
	if _, err := ParseAlarm(""); err == nil {
		t.Fatal("ParseAlarm(\"\") 应报错")
	}
	if _, err := ParseAlarm("{bad"); err == nil {
		t.Fatal("ParseAlarm 非法 JSON 应报错")
	}
	if _, err := ParseAlarm(`{"name":"x"}`); err == nil {
		t.Fatal("ParseAlarm 缺少 type 应报错")
	}
	c, err := ParseAlarm(`{"type":"ding_alarm","name":"运维告警群"}`)
	if err != nil {
		t.Fatalf("ParseAlarm 合法配置应成功, got err=%v", err)
	}
	if c.Type != AlarmTypeDing || c.Name != "运维告警群" {
		t.Fatalf("ParseAlarm 解析结果不符: %+v", c)
	}
}

func TestIsAlarmMasked(t *testing.T) {
	if IsAlarmMasked(nil) {
		t.Fatal("IsAlarmMasked(nil) 应为 false")
	}
	plain := &AlarmConf{Type: AlarmTypeDing, Webhook: "https://x?access_token=abc", Secret: "SEC"}
	if IsAlarmMasked(plain) {
		t.Fatal("未脱敏配置应为 false")
	}
	masked := &AlarmConf{Type: AlarmTypeDing, Webhook: "https://x?access_token=******", Secret: "******"}
	if !IsAlarmMasked(masked) {
		t.Fatal("脱敏配置应为 true")
	}
}

// TestMaskAlarmRoundTrip 校验脱敏后仍是合法 JSON 且敏感字段被隐藏
func TestMaskAlarmRoundTrip(t *testing.T) {
	raw := `{"type":"ding_alarm","webhook":"https://oapi.dingtalk.com/robot/send?access_token=TOPSECRET","secret":"SIGNSECRET"}`
	got := MaskAlarm(raw)
	var c AlarmConf
	if err := json.Unmarshal([]byte(got), &c); err != nil {
		t.Fatalf("脱敏结果应为合法 JSON, got %q err=%v", got, err)
	}
	if strings.Contains(c.Webhook, "TOPSECRET") || c.Secret != "******" {
		t.Fatalf("脱敏未生效: %+v", c)
	}
	if !IsAlarmMasked(&c) {
		t.Fatal("脱敏结果应被 IsAlarmMasked 识别")
	}
}

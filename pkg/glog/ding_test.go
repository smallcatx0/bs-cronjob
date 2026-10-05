package glog_test

import (
	"testing"

	"cron-job/pkg/glog"

	"github.com/stretchr/testify/assert"
)

var webHook = "https://oapi.dingtalk.com/robot/send?access_token=***"
var secret = "****"

func TestTextMsg(t *testing.T) {
	assert := assert.New(t)
	ala := glog.DingAlarmNew(webHook, secret)
	err := ala.Text("测试普通消息", "多行文本内容", "自定义消息体").AtPhones("18681636749").Send()
	ala.Text("消息粘滞").Send()
	assert.NoError(err)
}

func TestMDMsg(t *testing.T) {
	assert := assert.New(t)
	ala := glog.DingAlarmNew(webHook, secret)
	err := ala.Markdown("title", "### 三级标题", "> 引用", "内容").Send()
	assert.NoError(err)
}

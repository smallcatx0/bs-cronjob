package conf

import (
	"os"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

var AppConf *viper.Viper

var hostname string

// dingRobot 预定义告警机器人配置
type dingRobot struct {
	webhook string
	secret  string
}

var (
	dingRobotMu    sync.RWMutex
	dingRobots     = map[string]dingRobot{}
	dingRobotOrder = []string{}
)

// InitDingRobots 读取 ding_robots 配置缓存为包级变量(需在 InitAppConf 之后调用), 供 controller 下拉枚举与 worker 发送共用
func InitDingRobots() {
	if AppConf == nil {
		return
	}
	var list []struct {
		Name    string `mapstructure:"name"`
		Webhook string `mapstructure:"webhook"`
		Secret  string `mapstructure:"secret"`
	}
	// 用 UnmarshalKey 而非 Get: viper 对 YAML 嵌套列表返回 map[interface{}]interface{}, Get+断言 map[string]any 会失败导致缓存为空
	if err := AppConf.UnmarshalKey("ding_robots", &list); err != nil {
		return
	}
	robots := map[string]dingRobot{}
	order := []string{}
	for _, it := range list {
		name := strings.TrimSpace(it.Name)
		if name == "" {
			continue
		}
		if _, dup := robots[name]; dup {
			continue
		}
		robots[name] = dingRobot{webhook: it.Webhook, secret: it.Secret}
		order = append(order, name)
	}
	dingRobotMu.Lock()
	dingRobots = robots
	dingRobotOrder = order
	dingRobotMu.Unlock()
}

// DingRobotNames 预定义机器人名称列表(供前端下拉, 不暴露 webhook/secret)
func DingRobotNames() []string {
	dingRobotMu.RLock()
	defer dingRobotMu.RUnlock()
	names := make([]string, len(dingRobotOrder))
	copy(names, dingRobotOrder)
	return names
}

// HasDingRobot 判定名称是否为已配置的预定义机器人
func HasDingRobot(name string) bool {
	dingRobotMu.RLock()
	defer dingRobotMu.RUnlock()
	_, ok := dingRobots[name]
	return ok
}

// LookupDingRobot 按名称查询预定义机器人的 webhook/secret
func LookupDingRobot(name string) (webhook, secret string, ok bool) {
	dingRobotMu.RLock()
	defer dingRobotMu.RUnlock()
	r, ok := dingRobots[name]
	return r.webhook, r.secret, ok
}

func HostName() string {
	if hostname != "" {
		return hostname
	}
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		return "unknow"
	}
	return hostname
}

func InitAppConf(filepath *string) error {
	AppConf = viper.New()
	AppConf.SetConfigFile(*filepath)
	AppConf.SetConfigType("yaml")

	// 设置默认
	AppConf.SetDefault("base.env", "dev")
	AppConf.SetDefault("base.debug", true)
	AppConf.SetDefault("base.http_port", "80")
	AppConf.Set("flag_param.c", *filepath)

	err := AppConf.ReadInConfig()
	if err != nil {
		return err
	}
	return nil
}

func Env() string {
	return AppConf.GetString("env")
}

func IsDebug() bool {
	return AppConf.GetBool("debug")
}

func HttpPort() string {
	return AppConf.GetString("http_port")
}

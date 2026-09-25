package valid

import (
	"cron-job/middleware/resp"
)

// checkSpec 校验 cron 表达式, 与调度器保持一致: 仅标准 5 段(分 时 日 月 周)及 @every 等描述符
func checkSpec(spec string) error {
	if err := CheckCronExpr(spec); err != nil {
		return resp.ParamInValid(err.Error())
	}
	return nil
}

// TtlQuery tabledata_ttl 列表查询条件
type TtlQuery struct {
	UnKey     string `form:"unkey" binding:"omitempty,max=128"`
	DbName    string `form:"db_name" binding:"omitempty,max=64"`
	Tablename string `form:"table_name" binding:"omitempty,max=128"`
	Status    string `form:"status" binding:"omitempty,oneof=offline online"`
}

// TtlAdd 新增 TTL 策略配置(状态固定为 offline 不可传入, 上线需单独调用 toggle 接口)
type TtlAdd struct {
	UnKey      string `json:"unkey" binding:"required,max=128"`
	Dsn        string `json:"dsn" binding:"required"`
	DbName     string `json:"db_name" binding:"omitempty,max=64"`
	Tablename  string `json:"table_name" binding:"required,max=128"`
	ColumnName string `json:"column_name" binding:"required,max=64"`
	ColumnType string `json:"column_type" binding:"required,oneof=unix timestamp datetime"`
	FindWh     string `json:"find_wh" binding:"omitempty,max=255"`
	TtlValue   int64  `json:"ttl_value" binding:"required,min=1"`
	Limit      int64  `json:"limit" binding:"required,min=1"`
	Spec       string `json:"spec" binding:"required"`
	Desc       string `json:"desc" binding:"omitempty,max=255"`
}

func (p *TtlAdd) Valid() error {
	return checkSpec(p.Spec)
}

// TtlUpdate 更新 TTL 策略配置(unkey 为调度任务名组成部分, 不可更新; 仅 offline 状态可更新, 状态切换走 toggle)
type TtlUpdate struct {
	ID         int64  `json:"id" binding:"required"`
	Dsn        string `json:"dsn" binding:"omitempty"`
	DbName     string `json:"db_name" binding:"omitempty,max=64"`
	Tablename  string `json:"table_name" binding:"omitempty,max=128"`
	ColumnName string `json:"column_name" binding:"omitempty,max=64"`
	ColumnType string `json:"column_type" binding:"omitempty,oneof=unix timestamp datetime"`
	FindWh     string `json:"find_wh" binding:"omitempty,max=255"`
	TtlValue   int64  `json:"ttl_value" binding:"omitempty,min=1"`
	Limit      int64  `json:"limit" binding:"omitempty,min=1"`
	Desc       string `json:"desc" binding:"omitempty,max=255"`
}

// RetryQuery tabledata_retry 列表查询条件
type RetryQuery struct {
	Unkey     string `form:"unkey" binding:"omitempty,max=128"`
	DbName    string `form:"db_name" binding:"omitempty,max=64"`
	Tablename string `form:"table_name" binding:"omitempty,max=128"`
	Status    string `form:"status" binding:"omitempty,oneof=offline online"`
}

// RetryAdd 新增 Retry 策略配置(状态固定为 offline 不可传入, 上线需单独调用 toggle 接口)
type RetryAdd struct {
	Unkey      string `json:"unkey" binding:"required,max=128"`
	Dsn        string `json:"dsn" binding:"required"`
	DbName     string `json:"db_name" binding:"omitempty,max=64"`
	Tablename  string `json:"table_name" binding:"required,max=128"`
	ColumnName string `json:"column_name" binding:"required,max=64"`
	ColumnType string `json:"column_type" binding:"required,oneof=unix timestamp datetime"`
	FindWh     string `json:"find_wh" binding:"required"`
	SetFields  string `json:"set_fields" binding:"required"`
	Before     int64  `json:"before" binding:"required,min=1"`
	Duration   int64  `json:"duration" binding:"required,min=1"`
	Limit      int64  `json:"limit" binding:"required,min=1"`
	Spec       string `json:"spec" binding:"required"`
	Desc       string `json:"desc" binding:"omitempty,max=255"`
}

func (p *RetryAdd) Valid() error {
	return checkSpec(p.Spec)
}

// RetryUpdate 更新 Retry 策略配置(unkey 为调度任务名组成部分, 不可更新; 仅 offline 状态可更新, 状态切换走 toggle)
type RetryUpdate struct {
	ID         int64  `json:"id" binding:"required"`
	Dsn        string `json:"dsn" binding:"omitempty"`
	DbName     string `json:"db_name" binding:"omitempty,max=64"`
	Tablename  string `json:"table_name" binding:"omitempty,max=128"`
	ColumnName string `json:"column_name" binding:"omitempty,max=64"`
	ColumnType string `json:"column_type" binding:"omitempty,oneof=unix timestamp datetime"`
	FindWh     string `json:"find_wh" binding:"omitempty"`
	SetFields  string `json:"set_fields" binding:"omitempty"`
	Before     int64  `json:"before" binding:"omitempty,min=1"`
	Duration   int64  `json:"duration" binding:"omitempty,min=1"`
	Limit      int64  `json:"limit" binding:"omitempty,min=1"`
	Desc       string `json:"desc" binding:"omitempty,max=255"`
}

// StrategyLogQuery TTL/Retry 策略执行日志查询条件(start/end 按 started_at 过滤)
type StrategyLogQuery struct {
	Kind       string `form:"kind" binding:"omitempty,oneof=ttl retry"`
	StrategyID int64  `form:"strategy_id" binding:"omitempty,min=1"`
	Status     string `form:"status" binding:"omitempty,oneof=running success failed"`
	Start      string `form:"start" binding:"omitempty,datetime=2006-01-02 15:04:05"`
	End        string `form:"end" binding:"omitempty,datetime=2006-01-02 15:04:05"`
}

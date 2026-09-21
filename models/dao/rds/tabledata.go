package rds

import (
	"fmt"
	"time"

	"cron-job/models/dao"
	"cron-job/pkg/glog"
)

// 时间列类型(column_type)
const (
	ColumType_Unix      = "unix"      // 时间戳数据库中 整数保存时间
	ColumType_Timestamp = "timestamp" // 时间戳
	ColumType_Datetime  = "datetime"  // 日期时间
)

// 策略状态(status)
const (
	StrategyOffline = "offline" // 未调度(可编辑/删除)
	StrategyOnline  = "online"  // 已注册 asynq 调度(不可编辑/删除)
)

type TabledataRetry struct {
	ID         int64  `gorm:"primaryKey; column:id" json:"id"`
	Unkey      string `gorm:"column:unkey" json:"unkey"`                                    // 任务唯一名
	Dsn        string `gorm:"column:dsn" json:"dsn"`                                        // 数据库链接
	DbName     string `gorm:"column:db_name" json:"db_name"`                                // 数据库名
	Tablename  string `gorm:"column:table_name" json:"table_name"`                          // 表名
	ColumnName string `gorm:"column:column_name" json:"column_name"`                        // 依据字段名
	ColumnType string `gorm:"column:column_type" json:"column_type"`                        // 依据字段类型
	FindWh     string `gorm:"column:find_wh" json:"find_wh"`                                // 查找条件
	SetFields  string `gorm:"column:set_fields" json:"set_fields"`                          // 更新的字段
	Before     int64  `gorm:"column:before" json:"before"`                                  // 从当前时间之前多少秒
	Duration   int64  `gorm:"column:duration" json:"duration"`                              // 时间间隔
	Limit      int64  `gorm:"column:limit" json:"limit"`                                    // 一次执行条数
	Spec       string `gorm:"column:spec" json:"spec"`                                      // cron表达式
	Status     string `gorm:"size:16;not null;default:offline;column:status" json:"status"` // 状态 offline/online
	Desc       string `gorm:"column:desc" json:"desc"`                                      // 描述
}

func (*TabledataRetry) TableName() string {
	return "bs_tabledata_retry"
}

func (t *TabledataRetry) GetCfgs() ([]TabledataRetry, error) {
	cfgs := []TabledataRetry{}
	err := dao.MysqlCli.Find(&cfgs).Error
	return cfgs, err
}

// GetByID 按主键查询单条 Retry 策略配置, 供消费端执行时加载最新配置
func (t *TabledataRetry) GetByID(id int64) (*TabledataRetry, error) {
	var cfg TabledataRetry
	if err := dao.MysqlCli.Where("id = ?", id).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetStatus 更新 Retry 策略状态(offline/online)
func (t *TabledataRetry) SetStatus(id int64, status string) error {
	return dao.MysqlCli.Model(&TabledataRetry{}).Where("id = ?", id).
		Update("status", status).Error
}

// BuildRetrySqls 根据 Retry 配置生成 count 与 update 两条 SQL, 执行端与预览端共用,
// now 为基准时间由调用方传入以便单测; update 语句带 LIMIT limit 分批更新(与 TTL 分批删除对齐),
// column_type 非法或 limit 非正时返回 error
func BuildRetrySqls(tablename, columnName, columnType, findWh, setFields string, before, duration, limit int64, now time.Time) (countSql, updateSql string, err error) {
	if limit < 1 {
		return "", "", fmt.Errorf("limit:%d 非法, 单次执行条数必须 >= 1", limit)
	}
	st := now.Add(-time.Second * time.Duration(before))
	ed := st.Add(time.Second * time.Duration(duration))
	var where string
	switch columnType {
	case ColumType_Unix:
		where = fmt.Sprintf("`%s` >= %d AND `%s` < %d", columnName, st.Unix(), columnName, ed.Unix())
	case ColumType_Timestamp, ColumType_Datetime:
		where = fmt.Sprintf("`%s` >= '%s' AND `%s` < '%s'",
			columnName, st.Format("2006-01-02 15:04:05"), columnName, ed.Format("2006-01-02 15:04:05"))
	default:
		return "", "", fmt.Errorf("column_type:%s 不支持，可选：%s/%s/%s",
			columnType, ColumType_Unix, ColumType_Timestamp, ColumType_Datetime)
	}
	where += " AND " + findWh
	countSql = fmt.Sprintf("SELECT count(*) c FROM `%s` WHERE %s", tablename, where)
	updateSql = fmt.Sprintf("UPDATE `%s` SET %s WHERE %s LIMIT %d", tablename, setFields, where, limit)
	return countSql, updateSql, nil
}

// ParseSql 根据 Retry 策略配置生成实际执行的更新 SQL(含 LIMIT 分批), 供前端预览即将执行的语句;
// 生成逻辑收敛在 BuildRetrySqls, column_type 非法或 limit 非正时返回 error
func (t *TabledataRetry) ParseSql() (string, error) {
	_, updateSql, err := BuildRetrySqls(t.Tablename, t.ColumnName, t.ColumnType, t.FindWh, t.SetFields, t.Before, t.Duration, t.Limit, time.Now())
	return updateSql, err
}

type TabledataTtl struct {
	ID         int64  `gorm:"primaryKey; column:id" json:"id"`
	UnKey      string `gorm:"column:unkey" json:"unkey"`                                    // 策略唯一key
	Dsn        string `gorm:"column:dsn" json:"dsn"`                                        // 数据库链接
	DbName     string `gorm:"column:db_name" json:"db_name"`                                // 数据库名
	Tablename  string `gorm:"column:table_name" json:"table_name"`                          // 表名
	ColumnName string `gorm:"column:column_name" json:"column_name"`                        // 依据字段名
	ColumnType string `gorm:"column:column_type" json:"column_type"`                        // 依据字段类型
	TtlValue   int64  `gorm:"column:ttl_value" json:"ttl_value"`                            // TTL过期时间
	Limit      int64  `gorm:"column:limit" json:"limit"`                                    // 一次执行条数
	Spec       string `gorm:"column:spec" json:"spec"`                                      // cron表达式
	Status     string `gorm:"size:16;not null;default:offline;column:status" json:"status"` // 状态 offline/online
	Desc       string `gorm:"column:desc" json:"desc"`                                      // 描述
}

func (*TabledataTtl) TableName() string {
	return "bs_tabledata_ttl"
}

func (t *TabledataTtl) GetCfgs() ([]TabledataTtl, error) {
	ttls := []TabledataTtl{}
	err := dao.MysqlCli.Find(&ttls).Error
	return ttls, err
}

// GetByID 按主键查询单条 TTL 策略配置, 供消费端执行时加载最新配置
func (t *TabledataTtl) GetByID(id int64) (*TabledataTtl, error) {
	var cfg TabledataTtl
	if err := dao.MysqlCli.Where("id = ?", id).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SetStatus 更新 TTL 策略状态(offline/online)
func (t *TabledataTtl) SetStatus(id int64, status string) error {
	return dao.MysqlCli.Model(&TabledataTtl{}).Where("id = ?", id).
		Update("status", status).Error
}

// BuildTtlDeleteSql 根据 TTL 配置生成删除 SQL, 执行端与预览端共用,
// cutoff 为过期界限时间由调用方传入以便单测; column_type 非法时返回 error
func BuildTtlDeleteSql(tablename, columnName, columnType string, cutoff time.Time, limit int64) (string, error) {
	var where string
	switch columnType {
	case ColumType_Unix:
		where = fmt.Sprintf("`%s` < %d", columnName, cutoff.Unix())
	case ColumType_Timestamp, ColumType_Datetime:
		where = fmt.Sprintf("`%s` < '%s'", columnName, cutoff.Format("2006-01-02 15:04:05"))
	default:
		return "", fmt.Errorf("column_type:%s 不支持，可选：%s/%s/%s",
			columnType, ColumType_Unix, ColumType_Timestamp, ColumType_Datetime)
	}
	return fmt.Sprintf("DELETE FROM `%s` WHERE %s LIMIT %d", tablename, where, limit), nil
}

// ParseSql 根据 TTL 策略配置生成实际执行的删除 SQL, 供前端预览即将执行的语句;
// 生成逻辑收敛在 BuildTtlDeleteSql, column_type 非法时返回 error
func (t *TabledataTtl) ParseSql() (string, error) {
	cutoff := time.Now().Add(-time.Second * time.Duration(t.TtlValue))
	return BuildTtlDeleteSql(t.Tablename, t.ColumnName, t.ColumnType, cutoff, t.Limit)
}

// TabledataStrategyLog TTL/Retry 策略执行日志, 复用 JobLog 状态(running/success/failed)
type TabledataStrategyLog struct {
	ID           int64      `gorm:"primaryKey; column:id" json:"id"`
	Kind         string     `gorm:"size:16;index; column:kind" json:"kind"`              // ttl / retry
	StrategyID   int64      `gorm:"index; column:strategy_id" json:"strategy_id"`        // 策略配置行主键
	StrategyName string     `gorm:"size:128; column:strategy_name" json:"strategy_name"` // 策略唯一名 unkey
	Status       string     `gorm:"size:32; column:status" json:"status"`                // running / success / failed
	Output       string     `gorm:"type:text; column:output" json:"output"`
	Error        string     `gorm:"type:text; column:error" json:"error"`
	StartedAt    *time.Time `gorm:"column:started_at" json:"started_at"`
	FinishedAt   *time.Time `gorm:"column:finished_at" json:"finished_at"`
	CreatedAt    time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (TabledataStrategyLog) TableName() string { return "bs_dbstrategy_log" }

// StartStrategyLog 策略开始执行,写 running 日志(参照 StartJobLog)
func StartStrategyLog(kind string, id int64, name string) *TabledataStrategyLog {
	now := time.Now()
	jl := &TabledataStrategyLog{
		Kind:         kind,
		StrategyID:   id,
		StrategyName: name,
		Status:       LogRunning,
		StartedAt:    &now,
	}
	if err := dao.MysqlCli.Create(jl).Error; err != nil {
		// 写库失败不阻断执行, 但必须落错误日志, 避免"该跑没跑"在平台内不可观测
		glog.Z().Error("[rds] 写策略running日志失败 kind=" + kind + fmt.Sprintf(" id=%d err=%s", id, err.Error()))
	}
	return jl
}

// FinishStrategyLog 策略执行结束,回写结果(参照 FinishJobLog)
func FinishStrategyLog(jl *TabledataStrategyLog, output string, runErr error) {
	now := time.Now()
	jl.FinishedAt = &now
	jl.Output = truncate(output, 8192)
	if runErr != nil {
		jl.Status = LogFailed
		jl.Error = truncate(runErr.Error(), 2048)
	} else {
		jl.Status = LogSuccess
	}
	if err := dao.MysqlCli.Save(jl).Error; err != nil {
		glog.Z().Error("[rds] 回写策略终态日志失败" + fmt.Sprintf(" log_id=%d status=%s err=%s", jl.ID, jl.Status, err.Error()))
	}
}

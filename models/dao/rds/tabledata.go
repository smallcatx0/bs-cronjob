package rds

import (
	"cron-job/models/dao"
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

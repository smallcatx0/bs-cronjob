package dao

import (
	"cron-job/internal/conf"
	"cron-job/pkg/glog"
	"fmt"
	"log"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var MysqlCli *gorm.DB
var MdbPrefix string

func MustInitMysql() {
	c := conf.AppConf
	// 读配置
	dsn := c.GetString("mysql.dsn")
	MdbPrefix = c.GetString("mysql.prefix")
	isDebug := c.GetBool("mysql.debug")
	maxIdleConns := c.GetInt("mysql.maxIdleConns")
	maxOpenConns := c.GetInt("mysql.maxOpenConns")
	connMaxLifetime := c.GetInt("mysql.connMaxLifetime")

	db, err := ConnMysql(dsn, isDebug)
	if err != nil {
		log.Panic("[store_mysql] conn mysql fail err=", err)
	}
	mdb, _ := db.DB()
	mdb.SetMaxIdleConns(maxIdleConns)
	mdb.SetMaxOpenConns(maxOpenConns)
	mdb.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)
	// 赋给全局变量
	MysqlCli = db
}

func ConnMysql(dsn string, isDebug bool) (db *gorm.DB, err error) {
	w := &ZapWriter{glog.D().Z().With(zap.String("type", "sql_log"))}
	logger := logger.New(w, logger.Config{
		SlowThreshold: time.Millisecond * 200,
		LogLevel:      logger.Silent,
		Colorful:      false,
	})
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger,
	})
	if err != nil {
		return
	}
	if isDebug {
		db = db.Debug()
	}
	mdb, err := db.DB()
	if err != nil {
		return
	}
	err = mdb.Ping()
	if err != nil {
		return
	}
	return
}

func CloseTmpMysql(db *gorm.DB) {
	mdb, err := db.DB()
	if err != nil {
		return
	}
	mdb.Close()
}

// 数据库链接脱敏, 对空/非法 DSN 做安全兜底, 避免解析越界 panic
func DsnMask(dsn string) string {
	if dsn == "" {
		return ""
	}
	info := strings.SplitN(dsn, "@", 2)
	if len(info) < 2 {
		// 不含 @ 分隔符, 无法定位账号密码, 整体脱敏
		return "******"
	}
	u := strings.SplitN(info[0], ":", 2)
	return u[0] + ":******@" + info[1]
}

// 接管mysql 日志
type ZapWriter struct {
	*zap.Logger
}

func (w *ZapWriter) Printf(tpl string, args ...any) {
	tpl = strings.ReplaceAll(tpl, "\n", " ")
	msg := "[sql] " + fmt.Sprintf(tpl, args...)
	if _, ok := args[1].(error); ok {
		w.Error(msg)
	} else {
		w.Info(msg)
	}
}

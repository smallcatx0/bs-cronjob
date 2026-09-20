package dbstrategy

import (
	"context"
	"encoding/json"
	"testing"

	"cron-job/internal/tasks"
	"cron-job/models/dao"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

var test_db_dsn string
var test_redis_addr string

var ttl = TabledataTtl{
	ID:         1,
	UnKey:      "bs_sth_unitest",
	DbName:     "bs",
	Tablename:  "bs_sth_task",
	ColumnName: "updated_at",
	ColumnType: ColumType_Timestamp,
	Limit:      100,
	TtlValue:   3600 * 24 * 30, // 保存一个月
}

func Test_deleteTableRecord(t *testing.T) {

	ttl.Dsn = test_db_dsn
	stg, err := NewDbStrategy(nil, nil)
	assert.NoError(t, err)
	stg.Debug = true
	err = stg.deleteTableRecord(ttl)
	assert.NoError(t, err)
}

var dbRetry = TabledataRetry{
	ID:         1,
	Unkey:      "bs_sth_job_retry",
	Dsn:        "",
	DbName:     "bs",
	Tablename:  "bs_sth_task",
	ColumnName: "updated_at",
	ColumnType: "timestamp",
	FindWh:     "`status`=30",
	SetFields:  "`status`=1",
	Before:     380000,
	Duration:   3600,
	Limit:      200,
	Spec:       "",
	Desc:       "数据库重试任务-单元测试",
}

func Test_updateTableRecord(t *testing.T) {
	dbRetry.Dsn = test_db_dsn
	stg, err := NewDbStrategy(nil, nil)
	assert.NoError(t, err)
	stg.Debug = true
	err = stg.updateTableRecord(dbRetry)
	assert.NoError(t, err)
}

var (
	redisCli *redis.Client
	dbCli    *gorm.DB
)

func MustInitDao() {
	var err error
	redisCli, err = dao.ConnRedis(&redis.Options{
		Addr: test_redis_addr,
	})
	if err != nil {
		panic(err)
	}
	dbCli, err = dao.ConnMysql(test_db_dsn, true)
	if err != nil {
		panic(err)
	}
}
func Test_cronFun(t *testing.T) {
	var (
		spec = "* * * * *" // asynq 仅支持标准 5 段表达式, 最快每分钟触发
	)
	MustInitDao()
	stg, err := NewDbStrategy(dbCli, redisCli)
	assert.NoError(t, err)

	// 直接入队一次策略任务, 验证 asynq 链路打通(payload 仅携带 ID, 消费端按 ID 查配置库)
	p := ttl
	p.Dsn = test_db_dsn
	p.UnKey = p.UnKey + ":cron"
	b, err := json.Marshal(&Payload{Kind: KindTtl, ID: p.ID})
	assert.NoError(t, err)
	_, err = stg.client.EnqueueContext(context.Background(),
		asynq.NewTask(TypeTtlStrategy, b),
		asynq.Queue(tasks.StrategyQueue()),
		asynq.TaskID("dbstrategy:ttl:"+p.UnKey),
	)
	assert.NoError(t, err)

	// 调度器注册验证(不启动 Run, 仅确认 5 段表达式可被接受)
	stg.AddCronTask(spec, TypeTtlStrategy, "ttl:"+p.UnKey, &Payload{Kind: KindTtl, ID: p.ID})
	assert.Contains(t, stg.entries, "ttl:"+p.UnKey)
}

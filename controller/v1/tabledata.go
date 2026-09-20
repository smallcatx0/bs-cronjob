package v1

import (
	dbstrategy "cron-job/internal/db_strategy"
	"cron-job/middleware/resp"
	"cron-job/models/dao"
	"cron-job/models/dao/rds"
	"cron-job/models/valid"

	"github.com/gin-gonic/gin"
)

type Tabledata struct{}

// TtlList 列出 TTL 策略配置
func (Tabledata) TtlList(c *gin.Context) {
	p := valid.TtlQuery{}
	if err := valid.BindQueryAndCheck(c, &p); err != nil {
		resp.Fail(c, err)
		return
	}
	q := dao.MysqlCli.Model(&rds.TabledataTtl{})
	if p.UnKey != "" {
		q = q.Where("unkey LIKE ?", "%"+p.UnKey+"%")
	}
	if p.DbName != "" {
		q = q.Where("db_name = ?", p.DbName)
	}
	if p.Tablename != "" {
		q = q.Where("table_name LIKE ?", "%"+p.Tablename+"%")
	}
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	q = q.Order("id DESC")

	pg := resp.NewPage(c)
	q, err := pg.Paginate(q)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if pg.Total == 0 {
		resp.Paginate(c, pg, nil)
		return
	}
	list := make([]rds.TabledataTtl, 0, pg.Limit)
	if err = q.Find(&list).Error; err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Paginate(c, pg, list)
}

// TtlAdd 新建 TTL 策略配置(状态默认 offline, 不自动注册 asynq, 上线走 toggle 接口)
func (Tabledata) TtlAdd(c *gin.Context) {
	p := valid.TtlAdd{}
	if err := valid.BindJsonAndCheck(c, &p); err != nil {
		resp.Fail(c, err)
		return
	}
	if err := checkUnkeyTtl(p.UnKey); err != nil {
		resp.Fail(c, err)
		return
	}
	cfg := rds.TabledataTtl{
		UnKey:      p.UnKey,
		Dsn:        p.Dsn,
		DbName:     p.DbName,
		Tablename:  p.Tablename,
		ColumnName: p.ColumnName,
		ColumnType: p.ColumnType,
		TtlValue:   p.TtlValue,
		Limit:      p.Limit,
		Spec:       p.Spec,
		Status:     rds.StrategyOffline,
		Desc:       p.Desc,
	}
	if err := dao.MysqlCli.Create(&cfg).Error; err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Succ(c, cfg)
}

// TtlDetail TTL 策略配置详情
func (Tabledata) TtlDetail(c *gin.Context) {
	p := struct {
		ID int64 `form:"id" binding:"required"`
	}{}
	if err := valid.BindQueryAndCheck(c, &p); err != nil {
		resp.Fail(c, err)
		return
	}
	var cfg rds.TabledataTtl
	if err := dao.MysqlCli.First(&cfg, p.ID).Error; err != nil {
		resp.Fail(c, err)
		return
	}

	resp.Succ(c, cfg)
}

// TtlUpdate 更新 TTL 策略配置(仅 offline 状态可更新, 只更新提交的字段, unkey 不可变更)
func (Tabledata) TtlUpdate(c *gin.Context) {
	p := valid.TtlUpdate{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var cfg *rds.TabledataTtl
	cfg, err = cfg.GetByID(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if cfg.Status == rds.StrategyOnline {
		resp.Fail(c, resp.ParamInValid("online 状态的策略不可编辑, 请先退回 offline"))
		return
	}
	updates := map[string]interface{}{}
	if p.Dsn != "" {
		updates["dsn"] = p.Dsn
	}
	if p.DbName != "" {
		updates["db_name"] = p.DbName
	}
	if p.Tablename != "" {
		updates["table_name"] = p.Tablename
	}
	if p.ColumnName != "" {
		updates["column_name"] = p.ColumnName
	}
	if p.ColumnType != "" {
		updates["column_type"] = p.ColumnType
	}
	if p.TtlValue > 0 {
		updates["ttl_value"] = p.TtlValue
	}
	if p.Limit > 0 {
		updates["limit"] = p.Limit
	}
	if p.Desc != "" {
		updates["desc"] = p.Desc
	}
	if len(updates) > 0 {
		if err = dao.MysqlCli.Model(&rds.TabledataTtl{}).
			Where("id = ?", cfg.ID).
			Updates(updates).Error; err != nil {
			resp.Fail(c, err)
			return
		}
	}
	cfg, _ = cfg.GetByID(p.ID)
	resp.Succ(c, cfg)
}

// TtlDelete 删除 TTL 策略配置(仅 offline 状态可删除)
func (Tabledata) TtlDelete(c *gin.Context) {
	p := struct {
		ID int64 `json:"id" binding:"required"`
	}{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var cfg *rds.TabledataTtl
	cfg, err = cfg.GetByID(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if cfg.Status == rds.StrategyOnline {
		resp.Fail(c, resp.ParamInValid("online 状态的策略不可删除, 请先退回 offline"))
		return
	}
	if err = dao.MysqlCli.Delete(cfg).Error; err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Succ(c, cfg)
}

// TtlToggle 切换 TTL 策略状态(offline->online 注册到 asynq 调度, online->offline 撤销调度)
func (Tabledata) TtlToggle(c *gin.Context) {
	p := struct {
		ID     int64  `json:"id" binding:"required"`
		Status string `json:"status" binding:"required,oneof=offline online"` // offline/online
	}{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var cfg *rds.TabledataTtl
	cfg, err = cfg.GetByID(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if cfg.Status == p.Status {
		resp.Succ(c, cfg)
		return
	}
	funName := "ttl:" + cfg.UnKey
	stg := dbstrategy.Instance()
	if p.Status == rds.StrategyOnline { // 上线: 先注册调度, 再落库状态
		if err = valid.CheckCronExpr(cfg.Spec); err != nil {
			resp.Fail(c, resp.ParamInValid(err.Error()))
			return
		}
		payload := &dbstrategy.Payload{Kind: dbstrategy.KindTtl, ID: cfg.ID}
		if err = stg.AddCronTask(cfg.Spec, dbstrategy.TypeTtlStrategy, funName, payload); err != nil {
			resp.Fail(c, resp.ParamInValid("策略注册失败", err.Error()))
			return
		}
		if err = cfg.SetStatus(cfg.ID, p.Status); err != nil {
			stg.RemoveCronTask(funName) // 落库失败回滚注册
			resp.Fail(c, err)
			return
		}
	} else { // 下线: 先落库状态, 再撤销调度
		if err = cfg.SetStatus(cfg.ID, p.Status); err != nil {
			resp.Fail(c, err)
			return
		}
		stg.RemoveCronTask(funName)
	}
	cfg, _ = cfg.GetByID(p.ID)
	resp.Succ(c, cfg)
}

// RetryList 列出 Retry 策略配置
func (Tabledata) RetryList(c *gin.Context) {
	p := valid.RetryQuery{}
	if err := valid.BindQueryAndCheck(c, &p); err != nil {
		resp.Fail(c, err)
		return
	}
	q := dao.MysqlCli.Model(&rds.TabledataRetry{})
	if p.Unkey != "" {
		q = q.Where("unkey LIKE ?", "%"+p.Unkey+"%")
	}
	if p.DbName != "" {
		q = q.Where("db_name = ?", p.DbName)
	}
	if p.Tablename != "" {
		q = q.Where("table_name LIKE ?", "%"+p.Tablename+"%")
	}
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	q = q.Order("id DESC")

	pg := resp.NewPage(c)
	q, err := pg.Paginate(q)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if pg.Total == 0 {
		resp.Paginate(c, pg, nil)
		return
	}
	list := make([]rds.TabledataRetry, 0, pg.Limit)
	if err = q.Find(&list).Error; err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Paginate(c, pg, list)
}

// RetryAdd 新建 Retry 策略配置(状态默认 offline, 不自动注册 asynq, 上线走 toggle 接口)
func (Tabledata) RetryAdd(c *gin.Context) {
	p := valid.RetryAdd{}
	if err := valid.BindJsonAndCheck(c, &p); err != nil {
		resp.Fail(c, err)
		return
	}
	if err := checkUnkeyRetry(p.Unkey); err != nil {
		resp.Fail(c, err)
		return
	}
	cfg := rds.TabledataRetry{
		Unkey:      p.Unkey,
		Dsn:        p.Dsn,
		DbName:     p.DbName,
		Tablename:  p.Tablename,
		ColumnName: p.ColumnName,
		ColumnType: p.ColumnType,
		FindWh:     p.FindWh,
		SetFields:  p.SetFields,
		Before:     p.Before,
		Duration:   p.Duration,
		Limit:      p.Limit,
		Spec:       p.Spec,
		Status:     rds.StrategyOffline,
		Desc:       p.Desc,
	}
	if err := dao.MysqlCli.Create(&cfg).Error; err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Succ(c, cfg)
}

// RetryDetail Retry 策略配置详情
func (Tabledata) RetryDetail(c *gin.Context) {
	p := struct {
		ID int64 `form:"id" binding:"required"`
	}{}
	err := valid.BindQueryAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var cfg *rds.TabledataRetry
	cfg, err = cfg.GetByID(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Succ(c, cfg)
}

// RetryUpdate 更新 Retry 策略配置(仅 offline 状态可更新, 只更新提交的字段, unkey 不可变更)
func (Tabledata) RetryUpdate(c *gin.Context) {
	p := valid.RetryUpdate{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var cfg *rds.TabledataRetry
	cfg, err = cfg.GetByID(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if cfg.Status == rds.StrategyOnline {
		resp.Fail(c, resp.ParamInValid("online 状态的策略不可编辑, 请先退回 offline"))
		return
	}
	updates := map[string]interface{}{}
	if p.Dsn != "" {
		updates["dsn"] = p.Dsn
	}
	if p.DbName != "" {
		updates["db_name"] = p.DbName
	}
	if p.Tablename != "" {
		updates["table_name"] = p.Tablename
	}
	if p.ColumnName != "" {
		updates["column_name"] = p.ColumnName
	}
	if p.ColumnType != "" {
		updates["column_type"] = p.ColumnType
	}
	if p.FindWh != "" {
		updates["find_wh"] = p.FindWh
	}
	if p.SetFields != "" {
		updates["set_fields"] = p.SetFields
	}
	if p.Before > 0 {
		updates["before"] = p.Before
	}
	if p.Duration > 0 {
		updates["duration"] = p.Duration
	}
	if p.Limit > 0 {
		updates["limit"] = p.Limit
	}
	if p.Desc != "" {
		updates["desc"] = p.Desc
	}
	if len(updates) > 0 {
		err = dao.MysqlCli.
			Model(&rds.TabledataRetry{}).
			Where("id = ?", cfg.ID).
			Updates(updates).Error
		if err != nil {
			resp.Fail(c, err)
			return
		}
	}
	cfg, _ = cfg.GetByID(p.ID)
	resp.Succ(c, cfg)
}

// RetryDelete 删除 Retry 策略配置(仅 offline 状态可删除)
func (Tabledata) RetryDelete(c *gin.Context) {
	p := struct {
		ID int64 `json:"id" binding:"required"`
	}{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var cfg *rds.TabledataRetry
	cfg, err = cfg.GetByID(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if cfg.Status == rds.StrategyOnline {
		resp.Fail(c, resp.ParamInValid("online 状态的策略不可删除, 请先退回 offline"))
		return
	}
	if err = dao.MysqlCli.Delete(cfg).Error; err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Succ(c, cfg)
}

// RetryToggle 切换 Retry 策略状态(offline->online 注册到 asynq 调度, online->offline 撤销调度)
func (Tabledata) RetryToggle(c *gin.Context) {
	p := struct {
		ID     int64  `json:"id" binding:"required"`
		Status string `json:"status" binding:"required,oneof=offline online"` // offline/online
	}{}
	err := valid.BindJsonAndCheck(c, &p)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	var cfg *rds.TabledataRetry
	cfg, err = cfg.GetByID(p.ID)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if cfg.Status == p.Status {
		resp.Succ(c, cfg)
		return
	}
	funName := "retry:" + cfg.Unkey
	stg := dbstrategy.Instance()
	if p.Status == rds.StrategyOnline { // 上线: 先注册调度, 再落库状态
		if err = valid.CheckCronExpr(cfg.Spec); err != nil {
			resp.Fail(c, resp.ParamInValid(err.Error()))
			return
		}
		payload := &dbstrategy.Payload{Kind: dbstrategy.KindRetry, ID: cfg.ID}
		if err = stg.AddCronTask(cfg.Spec, dbstrategy.TypeRetryStrategy, funName, payload); err != nil {
			resp.Fail(c, resp.ParamInValid("策略注册失败", err.Error()))
			return
		}
		if err = cfg.SetStatus(cfg.ID, p.Status); err != nil {
			stg.RemoveCronTask(funName) // 落库失败回滚注册
			resp.Fail(c, err)
			return
		}
	} else { // 下线: 先落库状态, 再撤销调度
		if err = cfg.SetStatus(cfg.ID, p.Status); err != nil {
			resp.Fail(c, err)
			return
		}
		stg.RemoveCronTask(funName)
	}
	cfg, _ = cfg.GetByID(p.ID)
	resp.Succ(c, cfg)
}

// StrategyLog 分页查询 TTL/Retry 策略执行日志(按开始时间倒序, 与 jobs/log 同构)
func (Tabledata) StrategyLog(c *gin.Context) {
	p := valid.StrategyLogQuery{}
	if err := valid.BindQueryAndCheck(c, &p); err != nil {
		resp.Fail(c, err)
		return
	}
	q := dao.MysqlCli.Model(&rds.TabledataStrategyLog{})
	if p.Kind != "" {
		q = q.Where("kind = ?", p.Kind)
	}
	if p.StrategyID > 0 {
		q = q.Where("strategy_id = ?", p.StrategyID)
	}
	if p.Status != "" {
		q = q.Where("status = ?", p.Status)
	}
	if p.Start != "" {
		q = q.Where("started_at >= ?", p.Start)
	}
	if p.End != "" {
		q = q.Where("started_at <= ?", p.End)
	}
	q = q.Order("started_at DESC, id DESC")

	pg := resp.NewPage(c)
	q, err := pg.Paginate(q)
	if err != nil {
		resp.Fail(c, err)
		return
	}
	if pg.Total == 0 {
		resp.Paginate(c, pg, nil)
		return
	}
	logs := make([]rds.TabledataStrategyLog, 0, pg.Limit)
	if err = q.Find(&logs).Error; err != nil {
		resp.Fail(c, err)
		return
	}
	resp.Paginate(c, pg, logs)
}

// checkUnkeyTtl 校验 TTL 策略 unkey 唯一(作为 asynq 任务名组成部分)
func checkUnkeyTtl(unkey string) error {
	var cnt int64
	dao.MysqlCli.Model(&rds.TabledataTtl{}).Where("unkey = ?", unkey).Count(&cnt)
	return uniqErr(cnt)
}

// checkUnkeyRetry 校验 Retry 策略 unkey 唯一(作为 asynq 任务名组成部分)
func checkUnkeyRetry(unkey string) error {
	var cnt int64
	dao.MysqlCli.Model(&rds.TabledataRetry{}).Where("unkey = ?", unkey).Count(&cnt)
	return uniqErr(cnt)
}

func uniqErr(cnt int64) error {
	if cnt > 0 {
		return resp.ParamInValid("unkey 已存在, 请更换")
	}
	return nil
}

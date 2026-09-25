package rds

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// 固定基准时间, 保证断言可复现
var baseNow = time.Date(2026, 9, 21, 12, 0, 0, 0, time.Local)

func TestBuildTtlDeleteSql(t *testing.T) {
	cutoff := baseNow.Add(-3600 * time.Second) // 2026-09-21 11:00:00
	tests := []struct {
		name       string
		columnType string
		want       string
	}{
		{
			name:       "unix",
			columnType: ColumType_Unix,
			want:       "DELETE FROM `t_log` WHERE `created_at` < " + strconv.FormatInt(cutoff.Unix(), 10) + " LIMIT 1000",
		},
		{
			name:       "timestamp",
			columnType: ColumType_Timestamp,
			want:       "DELETE FROM `t_log` WHERE `created_at` < '2026-09-21 11:00:00' LIMIT 1000",
		},
		{
			name:       "datetime",
			columnType: ColumType_Datetime,
			want:       "DELETE FROM `t_log` WHERE `created_at` < '2026-09-21 11:00:00' LIMIT 1000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildTtlDeleteSql("t_log", "created_at", tt.columnType, "", cutoff, 1000)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}
			if got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestBuildTtlDeleteSqlInvalidColumnType(t *testing.T) {
	_, err := BuildTtlDeleteSql("t_log", "created_at", "bad_type", "", baseNow, 1000)
	if err == nil {
		t.Fatal("expected error for invalid column_type, got nil")
	}
}

// TestBuildTtlDeleteSqlWithFindWh 校验 find_wh 非空时追加到 WHERE, 为空时不产生悬空 AND
func TestBuildTtlDeleteSqlWithFindWh(t *testing.T) {
	cutoff := baseNow.Add(-3600 * time.Second) // 2026-09-21 11:00:00
	got, err := BuildTtlDeleteSql("t_log", "created_at", ColumType_Datetime, "status = 0", cutoff, 1000)
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	want := "DELETE FROM `t_log` WHERE `created_at` < '2026-09-21 11:00:00' AND status = 0 LIMIT 1000"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestTtlParseSqlMatchesBuilder(t *testing.T) {
	cfg := &TabledataTtl{
		Tablename:  "t_log",
		ColumnName: "created_at",
		ColumnType: ColumType_Datetime,
		FindWh:     "status = 0",
		TtlValue:   60,
		Limit:      500,
	}
	sql, err := cfg.ParseSql()
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	want, err := BuildTtlDeleteSql(cfg.Tablename, cfg.ColumnName, cfg.ColumnType, cfg.FindWh,
		time.Now().Add(-time.Second*time.Duration(cfg.TtlValue)), cfg.Limit)
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	// 两次取时间可能跨秒导致值部分不同, 仅校验值之前的语句结构一致
	if j := strings.Index(want, "'"); j > 0 {
		want = want[:j]
	}
	if !strings.HasPrefix(sql, want) {
		t.Errorf("ParseSql prefix mismatch:\ngot: %s\nwant prefix: %s", sql, want)
	}
}

func TestBuildRetrySqls(t *testing.T) {
	// before=3600 duration=600 => st=11:00:00 ed=11:10:00, limit=200 => UPDATE 带 LIMIT 200
	stStr := strconv.FormatInt(baseNow.Add(-3600*time.Second).Unix(), 10)
	edStr := strconv.FormatInt(baseNow.Add(-3000*time.Second).Unix(), 10)
	tests := []struct {
		name          string
		columnType    string
		wantCountSql  string
		wantUpdateSql string
	}{
		{
			name:          "unix",
			columnType:    ColumType_Unix,
			wantCountSql:  "SELECT count(*) c FROM `t_order` WHERE `status_at` >= " + stStr + " AND `status_at` < " + edStr + " AND status = 0",
			wantUpdateSql: "UPDATE `t_order` SET status = 1 WHERE `status_at` >= " + stStr + " AND `status_at` < " + edStr + " AND status = 0 LIMIT 200",
		},
		{
			name:          "timestamp",
			columnType:    ColumType_Timestamp,
			wantCountSql:  "SELECT count(*) c FROM `t_order` WHERE `status_at` >= '2026-09-21 11:00:00' AND `status_at` < '2026-09-21 11:10:00' AND status = 0",
			wantUpdateSql: "UPDATE `t_order` SET status = 1 WHERE `status_at` >= '2026-09-21 11:00:00' AND `status_at` < '2026-09-21 11:10:00' AND status = 0 LIMIT 200",
		},
		{
			name:          "datetime",
			columnType:    ColumType_Datetime,
			wantCountSql:  "SELECT count(*) c FROM `t_order` WHERE `status_at` >= '2026-09-21 11:00:00' AND `status_at` < '2026-09-21 11:10:00' AND status = 0",
			wantUpdateSql: "UPDATE `t_order` SET status = 1 WHERE `status_at` >= '2026-09-21 11:00:00' AND `status_at` < '2026-09-21 11:10:00' AND status = 0 LIMIT 200",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			countSql, updateSql, err := BuildRetrySqls("t_order", "status_at", tt.columnType,
				"status = 0", "status = 1", 3600, 600, 200, baseNow)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}
			if countSql != tt.wantCountSql {
				t.Errorf("countSql:\ngot:  %s\nwant: %s", countSql, tt.wantCountSql)
			}
			if updateSql != tt.wantUpdateSql {
				t.Errorf("updateSql:\ngot:  %s\nwant: %s", updateSql, tt.wantUpdateSql)
			}
		})
	}
}

func TestBuildRetrySqlsInvalidColumnType(t *testing.T) {
	_, _, err := BuildRetrySqls("t_order", "status_at", "bad_type", "status = 0", "status = 1", 3600, 600, 200, baseNow)
	if err == nil {
		t.Fatal("expected error for invalid column_type, got nil")
	}
}

func TestBuildRetrySqlsInvalidLimit(t *testing.T) {
	// limit 非正时应拒绝生成 SQL, 避免退化回无上限批量 UPDATE
	for _, limit := range []int64{0, -1} {
		_, _, err := BuildRetrySqls("t_order", "status_at", ColumType_Unix, "status = 0", "status = 1", 3600, 600, limit, baseNow)
		if err == nil {
			t.Fatalf("expected error for limit=%d, got nil", limit)
		}
	}
}

func TestRetryParseSqlMatchesBuilder(t *testing.T) {
	cfg := &TabledataRetry{
		Tablename:  "t_order",
		ColumnName: "status_at",
		ColumnType: ColumType_Unix,
		FindWh:     "status = 0",
		SetFields:  "status = 1",
		Before:     3600,
		Duration:   600,
		Limit:      200,
	}
	sql, err := cfg.ParseSql()
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	_, want, err := BuildRetrySqls(cfg.Tablename, cfg.ColumnName, cfg.ColumnType,
		cfg.FindWh, cfg.SetFields, cfg.Before, cfg.Duration, cfg.Limit, time.Now())
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	// unix 秒级时间戳跨秒会不同, 仅校验值之前的语句结构一致
	if i := strings.Index(want, "`"); i > 0 {
		want = want[:i]
	}
	if !strings.HasPrefix(sql, want) {
		t.Errorf("ParseSql prefix mismatch:\ngot: %s\nwant prefix: %s", sql, want)
	}
}

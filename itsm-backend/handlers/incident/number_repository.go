package incident

import (
	"context"
	"database/sql"
	"fmt"
)

// incidentNumberRepository 负责 incident 编号生成。
//
// 设计要点：
//   - 编号由 PostgreSQL 序列 incident_number_seq 原子生成，
//     彻底消除「COUNT+1 并发复用 / 删除回退 / UTC 窗口错乱」三类编号冲突。
//   - 前缀 INC-YYYYMM 使用调用方传入的本地年月（service.go 用 time.Now()
//     本地时区），与统计窗口口径一致，不再出现 UTC 空窗 INC-...-000001。
//   - (tenant_id, incident_number) 唯一索引作为最后兜底，即便异常也不会
//     落库重复编号。
//
// 序列在 database.InitDatabase 中创建并播种为「历史最大后缀+1」。
type incidentNumberRepository struct {
	db *sql.DB
}

func newIncidentNumberRepository(db *sql.DB) *incidentNumberRepository {
	if db == nil {
		return nil
	}
	return &incidentNumberRepository{db: db}
}

// Generate 返回形如 INC-YYYYMM-NNNNNN 的事件编号。
// 参数 year/month 由调用方提供本地时区的年月，避免 server UTC 漂移。
func (r *incidentNumberRepository) Generate(ctx context.Context, year, month int) (string, error) {
	var seq int64
	if err := r.db.QueryRowContext(ctx,
		"SELECT nextval('incident_number_seq')",
	).Scan(&seq); err != nil {
		return "", fmt.Errorf("generate incident number: %w", err)
	}
	return fmt.Sprintf("INC-%04d%02d-%06d", year, month, seq), nil
}

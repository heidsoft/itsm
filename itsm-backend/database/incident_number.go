package database

import (
	"context"
	"database/sql"
	"fmt"
)

// GenerateIncidentNumber 是 INC-YYYYMM-NNNNNN 编号的唯一生成入口。
// 序列 incident_number_seq 由 EnsureOperationalIndexes 创建并播种；
// HTTP 创建（handlers/incident）、BPMN 服务任务（service/bpmn）、云信号自动建单
// 与邮件建单（service.IncidentService）共用，禁止在调用方各自拼接编号格式
// 或引入第二套号源。
func GenerateIncidentNumber(ctx context.Context, db *sql.DB, year, month int) (string, error) {
	var seq int64
	if err := db.QueryRowContext(ctx,
		"SELECT nextval('incident_number_seq')",
	).Scan(&seq); err != nil {
		return "", fmt.Errorf("generate incident number: %w", err)
	}
	return fmt.Sprintf("INC-%04d%02d-%06d", year, month, seq), nil
}

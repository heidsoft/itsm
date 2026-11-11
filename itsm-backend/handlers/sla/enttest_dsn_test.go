package sla

import (
	"fmt"
	"sync/atomic"
)

var slaTestDBCounter int64

// testDSN 每次调用返回唯一的 SQLite 内存数据库 DSN。
// 原 helper 在 service 包测试文件里，跨包不可见；随实现迁入本域后沿用同一口径，
// 避免上一个测试的连接池尚未关闭时复用同一个残留数据库。
func testDSN() string {
	return fmt.Sprintf("file:sla_test_%d?mode=memory&cache=shared&_fk=1", atomic.AddInt64(&slaTestDBCounter, 1))
}

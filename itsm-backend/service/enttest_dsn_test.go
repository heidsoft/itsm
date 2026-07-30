package service

import (
	"fmt"
	"sync/atomic"
)

var testDBCounter int64

// testDSN 每次调用返回唯一的 SQLite 内存数据库 DSN。
// 历史上所有测试共用 "file:ent?mode=memory&cache=shared&_fk=1"，
// 当上一个测试的连接池尚未完全关闭时，下一个测试会连接到同一个残留数据库，
// 触发 users.email 等唯一约束的偶发冲突。唯一命名可彻底隔离测试间数据。
func testDSN() string {
	return fmt.Sprintf("file:svc_test_%d?mode=memory&cache=shared&_fk=1", atomic.AddInt64(&testDBCounter, 1))
}

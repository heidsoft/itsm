package project

import (
	"fmt"
	"sync/atomic"
)

var testDBCounter int64

// testDSN 每次调用返回唯一的 SQLite 内存数据库 DSN，隔离测试间残留数据
// （共享命名会在上一个测试连接池未完全关闭时撞唯一约束）。
func testDSN() string {
	return fmt.Sprintf("file:project_test_%d?mode=memory&cache=shared&_fk=1", atomic.AddInt64(&testDBCounter, 1))
}

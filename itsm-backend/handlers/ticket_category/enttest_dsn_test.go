package ticket_category

import (
	"fmt"
	"sync/atomic"
)

var testDBCounter int64

// testDSN 每次调用返回唯一的 SQLite 内存数据库 DSN，隔离测试间残留数据。
func testDSN() string {
	return fmt.Sprintf("file:ticket_category_test_%d?mode=memory&cache=shared&_fk=1", atomic.AddInt64(&testDBCounter, 1))
}

package timer

import (
	"context"

	"itsm-backend/dto"
	"itsm-backend/ent"
)

// TimerReadStore 是 timer 读面与重放所需的最小能力集，由组合根注入
// 共享层定时器存储接口的实现。
//
// 接口名刻意不叫 TimerStore：共享层的同名接口若回退进本包，集中式回归锁的
// 词边界断言会把共享层同名接口的限定符误判成已使用域内契约而假绿。
// List/Stats 的入参与返回值使用 dto 中的同一类型（service 侧为类型别名），
// 因此实现的方法签名与此处完全一致。
type TimerReadStore interface {
	GetByTimerID(ctx context.Context, timerID string) (*ent.ProcessTimer, error)
	List(ctx context.Context, filter dto.TimerListFilter) ([]*ent.ProcessTimer, int, error)
	Replay(ctx context.Context, tenantID int, timerID string) (*ent.ProcessTimer, error)
	Stats(ctx context.Context, tenantID int) (*dto.TimerStats, error)
}

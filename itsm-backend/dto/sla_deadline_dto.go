// SLA 截止时间计算结果：批次 3 第 5 步 B 从共享层下沉，供 problem 域窄接口引用。
package dto

import "time"

// SLADeadlineResult SLA截止时间计算结果
type SLADeadlineResult struct {
	SLADefinitionID    int
	ResponseDeadline   *time.Time
	ResolutionDeadline *time.Time
	BusinessHoursOnly  bool
}

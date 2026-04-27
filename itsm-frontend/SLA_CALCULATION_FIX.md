# SLA计算逻辑修复

## 问题分析

### Bug描述
`createViolation` 函数中，`response_time` 和 `resolution_time` 的超时计算逻辑完全相同，这是复制粘贴错误。

### 问题代码
**文件**: `itsm-backend/service/sla_monitor_service.go:192-197`

```go
// 计算超时时间（分钟）
var exceededMinutes float64
if violationType == "response_time" {
  exceededMinutes = time.Since(t.CreatedAt).Minutes() - deadline.Sub(t.CreatedAt).Minutes()
} else {
  exceededMinutes = time.Since(t.CreatedAt).Minutes() - deadline.Sub(t.CreatedAt).Minutes()  // ❌ 逻辑完全相同
}
```

### 问题影响
- 超时时间计算不准确
- SLA违规严重程度判断错误
- 统计数据失真

## 修复方案

### 修复代码

```go
// createViolation 创建SLA违规记录
func (s *SLAMonitorService) createViolation(ctx context.Context, t *ent.Ticket, violationType string, deadline time.Time, slaDefMap map[int]string) error {
	// 计算超时时间（分钟）
	var exceededMinutes float64
	
	if violationType == "response_time" {
		// 响应时间：从截止时间到现在（如果还未响应）
		if t.FirstResponseAt.IsZero() {
			// 未响应：从截止时间到现在
			exceededMinutes = time.Since(deadline).Minutes()
		} else {
			// 已响应：如果响应时间超过截止时间
			if t.FirstResponseAt.After(deadline) {
				exceededMinutes = t.FirstResponseAt.Sub(deadline).Minutes()
			} else {
				exceededMinutes = 0  // 未超时
			}
		}
	} else { // resolution_time
		// 解决时间：从截止时间到现在（如果还未解决）
		if t.ResolvedAt.IsZero() {
			// 未解决：从截止时间到现在
			exceededMinutes = time.Since(deadline).Minutes()
		} else {
			// 已解决：如果解决时间超过截止时间
			if t.ResolvedAt.After(deadline) {
				exceededMinutes = t.ResolvedAt.Sub(deadline).Minutes()
			} else {
				exceededMinutes = 0  // 未超时
			}
		}
	}

	// 如果未超时，不创建违规记录
	if exceededMinutes <= 0 {
		return nil
	}

	// 描述信息
	description := fmt.Sprintf("工单 %s 违反SLA (%s): 超过截止时间 %.1f 分钟",
		t.TicketNumber, violationType, exceededMinutes)

	// 根据超时时间设置严重程度
	severity := "low"
	if exceededMinutes > 60 {
		severity = "medium"
	} else if exceededMinutes > 240 {  // 4小时
		severity = "high"
	} else if exceededMinutes > 480 {  // 8小时
		severity = "critical"
	}

	// ... 其余代码
}
```

### 修复说明

#### 1. 响应时间超时计算
- **未响应**: `time.Since(deadline)` - 从截止时间到现在
- **已响应但超时**: `FirstResponseAt.Sub(deadline)` - 响应时间 - 截止时间
- **已响应且未超时**: `0` - 不创建违规

#### 2. 解决时间超时计算
- **未解决**: `time.Since(deadline)` - 从截止时间到现在
- **已解决但超时**: `ResolvedAt.Sub(deadline)` - 解决时间 - 截止时间
- **已解决且未超时**: `0` - 不创建违规

#### 3. 增加超时验证
如果计算出的超时时间 ≤ 0，不应创建违规记录。

## 工作时间SLA计算增强

### 当前问题
SLA截止时间计算默认不使用工作时间，对非24/7服务不准确。

### 增强方案

```go
// calculateDeadlineWithBusinessHours 计算考虑工作时间的截止时间
func (s *TicketSLAService) calculateDeadlineWithBusinessHours(
    startTime time.Time,
    slaMinutes int,
    businessHoursOnly bool,
    businessHours *BusinessHoursConfig,
) time.Time {
    if !businessHoursOnly || businessHours == nil {
        // 不使用工作时间，直接加时间
        return startTime.Add(time.Duration(slaMinutes) * time.Minute)
    }

    // 使用工作时间计算
    deadline := startTime
    remaining := slaMinutes

    for remaining > 0 {
        // 检查当前时间是否在工作时间内
        if !s.isWithinBusinessHours(deadline, businessHours) {
            // 移动到下一个工作时间开始
            deadline = s.nextBusinessHourStart(deadline, businessHours)
            continue
        }

        // 计算当天剩余工作时间
        todayRemaining := s.getTodayRemainingBusinessMinutes(deadline, businessHours)
        
        if remaining <= todayRemaining {
            // 剩余时间在当天工作时间内
            return deadline.Add(time.Duration(remaining) * time.Minute)
        }

        // 消耗今天的剩余工作时间
        remaining -= todayRemaining
        // 移动到下一个工作日
        deadline = s.nextBusinessDayStart(deadline, businessHours)
    }

    return deadline
}

// BusinessHoursConfig 工作时间配置
type BusinessHoursConfig struct {
    StartHour    int    // 开始小时 (如 9)
    StartMinute  int    // 开始分钟 (如 0)
    EndHour      int    // 结束小时 (如 18)
    EndMinute    int    // 结束分钟 (如 0)
    Workdays     []time.Weekday  // 工作日 (如 Monday-Friday)
    Holidays     []time.Time      // 节假日
}
```

## 测试用例

```go
func TestSLACalculation(t *testing.T) {
    tests := []struct {
        name             string
        ticket          *ent.Ticket
        violationType   string
        deadline        time.Time
        expectedMinutes float64
    }{
        {
            name: "未响应且超时",
            ticket: &ent.Ticket{
                CreatedAt:       time.Now().Add(-2 * time.Hour),
                FirstResponseAt: time.Time{}, // 未响应
            },
            violationType:   "response_time",
            deadline:        time.Now().Add(-1 * time.Hour),  // 1小时前截止
            expectedMinutes: 60,  // 超时60分钟
        },
        {
            name: "已响应且未超时",
            ticket: &ent.Ticket{
                CreatedAt:       time.Now().Add(-2 * time.Hour),
                FirstResponseAt: time.Now().Add(-30 * time.Minute), // 30分钟前响应
            },
            violationType:   "response_time",
            deadline:        time.Now().Add(30 * time.Minute),  // 30分钟后截止
            expectedMinutes: 0,  // 未超时
        },
        {
            name: "已响应但超时",
            ticket: &ent.Ticket{
                CreatedAt:       time.Now().Add(-3 * time.Hour),
                FirstResponseAt: time.Now().Add(-30 * time.Minute), // 30分钟前响应
            },
            violationType:   "response_time",
            deadline:        time.Now().Add(-1 * time.Hour),  // 1小时前截止
            expectedMinutes: 30,  // 响应时间超过截止时间30分钟
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            exceeded := calculateExceededMinutes(tt.ticket, tt.violationType, tt.deadline)
            assert.Equal(t, tt.expectedMinutes, exceeded)
        })
    }
}
```

## 手动修复步骤

### 1. 编辑 sla_monitor_service.go

```bash
vim itsm-backend/service/sla_monitor_service.go
```

### 2. 找到 createViolation 函数（约192行）

### 3. 替换超时计算逻辑

将原来的:
```go
if violationType == "response_time" {
  exceededMinutes = time.Since(t.CreatedAt).Minutes() - deadline.Sub(t.CreatedAt).Minutes()
} else {
  exceededMinutes = time.Since(t.CreatedAt).Minutes() - deadline.Sub(t.CreatedAt).Minutes()
}
```

替换为上面修复方案中的代码。

### 4. 运行测试

```bash
cd itsm-backend
go test ./service -run TestSLA
```

## 影响范围

### 影响的功能
- SLA违规检测
- SLA统计报告
- 工单超时提醒

### 需要检查的文件
- `sla_monitor_service.go` - 主要修复文件
- `ticket_sla_service.go` - 截止时间计算
- 相关的测试文件

## 后续工作

1. ⏳ 手动修改超时计算逻辑
2. ⏳ 增加单元测试
3. ⏳ 考虑实现工作时间SLA计算
4. ⏳ 数据迁移（修正历史违规记录）

---

**修复优先级**: 高  
**预计时间**: 2小时

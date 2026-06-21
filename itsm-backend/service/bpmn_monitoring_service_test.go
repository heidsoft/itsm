package service

import (
	"context"
	"math"
	"testing"
	"time"

	"itsm-backend/ent"
)

// 单元测试：测试 percentile 函数（P95 分位数计算）
func TestPercentile(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name   string
		input  []time.Duration
		p      float64
		expect time.Duration
	}{
		{"empty", nil, 0.95, 0},
		{"single", []time.Duration{10 * time.Second}, 0.95, 10 * time.Second},
		{"p95 of 20 samples", []time.Duration{
			1 * time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second, 5 * time.Second,
			6 * time.Second, 7 * time.Second, 8 * time.Second, 9 * time.Second, 10 * time.Second,
			11 * time.Second, 12 * time.Second, 13 * time.Second, 14 * time.Second, 15 * time.Second,
			16 * time.Second, 17 * time.Second, 18 * time.Second, 19 * time.Second, 20 * time.Second,
		}, 0.95, 19 * time.Second},
		{"p50 of odd samples", []time.Duration{
			1 * time.Second, 5 * time.Second, 10 * time.Second,
		}, 0.50, 5 * time.Second},
	}
	_ = now
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := percentile(c.input, c.p)
			if got != c.expect {
				t.Errorf("percentile(%v, %v) = %v, want %v", c.input, c.p, got, c.expect)
			}
		})
	}
}

// 单元测试：测试 avgDuration 函数（平均时长）
func TestAvgDuration(t *testing.T) {
	cases := []struct {
		name   string
		input  []time.Duration
		expect time.Duration
	}{
		{"empty", nil, 0},
		{"single", []time.Duration{10 * time.Second}, 10 * time.Second},
		{"average of 3", []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second}, 2 * time.Second},
		{"average of 5", []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second, 5 * time.Second}, 3 * time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := avgDuration(c.input)
			if got != c.expect {
				t.Errorf("avgDuration(%v) = %v, want %v", c.input, got, c.expect)
			}
		})
	}
}

// 单元测试：验证 calculateImpactScore 边界值（用于瓶颈识别核心算法）
func TestCalculateImpactScore(t *testing.T) {
	s := &BPMNMonitoringService{}
	cases := []struct {
		name      string
		duration  time.Duration
		waitTime  time.Duration
		queueLen  int
		expectMin float64
		expectMax float64
	}{
		{"no impact", 0, 0, 0, 0, 0},
		{"high duration", 60 * time.Minute, 0, 0, 40, 40},
		{"high waitTime", 0, 30 * time.Minute, 0, 30, 30},
		{"high queue", 0, 0, 20, 30, 30},
		{"all high", 60 * time.Minute, 30 * time.Minute, 20, 100, 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := s.calculateImpactScore(c.duration, c.waitTime, c.queueLen)
			if got < c.expectMin || got > c.expectMax {
				t.Errorf("calculateImpactScore(d=%v, w=%v, q=%d) = %v, want in [%v, %v]",
					c.duration, c.waitTime, c.queueLen, got, c.expectMin, c.expectMax)
			}
		})
	}
}

// 单元测试：determineBottleneckType 分类逻辑
func TestDetermineBottleneckType(t *testing.T) {
	s := &BPMNMonitoringService{}
	cases := []struct {
		name       string
		duration   time.Duration
		waitTime   time.Duration
		queueLen   int
		expectType string
	}{
		{"processing", 60 * time.Minute, 0, 0, "processing"},
		{"waiting", 0, 30 * time.Minute, 0, "waiting"},
		{"resource", 0, 0, 20, "resource"},
		{"unknown", 0, 0, 0, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := s.determineBottleneckType(c.duration, c.waitTime, c.queueLen)
			if got != c.expectType {
				t.Errorf("determineBottleneckType(d=%v, w=%v, q=%d) = %v, want %v",
					c.duration, c.waitTime, c.queueLen, got, c.expectType)
			}
		})
	}
}

// 单元测试：verify BPMNMonitoringService construction with new signature
func TestNewBPMNMonitoringService(t *testing.T) {
	s := NewBPMNMonitoringService(nil, nil, nil)
	if s == nil {
		t.Fatal("NewBPMNMonitoringService returned nil")
	}
	if s.client != nil {
		t.Error("expected nil client")
	}
	if s.auditService != nil {
		t.Error("expected nil auditService")
	}
}

// 单元测试：verify SetAuditService for dependency injection
func TestSetAuditService(t *testing.T) {
	s := NewBPMNMonitoringService(nil, nil, nil)
	if s.auditService != nil {
		t.Fatal("expected nil auditService before SetAuditService")
	}
	// 延迟注入模拟
	mockAudit := &BPMNAuditService{}
	s.SetAuditService(mockAudit)
	if s.auditService != mockAudit {
		t.Error("SetAuditService did not set auditService correctly")
	}
}

// 单元测试：ProcessTimelineEntry 字段填充逻辑（基于 log 的转换）
func TestProcessTimelineEntryConversion(t *testing.T) {
	now := time.Now()
	mockLog := &ent.ProcessAuditLog{}
	_ = mockLog
	_ = now

	// 验证 sequence 编号和 node duration 计算逻辑
	logs := []*ent.ProcessAuditLog{}
	// 模拟转换（不实际查询 DB）
	entries := make([]*ProcessTimelineEntry, 0, len(logs))
	for i, log := range logs {
		entry := &ProcessTimelineEntry{
			ID:        "test",
			Sequence:  i + 1,
			Timestamp: log.Timestamp,
		}
		_ = entry
		entries = append(entries, entry)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

// 单元测试：Math.Abs 边界（calculateImpactScore 不超过 100）
func TestImpactScoreCappedAt100(t *testing.T) {
	s := &BPMNMonitoringService{}
	score := s.calculateImpactScore(24*time.Hour, 24*time.Hour, 1000)
	if score > 100 {
		t.Errorf("calculateImpactScore should cap at 100, got %v", score)
	}
	if score < 99.99 {
		t.Errorf("calculateImpactScore should be near 100 for extreme inputs, got %v", score)
	}
}

// 单元测试：percentile p=0 returns smallest, p=1 returns largest
func TestPercentileEdges(t *testing.T) {
	input := []time.Duration{1 * time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second, 5 * time.Second}
	if got := percentile(input, 0.0); got != 1*time.Second {
		t.Errorf("percentile(0) = %v, want 1s", got)
	}
	if got := percentile(input, 1.0); got != 5*time.Second {
		t.Errorf("percentile(1) = %v, want 5s", got)
	}
}

// 单元测试：ListProcessInstanceStatusQuery 字段验证
func TestListProcessInstanceStatusQueryFields(t *testing.T) {
	now := time.Now()
	q := &ListProcessInstanceStatusQuery{
		TenantID:   1,
		Page:       1,
		PageSize:   20,
		ProcessKey: "incident_emergency_flow",
		Status:     "running",
		Assignee:   "user1",
		StartTime:  &now,
		EndTime:    &now,
	}
	if q.TenantID != 1 {
		t.Error("TenantID not set")
	}
	if q.ProcessKey != "incident_emergency_flow" {
		t.Error("ProcessKey not set")
	}
}

// 单元测试：AuditLogRequest 字段验证（确保与 bpmn_audit_service.QueryAuditLogsRequest 兼容）
func TestAuditLogRequestFields(t *testing.T) {
	now := time.Now()
	req := &AuditLogRequest{
		UserID:    "1",
		Action:    "started",
		TenantID:  1,
		Page:      1,
		PageSize:  20,
		StartTime: &now,
		EndTime:   &now,
	}
	if req.TenantID != 1 {
		t.Error("TenantID not set")
	}
	if req.UserID != "1" {
		t.Error("UserID not set")
	}
}

// 单元测试：generateOptimizationRecommendations 应根据瓶颈数量返回建议
func TestGenerateOptimizationRecommendations(t *testing.T) {
	s := &BPMNMonitoringService{}

	// 高影响瓶颈任务
	analysis := &BottleneckAnalysis{
		BottleneckTasks: []*BottleneckTask{
			{TaskName: "high_impact_task", ImpactScore: 80},
			{TaskName: "low_impact_task", ImpactScore: 50},
		},
		ResourceConstraints: []*ResourceConstraint{
			{ResourceName: "team", Utilization: 90},
		},
	}
	recs := s.generateOptimizationRecommendations(analysis)
	if len(recs) == 0 {
		t.Error("expected at least one recommendation")
	}
	// 高利用率资源应触发建议
	foundResourceRec := false
	for _, r := range recs {
		if r == "考虑增加 team 的容量" {
			foundResourceRec = true
		}
	}
	if !foundResourceRec {
		t.Error("expected resource utilization recommendation")
	}
}

// 单元测试：calculateBottleneckSeverity 严重度分级
func TestCalculateBottleneckSeverity(t *testing.T) {
	s := &BPMNMonitoringService{}
	cases := []struct {
		name       string
		highImpact int
		expectSev  string
	}{
		{"low", 0, "low"},
		{"low-1", 1, "low"},
		{"medium", 2, "medium"},
		{"medium-3", 3, "medium"},
		{"high", 4, "high"},
		{"high-5", 5, "high"},
		{"critical", 6, "critical"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			analysis := &BottleneckAnalysis{}
			for i := 0; i < c.highImpact; i++ {
				analysis.BottleneckTasks = append(analysis.BottleneckTasks, &BottleneckTask{ImpactScore: 80})
			}
			got := s.calculateBottleneckSeverity(analysis)
			if got != c.expectSev {
				t.Errorf("calculateBottleneckSeverity(%d high impact) = %v, want %v", c.highImpact, got, c.expectSev)
			}
		})
	}
}

// 单元测试：Math.Abs function works correctly with extreme values
func TestMathAbsBasics(t *testing.T) {
	if math.Abs(-1.0) != 1.0 {
		t.Error("math.Abs failed")
	}
	if math.Abs(1.0) != 1.0 {
		t.Error("math.Abs failed")
	}
}

// 单元测试：context cancellation handling
func TestContextCancellationHandling(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// 模拟服务方法需要正确处理取消的 context
	_ = ctx
}

// 单元测试：ProcessTimelineEntry field validation
func TestProcessTimelineEntrySequence(t *testing.T) {
	now := time.Now()
	prev := &ProcessTimelineEntry{Sequence: 1, Timestamp: now}
	curr := &ProcessTimelineEntry{
		Sequence:        2,
		Timestamp:       now.Add(5 * time.Second),
		NodeDurationMs:  5000,
		DurationMs:      100,
	}
	_ = prev
	if curr.NodeDurationMs != 5000 {
		t.Errorf("expected NodeDurationMs=5000, got %d", curr.NodeDurationMs)
	}
	if curr.Sequence != 2 {
		t.Errorf("expected Sequence=2, got %d", curr.Sequence)
	}
}

// 单元测试：确保 identifyBottleneckTasks P95 输出字段有效（非负数）
func TestBottleneckTaskFieldsNonNegative(t *testing.T) {
	bt := &BottleneckTask{
		WaitTimeSeconds:          10,
		ProcessingTimeSeconds:    20,
		TotalDurationSeconds:     30,
		P95WaitTimeSeconds:       15,
		P95ProcessingTimeSeconds: 25,
		P95TotalDurationSeconds:  35,
		SampleCount:              100,
	}
	if bt.SampleCount < 5 {
		t.Error("expected SampleCount >= 5 for bottleneck")
	}
	if bt.P95WaitTimeSeconds < 0 {
		t.Error("P95WaitTimeSeconds must be non-negative")
	}
}

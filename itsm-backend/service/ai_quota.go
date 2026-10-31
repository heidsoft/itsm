package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AIQuotaService 租户 token 配额与单价口径。
//
// 能力矩阵 AI 晋升条件之一：此前 token 用量只有计量（ai_llm_calls），
// 没有配额告警与单价口径——运营者无法回答「这个月花了多少、会不会爆」。
//
// 三条设计约束：
//
//  1. 数据源全部来自既有 ai_llm_calls 计量，不新建表：配额是「对既有计量
//     的新读法」，而不是一套并行的记账（两套记账必然漂移，谁也不知道哪个准）。
//  2. 未配置配额 = 不限制，但 metrics 如实报 quotaConfigured=false + unlimited：
//     不装作有限额，也不装作无限额——是「没配」。
//  3. 单价默认 0 并标注 currency/每千 token 口径，未配置时 estimatedCost=0 且
//     costConfigured=false——**绝不给一个猜的单价**：成本数字一旦展示给管理者，
//     就会被拿去做预算决策，猜错了比没有更糟。
type AIQuotaService struct {
	db *sql.DB

	mu          sync.RWMutex
	monthlyCap  int64 // 每租户每月 token 上限；<=0 = 未配置（不限制）
	warnRatio   float64
	pricePer1K  float64 // 每千 token 单价（元）
	currency    string
	costApplied bool // 单价是否已配置
}

// NewAIQuotaService 构造配额服务。配置来源（均可选）：
//
//	ITSM_AI_TOKEN_MONTHLY_QUOTA   租户月度 token 配额；未设置/解析失败 = 不限制
//	ITSM_AI_TOKEN_WARN_RATIO      预警阈值比例，默认 0.8，范围 (0,1]
//	ITSM_AI_TOKEN_PRICE_PER_1K    每千 token 单价；未设置 = 成本口径未配置
//	ITSM_AI_TOKEN_PRICE_CURRENCY  币种，默认 CNY
func NewAIQuotaService(db *sql.DB) *AIQuotaService {
	q := &AIQuotaService{
		db:          db,
		warnRatio:   0.8,
		currency:    "CNY",
		costApplied: false,
	}
	if v := os.Getenv("ITSM_AI_TOKEN_MONTHLY_QUOTA"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			q.monthlyCap = n
		}
	}
	if v := os.Getenv("ITSM_AI_TOKEN_WARN_RATIO"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 && f <= 1 {
			q.warnRatio = f
		}
	}
	if v := os.Getenv("ITSM_AI_TOKEN_PRICE_PER_1K"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			q.pricePer1K = f
			q.costApplied = true
		}
	}
	if v := os.Getenv("ITSM_AI_TOKEN_PRICE_CURRENCY"); v != "" {
		q.currency = strings.TrimSpace(v)
	}
	return q
}

// SetQuota 运行时更新配额（管理面/测试用）；cap<=0 表示解除限制。
func (s *AIQuotaService) SetQuota(cap int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.monthlyCap = cap
}

// QuotaSnapshot 当前配额配置快照。
type QuotaSnapshot struct {
	MonthlyCap     int64
	WarnRatio      float64
	PricePer1K     float64
	Currency       string
	CostConfigured bool
}

// Snapshot 返回当前配置（读锁）。
func (s *AIQuotaService) Snapshot() QuotaSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return QuotaSnapshot{
		MonthlyCap:     s.monthlyCap,
		WarnRatio:      s.warnRatio,
		PricePer1K:     s.pricePer1K,
		Currency:       s.currency,
		CostConfigured: s.costApplied,
	}
}

// monthWindow 本月自然月窗口（本地时区）。
// 注意与 kpiMonthWindows 同样的坑：先取本月 1 日，任何回退都从 1 日出发。
func monthWindow(now time.Time) (start, end time.Time) {
	start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	end = start.AddDate(0, 1, 0)
	return start, end
}

// monthTokens 本月已耗 token（tenant 维度，含失败调用——失败调用也花钱或至少
// 占用预算观察口径；按 success 过滤会让超限月份的数字偏小）。
func (s *AIQuotaService) monthTokens(ctx context.Context, tenantID int) (int64, error) {
	start, end := monthWindow(time.Now())
	const q = `SELECT COALESCE(SUM(tokens), 0) FROM ai_llm_calls WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3`
	var n int64
	if err := s.db.QueryRowContext(ctx, q, tenantID, start, end).Scan(&n); err != nil {
		return 0, fmt.Errorf("quota: sum month tokens: %w", err)
	}
	return n, nil
}

// QuotaUsage 配额用量评估结果。
type QuotaUsage struct {
	QuotaConfigured bool    `json:"quotaConfigured"`
	MonthlyQuota    int64   `json:"monthlyQuota"`
	TokensUsed      int64   `json:"tokensUsed"`
	TokensRemaining int64   `json:"tokensRemaining"`
	UsedRatio       float64 `json:"usedRatio"`
	// Alert: "" | "warning" | "exceeded"
	Alert string `json:"alert"`
	// AlertMessage 人类可读告警文案（Alert 非空时才有意义）。
	AlertMessage string `json:"alertMessage,omitempty"`
}

// Evaluate 评估租户本月用量：未配置配额时返回 quotaConfigured=false 的诚实空态。
func (s *AIQuotaService) Evaluate(ctx context.Context, tenantID int) (*QuotaUsage, error) {
	snap := s.Snapshot()
	used, err := s.monthTokens(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := &QuotaUsage{TokensUsed: used}
	if snap.MonthlyCap <= 0 {
		// 未配置：不限制，但也不装作有配额。
		return out, nil
	}
	out.QuotaConfigured = true
	out.MonthlyQuota = snap.MonthlyCap
	out.TokensRemaining = snap.MonthlyCap - used
	if out.TokensRemaining < 0 {
		out.TokensRemaining = 0
	}
	out.UsedRatio = float64(used) / float64(snap.MonthlyCap)
	switch {
	case used >= snap.MonthlyCap:
		out.Alert = "exceeded"
		out.AlertMessage = fmt.Sprintf("本月 token 用量 %d 已达配额 %d", used, snap.MonthlyCap)
	case out.UsedRatio >= snap.WarnRatio:
		out.Alert = "warning"
		out.AlertMessage = fmt.Sprintf("本月 token 用量 %d 已达配额 %d 的 %.0f%%", used, snap.MonthlyCap, out.UsedRatio*100)
	}
	return out, nil
}

// Check 检查本次调用是否放行（配额执行点）。
// 未配置配额恒放行；超限返回 false。调用方应把 false 映射为 429 而非 500。
func (s *AIQuotaService) Check(ctx context.Context, tenantID int) (bool, *QuotaUsage, error) {
	usage, err := s.Evaluate(ctx, tenantID)
	if err != nil {
		// 配额评估失败时放行（fail-open）：计量/DB 抖动不应挡业务调用。
		// 这里的取舍与权限判断相反——配额是预算保护不是安全边界，
		// fail-open 的代价（偶尔超一点）远小于 fail-closed（AI 全不可用）。
		return true, nil, err
	}
	if !usage.QuotaConfigured {
		return true, usage, nil
	}
	return usage.TokensUsed < usage.MonthlyQuota, usage, nil
}

// CostEstimate 按单价口径估算成本。
type CostEstimate struct {
	CostConfigured bool    `json:"costConfigured"`
	TotalTokens    int64   `json:"totalTokens"`
	EstimatedCost  float64 `json:"estimatedCost"`
	Currency       string  `json:"currency"`
	PricePer1K     float64 `json:"pricePer1k"`
}

// EstimateCost 按给定窗口的实际 token 用量估算成本。
// 单价未配置时返回 costConfigured=false 且 estimatedCost=0——宁可不给数字，
// 也不给一个猜的：成本数字会被拿去做预算决策。
func (s *AIQuotaService) EstimateCost(totalTokens int64) CostEstimate {
	snap := s.Snapshot()
	est := CostEstimate{TotalTokens: totalTokens, Currency: snap.Currency, PricePer1K: snap.PricePer1K}
	if !snap.CostConfigured {
		return est
	}
	est.CostConfigured = true
	est.EstimatedCost = float64(totalTokens) / 1000.0 * snap.PricePer1K
	// 保留 4 位小数，避免浮点尾数假装精度。
	est.EstimatedCost = math.Round(est.EstimatedCost*10000) / 10000
	return est
}

package knowledgeaccess

import (
	"testing"
	"time"

	"itsm-backend/ent"
)

// fixedNow 测试基准时间：2026-06-01 12:00:00 UTC
var fixedNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func ptr(t time.Time) *time.Time { return &t }

func newJudger(p FreshnessPolicy) *FreshnessJudger {
	return NewFreshnessJudger(p, func() time.Time { return fixedNow })
}

func TestFreshness_Judge_HardLifecycle(t *testing.T) {
	tests := []struct {
		name string
		f    FreshnessFields
		want FreshnessVerdict
	}{
		{
			name: "未声明任何时效字段_长期有效",
			f:    FreshnessFields{},
			want: FreshOK,
		},
		{
			name: "失效时间在未来_有效",
			f:    FreshnessFields{ValidUntil: ptr(fixedNow.Add(24 * time.Hour))},
			want: FreshOK,
		},
		{
			name: "失效时间恰好等于当前时刻_视为已失效",
			f:    FreshnessFields{ValidUntil: ptr(fixedNow)},
			want: FreshExpired,
		},
		{
			name: "失效时间已过_已失效",
			f:    FreshnessFields{ValidUntil: ptr(fixedNow.Add(-time.Second))},
			want: FreshExpired,
		},
		{
			name: "生效时间在未来_尚未生效",
			f:    FreshnessFields{ValidFrom: ptr(fixedNow.Add(time.Hour))},
			want: FreshNotYetEffective,
		},
		{
			name: "生效时间恰好等于当前时刻_视为已生效",
			f:    FreshnessFields{ValidFrom: ptr(fixedNow)},
			want: FreshOK,
		},
		{
			name: "生效时间已过_有效",
			f:    FreshnessFields{ValidFrom: ptr(fixedNow.Add(-24 * time.Hour))},
			want: FreshOK,
		},
		{
			name: "生效未到且已过失效期_优先判为已失效",
			f: FreshnessFields{
				ValidFrom:  ptr(fixedNow.Add(time.Hour)),
				ValidUntil: ptr(fixedNow.Add(-time.Hour)),
			},
			want: FreshExpired,
		},
		{
			name: "窗口期内_有效",
			f: FreshnessFields{
				ValidFrom:  ptr(fixedNow.Add(-24 * time.Hour)),
				ValidUntil: ptr(fixedNow.Add(24 * time.Hour)),
			},
			want: FreshOK,
		},
		{
			name: "未设复核周期时_即便从未复核也不判逾期",
			f:    FreshnessFields{ReviewIntervalDays: 0, LastReviewedAt: nil},
			want: FreshOK,
		},
		{
			name: "未设复核周期时_陈旧的上次复核时间不影响判定",
			f: FreshnessFields{
				ReviewIntervalDays: 0,
				LastReviewedAt:     ptr(fixedNow.AddDate(-5, 0, 0)),
			},
			want: FreshOK,
		},
	}

	j := newJudger(DefaultFreshnessPolicy())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := j.Judge(tt.f); got != tt.want {
				t.Fatalf("Judge() = %v(%s), want %v(%s)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestFreshness_Judge_ReviewOverdue(t *testing.T) {
	tests := []struct {
		name string
		f    FreshnessFields
		want FreshnessVerdict
	}{
		{
			name: "刚复核_有效",
			f: FreshnessFields{
				ReviewIntervalDays: 90,
				LastReviewedAt:     ptr(fixedNow.AddDate(0, 0, -10)),
			},
			want: FreshOK,
		},
		{
			name: "恰好在复核截止日_尚未逾期",
			f: FreshnessFields{
				ReviewIntervalDays: 90,
				LastReviewedAt:     ptr(fixedNow.AddDate(0, 0, -90)),
			},
			want: FreshOK,
		},
		{
			name: "超出复核周期一天_逾期",
			f: FreshnessFields{
				ReviewIntervalDays: 90,
				LastReviewedAt:     ptr(fixedNow.AddDate(0, 0, -91)),
			},
			want: FreshReviewOverdue,
		},
		{
			name: "声明了复核周期但从未复核_按逾期处理",
			f: FreshnessFields{
				ReviewIntervalDays: 90,
				LastReviewedAt:     nil,
			},
			want: FreshReviewOverdue,
		},
	}

	j := newJudger(DefaultFreshnessPolicy())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := j.Judge(tt.f); got != tt.want {
				t.Fatalf("Judge() = %v(%s), want %v(%s)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestFreshness_Judge_GracePeriod(t *testing.T) {
	p := DefaultFreshnessPolicy()
	p.ReviewGraceDays = 7
	j := newJudger(p)

	// 逾期 3 天，但宽限期 7 天 → 仍算有效
	within := FreshnessFields{
		ReviewIntervalDays: 90,
		LastReviewedAt:     ptr(fixedNow.AddDate(0, 0, -93)),
	}
	if got := j.Judge(within); got != FreshOK {
		t.Fatalf("宽限期内应为 FreshOK, got %v", got)
	}

	// 逾期 10 天，超出宽限期 → 逾期
	beyond := FreshnessFields{
		ReviewIntervalDays: 90,
		LastReviewedAt:     ptr(fixedNow.AddDate(0, 0, -100)),
	}
	if got := j.Judge(beyond); got != FreshReviewOverdue {
		t.Fatalf("超出宽限期应为 FreshReviewOverdue, got %v", got)
	}
}

func TestFreshness_Citable_RespectsPolicy(t *testing.T) {
	expired := FreshnessFields{ValidUntil: ptr(fixedNow.Add(-time.Hour))}
	notYet := FreshnessFields{ValidFrom: ptr(fixedNow.Add(time.Hour))}
	overdue := FreshnessFields{ReviewIntervalDays: 30, LastReviewedAt: ptr(fixedNow.AddDate(0, 0, -60))}

	t.Run("默认策略_三类逾期均不可引用", func(t *testing.T) {
		j := newJudger(DefaultFreshnessPolicy())
		for name, f := range map[string]FreshnessFields{
			"expired": expired, "not_yet": notYet, "overdue": overdue,
		} {
			if j.Citable(f) {
				t.Fatalf("%s 在默认策略下不应可引用", name)
			}
		}
	})

	t.Run("宽松策略_仅排除显式失效的文章", func(t *testing.T) {
		j := newJudger(PermissiveFreshnessPolicy())
		if got := j.Judge(overdue); got != FreshReviewOverdue {
			t.Fatalf("Judge 是事实判定，不应受策略影响, got %v", got)
		}
		if !j.Citable(overdue) {
			t.Fatal("宽松策略下复核逾期应仍可引用")
		}
		if j.Citable(expired) {
			t.Fatal("宽松策略下已失效文章仍不可引用")
		}
		if j.Citable(notYet) {
			t.Fatal("宽松策略下未生效文章仍不可引用")
		}
	})

	t.Run("全关闭策略_一律可引用", func(t *testing.T) {
		j := newJudger(FreshnessPolicy{})
		if !j.Citable(expired) || !j.Citable(notYet) || !j.Citable(overdue) {
			t.Fatal("策略全关闭时不应过滤任何文章")
		}
	})
}

func TestFreshness_CitableArticle_NilIsFailClosed(t *testing.T) {
	j := newJudger(DefaultFreshnessPolicy())
	if j.CitableArticle(nil) {
		t.Fatal("nil 文章必须按不可引用处理（fail-closed）")
	}
}

func TestFreshness_FieldsOf(t *testing.T) {
	if got := FieldsOf(nil); got != (FreshnessFields{}) {
		t.Fatalf("FieldsOf(nil) 应返回零值快照, got %+v", got)
	}

	vf := fixedNow.Add(-time.Hour)
	vu := fixedNow.Add(time.Hour)
	lr := fixedNow.AddDate(0, 0, -1)
	a := &ent.KnowledgeArticle{
		ValidFrom:          &vf,
		ValidUntil:         &vu,
		LastReviewedAt:     &lr,
		ReviewIntervalDays: 30,
	}
	got := FieldsOf(a)
	if got.ValidFrom != &vf || got.ValidUntil != &vu || got.LastReviewedAt != &lr {
		t.Fatal("FieldsOf 未正确透传时间字段指针")
	}
	if got.ReviewIntervalDays != 30 {
		t.Fatalf("FieldsOf 未正确读取复核周期, got %d", got.ReviewIntervalDays)
	}
}

func TestFreshness_FilterArticles_PreservesOrderAndCounts(t *testing.T) {
	j := newJudger(DefaultFreshnessPolicy())

	mk := func(id int, f FreshnessFields) *ent.KnowledgeArticle {
		a := &ent.KnowledgeArticle{ID: id}
		a.ValidFrom = f.ValidFrom
		a.ValidUntil = f.ValidUntil
		a.LastReviewedAt = f.LastReviewedAt
		a.ReviewIntervalDays = f.ReviewIntervalDays
		return a
	}

	in := []*ent.KnowledgeArticle{
		mk(1, FreshnessFields{}), // 保留
		mk(2, FreshnessFields{ValidUntil: ptr(fixedNow.Add(-time.Hour))}), // 剔除
		mk(3, FreshnessFields{}), // 保留
		mk(4, FreshnessFields{ReviewIntervalDays: 30, LastReviewedAt: ptr(fixedNow.AddDate(0, 0, -99))}), // 剔除
		mk(5, FreshnessFields{}), // 保留
	}

	kept, dropped := j.FilterArticles(in)
	if dropped != 2 {
		t.Fatalf("应剔除 2 篇, got %d", dropped)
	}
	if len(kept) != 3 {
		t.Fatalf("应保留 3 篇, got %d", len(kept))
	}
	wantIDs := []int{1, 3, 5}
	for i, a := range kept {
		if a.ID != wantIDs[i] {
			t.Fatalf("顺序或内容不符: idx=%d got id=%d want id=%d", i, a.ID, wantIDs[i])
		}
	}

	if k, d := j.FilterArticles(nil); len(k) != 0 || d != 0 {
		t.Fatal("空输入应原样返回且不计数")
	}
}

func TestFreshness_VerdictString(t *testing.T) {
	cases := map[FreshnessVerdict]string{
		FreshOK:              "ok",
		FreshNotYetEffective: "not_yet_effective",
		FreshExpired:         "expired",
		FreshReviewOverdue:   "review_overdue",
		FreshnessVerdict(99): "unknown",
	}
	for v, want := range cases {
		if got := v.String(); got != want {
			t.Fatalf("String() = %q, want %q", got, want)
		}
	}
	if !FreshOK.Citable() || FreshExpired.Citable() {
		t.Fatal("Citable() 语义错误")
	}
}

func TestFreshness_SQLPredicate_Composition(t *testing.T) {
	// 谓词无法脱离数据库执行，这里只验证不同策略组合均返回非 nil 谓词，
	// 避免调用方在极端配置下拿到 nil 触发 panic。
	policies := []struct {
		name string
		p    FreshnessPolicy
	}{
		{"默认", DefaultFreshnessPolicy()},
		{"宽松", PermissiveFreshnessPolicy()},
		{"全关闭", FreshnessPolicy{}},
		{"仅排除未生效", FreshnessPolicy{ExcludeNotYetEffective: true}},
		{"仅排除已失效", FreshnessPolicy{ExcludeExpired: true}},
	}
	for _, tt := range policies {
		t.Run(tt.name, func(t *testing.T) {
			j := newJudger(tt.p)
			if j.SQLPredicate() == nil {
				t.Fatal("SQLPredicate 不应返回 nil")
			}
		})
	}

	t.Run("后置过滤需求随策略变化", func(t *testing.T) {
		if !DefaultFreshnessPolicy().NeedsPostFilter() {
			t.Fatal("默认策略排除复核逾期，应需要后置过滤")
		}
		if PermissiveFreshnessPolicy().NeedsPostFilter() {
			t.Fatal("宽松策略不排除复核逾期，不应需要后置过滤")
		}
	})
}

func TestFreshness_ReviewIntervalOverflowGuard(t *testing.T) {
	p := DefaultFreshnessPolicy()
	p.ReviewGraceDays = 100 // 正常情况
	j := newJudger(p)

	f := FreshnessFields{
		ReviewIntervalDays: 30,
		LastReviewedAt:     ptr(fixedNow.AddDate(0, 0, -31)),
	}
	// 逾期 1 天，宽限 100 天 → 有效
	if got := j.Judge(f); got != FreshOK {
		t.Fatalf("宽限覆盖时应为 FreshOK, got %v", got)
	}
}

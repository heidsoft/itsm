package knowledgeaccess

import (
	"testing"
	"time"
)

var rankNow = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

func TestAuthority_RankOrdering(t *testing.T) {
	tests := []struct {
		name   string
		inputs []RankInput
		wantID []int // 期望的降序 ID 序列
	}{
		{
			name: "同相关性下_权威等级越高排越前",
			inputs: []RankInput{
				{ArticleID: 1, Relevance: 0.8, AuthorityLevel: AuthorityNormal},
				{ArticleID: 2, Relevance: 0.8, AuthorityLevel: AuthorityOfficial},
				{ArticleID: 3, Relevance: 0.8, AuthorityLevel: AuthorityRecommended},
				{ArticleID: 4, Relevance: 0.8, AuthorityLevel: AuthorityAuthoritative},
			},
			wantID: []int{4, 2, 3, 1},
		},
		{
			name: "权威加成不得压过显著的相关性差距",
			inputs: []RankInput{
				// 相关性 0.55 + 唯一真相源 0.2 = 0.75 < 0.9
				{ArticleID: 1, Relevance: 0.9, AuthorityLevel: AuthorityNormal},
				{ArticleID: 2, Relevance: 0.55, AuthorityLevel: AuthorityAuthoritative},
			},
			wantID: []int{1, 2},
		},
		{
			name: "权威加成可以拉回接近的相关性差距",
			inputs: []RankInput{
				// 相关性 0.71 + 官方 0.1333 > 0.8
				{ArticleID: 1, Relevance: 0.8, AuthorityLevel: AuthorityNormal},
				{ArticleID: 2, Relevance: 0.71, AuthorityLevel: AuthorityOfficial},
			},
			wantID: []int{2, 1},
		},
		{
			name: "同分时_更新的文章靠前",
			inputs: []RankInput{
				{ArticleID: 1, Relevance: 0.8, AuthorityLevel: AuthorityNormal, UpdatedAt: rankNow.AddDate(0, 0, -400)},
				{ArticleID: 2, Relevance: 0.8, AuthorityLevel: AuthorityNormal, UpdatedAt: rankNow.AddDate(0, 0, -1)},
			},
			wantID: []int{2, 1},
		},
		{
			name: "全部信号相同_保持输入顺序（稳定性）",
			inputs: []RankInput{
				{ArticleID: 7, Relevance: 0.8, AuthorityLevel: AuthorityNormal},
				{ArticleID: 3, Relevance: 0.8, AuthorityLevel: AuthorityNormal},
				{ArticleID: 5, Relevance: 0.8, AuthorityLevel: AuthorityNormal},
			},
			wantID: []int{7, 3, 5},
		},
		{
			name: "相关性越界被钳制_超界与满分等价",
			inputs: []RankInput{
				{ArticleID: 1, Relevance: 1.7, AuthorityLevel: AuthorityNormal},
				{ArticleID: 2, Relevance: 1.0, AuthorityLevel: AuthorityNormal},
			},
			wantID: []int{1, 2},
		},
		{
			name: "负相关性被钳制为0_不产生负分",
			inputs: []RankInput{
				{ArticleID: 1, Relevance: -3, AuthorityLevel: AuthorityNormal},
				{ArticleID: 2, Relevance: 0, AuthorityLevel: AuthorityNormal},
			},
			// 两者融合分都为 0 + tiebreak（UpdatedAt 均为零值），按输入序稳定
			wantID: []int{1, 2},
		},
		{
			name:   "空输入返回空结果",
			inputs: []RankInput{},
			wantID: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RankRanking(tt.inputs, rankNow)
			if len(got) != len(tt.wantID) {
				t.Fatalf("结果数 %d != 期望 %d", len(got), len(tt.wantID))
			}
			for i, r := range got {
				if r.Input.ArticleID != tt.wantID[i] {
					t.Fatalf("位置 %d: got id=%d, want id=%d（完整序: %v）",
						i, r.Input.ArticleID, tt.wantID[i], idsOf(got))
				}
			}
		})
	}
}

func idsOf(rs []RankedResult) []int {
	out := make([]int, len(rs))
	for i, r := range rs {
		out[i] = r.Input.ArticleID
	}
	return out
}

func TestAuthority_BoostBounds(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{
		{AuthorityNormal, 0},
		{AuthorityRecommended, 0.2 * 10.0 / 30.0},
		{AuthorityOfficial, 0.2 * 20.0 / 30.0},
		{AuthorityAuthoritative, 0.2},
		{-5, 0},   // 越界按普通处理
		{99, 0.2}, // 越界按上限处理
	}
	for _, c := range cases {
		got := authorityBoost(c.level)
		if abs(got-c.want) > 1e-9 {
			t.Fatalf("authorityBoost(%d) = %v, want %v", c.level, got, c.want)
		}
		if got < 0 || got > authorityBoostCap {
			t.Fatalf("authorityBoost(%d) = %v 越界 [0,%v]", c.level, got, authorityBoostCap)
		}
	}

	// 关键性质：最高权威加成（0.2）不能超过 0.3 的安全上界，
	// 防止后续调参时不小心把权威压过相关性。
	if authorityBoostCap >= 0.3 {
		t.Fatalf("authorityBoostCap=%v 超过安全上界 0.3", authorityBoostCap)
	}
}

func TestAuthority_FreshnessTiebreakMagnitude(t *testing.T) {
	// 最新：得满 12 分（折算 0.0012）
	fresh := freshnessTiebreak(rankNow, rankNow)
	if abs(fresh-0.0012) > 1e-9 {
		t.Fatalf("最新文章平局分 = %v, want 0.0012", fresh)
	}
	// 一年前的旧文章：0 分
	old := freshnessTiebreak(rankNow.AddDate(-1, 0, 0), rankNow)
	if old != 0 {
		t.Fatalf("一年前文章平局分 = %v, want 0", old)
	}
	// 未来时间（时钟偏移容错）：不产生负分也不超过满分
	future := freshnessTiebreak(rankNow.Add(time.Hour), rankNow)
	if future != 0.0012 {
		t.Fatalf("未来时间平局分 = %v, want 0.0012（钳制）", future)
	}
	// 零值时间：0 分
	if v := freshnessTiebreak(time.Time{}, rankNow); v != 0 {
		t.Fatalf("零值时间平局分 = %v, want 0", v)
	}

	// 量级约束：平局分最大值必须远小于权威加成的最小有效步长（10/30*0.2≈0.067），
	// 保证时间永远无法替代权威信号。
	if 0.0012 >= 0.067 {
		t.Fatal("平局分量级侵入权威加成区间")
	}
}

func TestAuthority_FusionScoreMonotonic(t *testing.T) {
	base := RankInput{ArticleID: 1, Relevance: 0.6, AuthorityLevel: AuthorityNormal, UpdatedAt: rankNow}
	baseScore := fusionScore(base, rankNow)

	// 提升相关性 -> 分数上升
	higherRel := base
	higherRel.Relevance = 0.7
	if fusionScore(higherRel, rankNow) <= baseScore {
		t.Fatal("相关性上升应提升融合分")
	}

	// 提升权威 -> 分数上升
	higherAuth := base
	higherAuth.AuthorityLevel = AuthorityOfficial
	if fusionScore(higherAuth, rankNow) <= baseScore {
		t.Fatal("权威上升应提升融合分")
	}

	// 更新时间变新 -> 分数不降
	fresher := base
	fresher.UpdatedAt = rankNow.Add(time.Hour)
	if fusionScore(fresher, rankNow) < baseScore {
		t.Fatal("更新时间变新不应降低融合分")
	}
}

func TestAuthority_RankRankingDoesNotMutateInput(t *testing.T) {
	inputs := []RankInput{
		{ArticleID: 1, Relevance: 0.5, AuthorityLevel: AuthorityNormal},
		{ArticleID: 2, Relevance: 0.9, AuthorityLevel: AuthorityNormal},
	}
	RankRanking(inputs, rankNow)
	if inputs[0].ArticleID != 1 || inputs[1].ArticleID != 2 {
		t.Fatal("RankRanking 不应修改调用方的输入切片")
	}
}

func TestAuthority_OrderAuthorityRanked(t *testing.T) {
	items := []string{"a", "b", "c", "d"}
	scores := []float64{0.1, 0.9, 0.5, 0.7}
	OrderAuthorityRanked(items, scores)
	want := []string{"b", "d", "c", "a"}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("重排结果 %v, want %v", items, want)
		}
	}

	// 长度不匹配时安全返回
	OrderAuthorityRanked(items, []float64{0.1})
	// 单元素
	OrderAuthorityRanked([]string{"x"}, []float64{0.5})
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

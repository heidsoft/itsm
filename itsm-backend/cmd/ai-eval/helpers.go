package main

// 本文件承载 cmd/ai-eval 的可测试辅助：JSONL 解析、ROUGE、AUC、特征映射。
//
// 它们必须在非 main 文件里才能被单元测试引用；同时保持极简（无外部依赖），
// 避免验收命令本身成为新的故障面。

import (
	"itsm-backend/ai/eval"
	"itsm-backend/service"
)

// auc 计算 ROC AUC（秩和法，ties 取平均秩）。
func auc(scores []float64, labels []bool) float64 {
	n := len(scores)
	type pair struct {
		s float64
		y bool
	}
	pairs := make([]pair, n)
	for i := range scores {
		pairs[i] = pair{scores[i], labels[i]}
	}
	// 简单排序（稳定即可）
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if pairs[j].s < pairs[i].s {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	pos, neg := 0, 0
	for _, p := range pairs {
		if p.y {
			pos++
		} else {
			neg++
		}
	}
	if pos == 0 || neg == 0 {
		return 1.0
	}
	// 正样本的秩和（1-based，ties 平均）
	rankSum := 0.0
	for i := 0; i < n; {
		j := i
		for j+1 < n && pairs[j+1].s == pairs[i].s {
			j++
		}
		avgRank := float64(i+j+2) / 2
		for k := i; k <= j; k++ {
			if pairs[k].y {
				rankSum += avgRank
			}
		}
		i = j + 1
	}
	u := rankSum - float64(pos)*(float64(pos)+1)/2
	return u / (float64(pos) * float64(neg))
}

// predictFeatures 把 JSON 特征映射为预测器输入（与 ai/eval 同口径）。
func predictFeatures(c eval.PredictionCase) service.SLABreachFeatures {
	return service.SLABreachFeatures{
		Priority:                     strField(c.Features["priority"]),
		HorizonMinutes:               c.HorizonMinutes,
		CreatedHour:                  int(flField(c.Features["created_hour"])),
		Weekday:                      int(flField(c.Features["weekday"])),
		HistoricalAvgResolutionHours: flField(c.Features["historical_avg_resolution_hours"]),
		SimilarTicketsLast30d:        int(flField(c.Features["similar_tickets_last_30d"])),
	}
}

func strField(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func flField(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		return 0
	}
}

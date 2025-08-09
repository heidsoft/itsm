package service

import "context"

// SummarizeService provides simple summarization stub; wire to LLM later
type SummarizeService struct{}

func NewSummarizeService() *SummarizeService { return &SummarizeService{} }

func (s *SummarizeService) Summarize(ctx context.Context, text string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 200
	}
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

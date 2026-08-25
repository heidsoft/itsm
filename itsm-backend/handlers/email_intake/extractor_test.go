package email_intake

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"itsm-backend/service"
)

type fakeEmailLLM struct {
	output string
	err    error
}

type sequenceEmailLLM struct {
	outputs []string
	index   int
}

type flakyEmailLLM struct {
	calls int
}

func (f *flakyEmailLLM) Chat(context.Context, string, []service.LLMMessage) (string, error) {
	f.calls++
	if f.calls == 1 {
		return "", context.DeadlineExceeded
	}
	return `{"intent":"report_incident","customerName":"客户A","reportedContractNumber":"SUP-RETRY","title":"线路中断","description":"down","impact":"high","urgency":"high","missingFields":[],"confidence":0.95}`, nil
}

func (f *sequenceEmailLLM) Chat(context.Context, string, []service.LLMMessage) (string, error) {
	output := f.outputs[f.index]
	if f.index < len(f.outputs)-1 {
		f.index++
	}
	return output, nil
}

func (f fakeEmailLLM) Chat(context.Context, string, []service.LLMMessage) (string, error) {
	return f.output, f.err
}

func TestEmailIntakeExtractorValidatesStructuredOutput(t *testing.T) {
	extractor := NewEmailIntakeExtractor(fakeEmailLLM{output: `{"intent":"report_incident","customerName":"上海ABC有限公司","reportedContractNumber":"SUP-1","title":"线路中断","description":"MPLS down","impact":"high","urgency":"high","missingFields":[],"confidence":0.96}`}, "test-model")
	result, raw, err := extractor.Extract(context.Background(), "报障", "邮件正文")
	require.NoError(t, err)
	require.Contains(t, raw, "report_incident")
	require.Equal(t, "上海ABC有限公司", result.CustomerName)
	require.Equal(t, 0.96, result.Confidence)
}

func TestEmailIntakeExtractorRejectsInvalidEnumsAndConfidence(t *testing.T) {
	extractor := NewEmailIntakeExtractor(fakeEmailLLM{output: `{"intent":"delete_database","customerName":"A","reportedContractNumber":"1","title":"x","description":"x","impact":"extreme","urgency":"high","confidence":2}`}, "test-model")
	_, _, err := extractor.Extract(context.Background(), "x", "x")
	require.Error(t, err)
}

func TestEmailIntakeExtractorRejectsTrailingOutput(t *testing.T) {
	extractor := NewEmailIntakeExtractor(fakeEmailLLM{output: `{"intent":"other","confidence":0.9} {"unexpected":true}`}, "test-model")
	_, _, err := extractor.Extract(context.Background(), "x", "x")
	require.Error(t, err)
}

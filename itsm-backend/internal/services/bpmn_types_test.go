package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBPMNStartEvent_Interface(t *testing.T) {
	e := &BPMNStartEvent{
		ID:         "start1",
		Name:       "开始",
		EventType:  "message",
		MessageRef: "msg1",
		TimerRef:   "timer1",
		SignalRef:  "sig1",
	}
	assert.Equal(t, "start1", e.GetID())
	assert.Equal(t, "开始", e.GetName())
	assert.Equal(t, "StartEvent", e.GetType())
}

func TestBPMNBoundaryEvent_Interface(t *testing.T) {
	e := &BPMNBoundaryEvent{
		ID:             "boundary1",
		Name:           "边界",
		AttachedToRef:  "task1",
		CancelActivity: true,
	}
	assert.Equal(t, "boundary1", e.GetID())
	assert.Equal(t, "边界", e.GetName())
	assert.Equal(t, "BoundaryEvent", e.GetType())
}

func TestBPMNIntermediateEvent_Interface(t *testing.T) {
	e := &BPMNIntermediateEvent{
		ID:        "intermediate1",
		Name:      "中间事件",
		EventType: "timer",
		TimerRef:  "timer1",
	}
	assert.Equal(t, "intermediate1", e.GetID())
	assert.Equal(t, "中间事件", e.GetName())
	assert.Equal(t, "IntermediateEvent", e.GetType())
}

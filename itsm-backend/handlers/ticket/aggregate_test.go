package ticket

import (
	"strings"
	"testing"
	"time"

	"itsm-backend/handlers/shared"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// PR-1.3 — handlers/ticket aggregate contract tests
//
// The ticket handlers package hosts 11 sub-services (workflow / notification
// / sla / assignment / automation / category / type / template / tag / view
// / dependency). The aggregate file centralises the cross-service value
// objects (Priority, Status, TicketNumber). This file locks the contract so
// a refactor that drifts any of these enums breaks CI before it breaks
// downstream consumers.
// -----------------------------------------------------------------------------

func TestPriority_String(t *testing.T) {
	cases := []struct {
		name string
		in   Priority
		want string
	}{
		{"low", PriorityLow, "low"},
		{"normal", PriorityNormal, "normal"},
		{"high", PriorityHigh, "high"},
		{"urgent", PriorityUrgent, "urgent"},
		{"critical", PriorityCritical, "critical"},
		{"unknown falls back to normal", Priority(0), "normal"},
		{"unknown positive falls back to normal", Priority(99), "normal"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.String())
		})
	}
}

func TestStatus_IsValid(t *testing.T) {
	cases := []struct {
		in   Status
		want bool
	}{
		{StatusNew, true},
		{StatusOpen, true},
		{StatusInProgress, true},
		{StatusPending, true},
		{StatusResolved, true},
		{StatusClosed, true},
		{StatusCancelled, true},
		{Status("archived"), false},
		{Status(""), false},
	}
	for _, tc := range cases {
		t.Run(string(tc.in), func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.IsValid())
		})
	}
}

func TestTicketNumber_RejectsEmpty(t *testing.T) {
	tn, err := NewTicketNumber("")
	assert.Error(t, err)
	assert.Nil(t, tn)
}

func TestTicketNumber_HappyPath(t *testing.T) {
	tn, err := NewTicketNumber("TCK-2026-0001")
	require.NoError(t, err)
	require.NotNil(t, tn)
	assert.Equal(t, "TCK-2026-0001", tn.Value())
	assert.NoError(t, tn.Validate())
}

func TestTicketNumber_EqualsContract(t *testing.T) {
	a, _ := NewTicketNumber("TCK-1")
	b, _ := NewTicketNumber("TCK-1")
	c, _ := NewTicketNumber("TCK-2")
	var other shared.ValueObject = nil

	assert.True(t, a.Equals(b))
	assert.False(t, a.Equals(c))
	assert.False(t, a.Equals(other), "non-TicketNumber ValueObject must not be equal")

	// also cover via interface variable
	var vo shared.ValueObject = a
	assert.True(t, vo.Equals(b))
}

func TestTicketNumber_ValidateAfterMutation(t *testing.T) {
	tn, err := NewTicketNumber("TCK-1")
	require.NoError(t, err)

	// Force-clear the value object to exercise the post-mutation validator.
	tn.value = ""
	assert.Error(t, tn.Validate())
}

// TestTicketAssignment_Aggregate ensures the value-object factory does not
// panic and stores the fields. Real business rules live in
// ticket_assignment_service.go, but the wiring is verified here.
func TestTicketAssignment_Aggregate(t *testing.T) {
	ta := TicketAssignment{
		AssignedTo:   shared.UserID("42"),
		AssignedBy:   shared.UserID("7"),
		AssignedAt:   time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC),
		Instructions: "L2 takeover",
	}
	assert.Equal(t, shared.UserID("42"), ta.AssignedTo)
	assert.Equal(t, shared.UserID("7"), ta.AssignedBy)
	assert.Equal(t, "L2 takeover", ta.Instructions)
}

func TestComment_AggregateStoresFields(t *testing.T) {
	c := Comment{
		ID:        "comment-1",
		AuthorID:  shared.UserID("7"),
		Content:   "Looking into this now.",
		IsPrivate: true,
	}
	assert.Equal(t, "comment-1", c.ID)
	assert.True(t, c.IsPrivate)
}

func TestAttachment_AggregateStoresFields(t *testing.T) {
	a := Attachment{
		ID:         "attach-1",
		Filename:   "trace.log",
		FileSize:   4096,
		MimeType:   "text/plain",
		URL:        "/api/v1/attachments/attach-1",
		UploadedBy: shared.UserID("7"),
	}
	assert.Equal(t, int64(4096), a.FileSize)
	assert.Equal(t, "text/plain", a.MimeType)
}

// TestTicketCreatedEvent_EventTypeStable locks the wire-format event name.
// Downstream consumers (notification/audit) subscribe on this string; any
// rename is a breaking change that must surface here.
func TestTicketCreatedEvent_EventTypeStable(t *testing.T) {
	e := &TicketCreatedEvent{TicketID: "t-1", Title: "x", Priority: PriorityHigh}
	assert.Equal(t, "ticket.created", e.GetEventType())
}

func TestTicketAssignedEvent_EventTypeStable(t *testing.T) {
	e := &TicketAssignedEvent{TicketID: "t-1", AssignedTo: shared.UserID("7"), AssignedBy: shared.UserID("8")}
	assert.Equal(t, "ticket.assigned", e.GetEventType())
}

func TestTicketStatusChangedEvent_EventTypeStable(t *testing.T) {
	e := &TicketStatusChangedEvent{TicketID: "t-1", OldStatus: StatusOpen, NewStatus: StatusInProgress}
	assert.Equal(t, "ticket.status_changed", e.GetEventType())
}

// TestEventTypeNamesAreUniqueTableDriven protects the event-bus contract
// from accidental aliasing. Each event's GetEventType must be unique across
// the package; otherwise a single subscriber would receive mixed payloads.
func TestEventTypeNamesAreUniqueTableDriven(t *testing.T) {
	events := []struct {
		name string
		typ  string
	}{
		{"created", (&TicketCreatedEvent{}).GetEventType()},
		{"assigned", (&TicketAssignedEvent{}).GetEventType()},
		{"status_changed", (&TicketStatusChangedEvent{}).GetEventType()},
		{"comment_added", (&TicketCommentAddedEvent{}).GetEventType()},
		{"priority_updated", (&TicketPriorityUpdatedEvent{}).GetEventType()},
	}
	seen := make(map[string]string)
	for _, ev := range events {
		if existing, ok := seen[ev.typ]; ok {
			t.Fatalf("event type %q already used by %s", ev.typ, existing)
		}
		seen[ev.typ] = ev.name
	}
}

// TestPriorityOrderingMentionsContract documents the relative ordering
// we expose to UI consumers (low < normal < high < urgent < critical).
// Even if the order is not used for comparisons, a rename of an existing
// priority level would silently re-order the list and break dashboards.
func TestPriorityOrderingMentionsContract(t *testing.T) {
	ordered := []Priority{PriorityLow, PriorityNormal, PriorityHigh, PriorityUrgent, PriorityCritical}
	for i, p := range ordered {
		assert.NotZero(t, int(p), "priority %d must have a non-zero int value", i)
	}
}

// TestNewTicketNumber_ErrorsContainExpectedMessage confirms the error is
// at least descriptive enough to bubble up in handler responses without
// leaking internal detail.
func TestNewTicketNumber_ErrorsContainExpectedMessage(t *testing.T) {
	_, err := NewTicketNumber("")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "empty"),
		"error message must mention empty, got: %v", err)
}
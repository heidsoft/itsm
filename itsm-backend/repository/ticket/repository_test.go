package ticket

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"
	"itsm-backend/repository/base"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type stubSequenceProvider struct {
	values []int64
	index  int
}

func (s *stubSequenceProvider) GetNextSequenceWithExpiry(_ context.Context, _ string, _ time.Time) (int64, error) {
	if s.index >= len(s.values) {
		return 0, fmt.Errorf("no more sequence values")
	}
	value := s.values[s.index]
	s.index++
	return value, nil
}

func TestRepository_SetSequenceService_TypedNilUsesDatabaseFallback(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	repo := fx.repo.(*EntRepository)
	var unavailableSequenceService *stubSequenceProvider
	repo.SetSequenceService(unavailableSequenceService)
	require.Nil(t, repo.sequenceService)

	tkt, err := repo.Create(fx.ctx, &CreateParams{
		Title:       "Database fallback ticket",
		Description: "Redis is unavailable",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, tkt.TicketNumber)
}

// repoFixture sets up an in-memory SQLite repo for testing.
type repoFixture struct {
	ctx    context.Context
	client *ent.Client
	repo   Repository
	tenant *ent.Tenant
	user   *ent.User
}

func newRepoFixture(t *testing.T) *repoFixture {
	t.Helper()
	ctx := context.Background()
	client := enttest.Open(t, "sqlite3", "file:repo_test?mode=memory&cache=shared&_fk=1")
	logger := zaptest.NewLogger(t).Sugar()
	repo := NewEntRepository(client, logger)

	tenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("test").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	user, err := client.User.Create().
		SetUsername("alice").
		SetEmail("alice@test.com").
		SetName("Alice").
		SetPasswordHash("hash").
		SetRole("end_user").
		SetActive(true).
		SetTenantID(tenant.ID).
		Save(ctx)
	require.NoError(t, err)

	return &repoFixture{ctx: ctx, client: client, repo: repo, tenant: tenant, user: user}
}

// =====================================================================
// Create
// =====================================================================

func TestRepository_Create(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	params := &CreateParams{
		Title:       "Test Ticket",
		Description: "Description",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}

	tkt, err := fx.repo.Create(fx.ctx, params, fx.tenant.ID)
	require.NoError(t, err)
	assert.NotZero(t, tkt.ID)
	assert.Equal(t, "Test Ticket", tkt.Title)
	assert.Equal(t, StatusNew, tkt.Status)
}

func TestRepository_Create_GeneratesTicketNumber(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	params := &CreateParams{
		Title:       "Number Test",
		Description: "",
		Priority:    PriorityLow,
		Type:        TypeServiceRequest,
		RequesterID: fx.user.ID,
	}

	tkt, err := fx.repo.Create(fx.ctx, params, fx.tenant.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, tkt.TicketNumber)
	assert.Contains(t, tkt.TicketNumber, "TKT-")
}

func TestRepository_Create_RetriesOnTicketNumberConflict(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	concreteRepo, ok := fx.repo.(*EntRepository)
	require.True(t, ok)

	now := time.Now()
	existingNumber := fmt.Sprintf("TKT-%04d%02d-%06d", now.Year(), int(now.Month()), 1)
	_, err := fx.client.Ticket.Create().
		SetTitle("Existing Ticket").
		SetDescription("Existing").
		SetType(string(TypeIncident)).
		SetPriority(string(PriorityLow)).
		SetTicketNumber(existingNumber).
		SetRequesterID(fx.user.ID).
		SetTenantID(fx.tenant.ID).
		SetStatus(string(StatusNew)).
		Save(fx.ctx)
	require.NoError(t, err)

	concreteRepo.SetSequenceService(&stubSequenceProvider{values: []int64{1, 2}})

	tkt, err := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Retry Create",
		Description: "Conflict then retry",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)
	require.NoError(t, err)
	assert.NotEqual(t, existingNumber, tkt.TicketNumber)
	assert.Equal(t, fmt.Sprintf("TKT-%04d%02d-%06d", now.Year(), int(now.Month()), 2), tkt.TicketNumber)
}

func TestRepository_Create_DifferentTenants(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	params := &CreateParams{
		Title:       "Cross Tenant",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}

	tkt1, err := fx.repo.Create(fx.ctx, params, fx.tenant.ID)
	require.NoError(t, err)

	// Tenant2 should not see tenant1's ticket
	_, err = fx.repo.GetByID(fx.ctx, tkt1.ID, 99999)
	assert.Error(t, err)
}

// =====================================================================
// GetByID
// =====================================================================

func TestRepository_GetByID(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Get Test",
		Description: "Desc",
		Priority:    PriorityHigh,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	found, err := fx.repo.GetByID(fx.ctx, created.ID, fx.tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, "Get Test", found.Title)
}

func TestRepository_GetByID_NotFound(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	_, err := fx.repo.GetByID(fx.ctx, 99999, fx.tenant.ID)
	assert.Error(t, err)
}

func TestRepository_GetByID_WrongTenant(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Tenant Isolation",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	_, err := fx.repo.GetByID(fx.ctx, created.ID, 99999)
	assert.Error(t, err)
}

// =====================================================================
// GetByNumber
// =====================================================================

func TestRepository_GetByNumber(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "By Number",
		Description: "",
		Priority:    PriorityLow,
		Type:        TypeServiceRequest,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	found, err := fx.repo.GetByNumber(fx.ctx, created.TicketNumber, fx.tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)
	assert.Equal(t, created.TicketNumber, found.TicketNumber)
}

func TestRepository_GetByNumber_NotFound(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	_, err := fx.repo.GetByNumber(fx.ctx, "TKT-DOES-NOT-EXIST", fx.tenant.ID)
	assert.Error(t, err)
}

func TestRepository_GetByNumber_WrongTenant(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "By Number Tenant",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	_, err := fx.repo.GetByNumber(fx.ctx, created.TicketNumber, 99999)
	assert.Error(t, err)
}

// =====================================================================
// Update
// =====================================================================

func TestRepository_Update(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Update Me",
		Description: "Original",
		Priority:    PriorityLow,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	title := "Updated Title"
	desc := "Updated Description"

	updated, err := fx.repo.Update(fx.ctx, created.ID, &UpdateParams{
		Title:       &title,
		Description: &desc,
		Priority:    func() *Priority { p := PriorityCritical; return &p }(),
		Version:     created.Version,
	}, fx.tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", updated.Title)
	assert.Equal(t, "Updated Description", updated.Description)
	assert.Equal(t, PriorityCritical, updated.Priority)
}

func TestRepository_Update_NotFound(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	title := "Should Fail"
	_, err := fx.repo.Update(fx.ctx, 99999, &UpdateParams{
		Title: &title,
	}, fx.tenant.ID)
	assert.Error(t, err)
}

func TestRepository_Update_RejectsStaleVersion(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()
	created, err := fx.repo.Create(fx.ctx, &CreateParams{
		Title: "Original", Priority: PriorityMedium, Type: TypeIncident, RequesterID: fx.user.ID,
	}, fx.tenant.ID)
	require.NoError(t, err)
	title := "Must not overwrite"
	_, err = fx.repo.Update(fx.ctx, created.ID, &UpdateParams{Title: &title, Version: created.Version + 1}, fx.tenant.ID)
	require.ErrorContains(t, err, "version conflict")
	unchanged, err := fx.repo.GetByID(fx.ctx, created.ID, fx.tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "Original", unchanged.Title)
}

// =====================================================================
// Delete
// =====================================================================

func TestRepository_Delete(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Delete Me",
		Description: "",
		Priority:    PriorityLow,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	err := fx.repo.Delete(fx.ctx, created.ID, fx.tenant.ID)
	require.NoError(t, err)

	_, err = fx.repo.GetByID(fx.ctx, created.ID, fx.tenant.ID)
	assert.Error(t, err)
}

func TestRepository_Delete_WrongTenant(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Delete Isolated",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	// Wrong tenant delete: implementation may return nil (no rows matched) or error.
	// The key invariant is the ticket still exists for the correct tenant.
	_ = fx.repo.Delete(fx.ctx, created.ID, 99999)

	_, err := fx.repo.GetByID(fx.ctx, created.ID, fx.tenant.ID)
	require.NoError(t, err)
}

// =====================================================================
// List
// =====================================================================

func TestRepository_List(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	for i := 0; i < 5; i++ {
		fx.repo.Create(fx.ctx, &CreateParams{
			Title:       "List Ticket",
			Description: "",
			Priority:    PriorityMedium,
			Type:        TypeIncident,
			RequesterID: fx.user.ID,
		}, fx.tenant.ID)
	}

	result, err := fx.repo.List(fx.ctx, fx.tenant.ID, &FilterParams{}, &base.QueryParams{})
	require.NoError(t, err)
	assert.Equal(t, 5, result.Total)
	assert.Len(t, result.Data, 5)
}

func TestRepository_List_Pagination(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	for i := 0; i < 10; i++ {
		fx.repo.Create(fx.ctx, &CreateParams{
			Title:       "Page Ticket",
			Description: "",
			Priority:    PriorityLow,
			Type:        TypeIncident,
			RequesterID: fx.user.ID,
		}, fx.tenant.ID)
	}

	result, err := fx.repo.List(fx.ctx, fx.tenant.ID, &FilterParams{}, &base.QueryParams{Page: 1, PageSize: 3})
	require.NoError(t, err)
	assert.Equal(t, 10, result.Total)
	assert.Len(t, result.Data, 3)
}

func TestRepository_List_TenantIsolation(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Tenant1 Ticket",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	result, err := fx.repo.List(fx.ctx, 99999, &FilterParams{}, &base.QueryParams{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Len(t, result.Data, 0)
}

func TestRepository_List_ParentTypeAndOverdueFilters(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()
	parent, err := fx.repo.Create(fx.ctx, &CreateParams{
		Title: "Parent", Priority: PriorityMedium, Type: TypeIncident, RequesterID: fx.user.ID,
	}, fx.tenant.ID)
	require.NoError(t, err)
	overdue, err := fx.repo.Create(fx.ctx, &CreateParams{
		Title: "Overdue problem", Priority: PriorityHigh, Type: TypeProblem, RequesterID: fx.user.ID, ParentTicketID: &parent.ID,
	}, fx.tenant.ID)
	require.NoError(t, err)
	resolved, err := fx.repo.Create(fx.ctx, &CreateParams{
		Title: "Resolved problem", Priority: PriorityHigh, Type: TypeProblem, RequesterID: fx.user.ID, ParentTicketID: &parent.ID,
	}, fx.tenant.ID)
	require.NoError(t, err)
	past := time.Now().Add(-time.Hour)
	_, err = fx.client.Ticket.UpdateOneID(overdue.ID).SetSLAResolutionDeadline(past).Save(fx.ctx)
	require.NoError(t, err)
	_, err = fx.client.Ticket.UpdateOneID(resolved.ID).SetSLAResolutionDeadline(past).SetStatus(string(StatusResolved)).Save(fx.ctx)
	require.NoError(t, err)

	problemType := TypeProblem
	result, err := fx.repo.List(fx.ctx, fx.tenant.ID, &FilterParams{
		Type: &problemType, ParentTicketID: &parent.ID, IsOverdue: true,
	}, &base.QueryParams{})
	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, overdue.ID, result.Data[0].ID)
}

// =====================================================================
// BatchDelete
// =====================================================================

func TestRepository_BatchDelete(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	ids := make([]int, 0, 3)
	for i := 0; i < 3; i++ {
		tkt, _ := fx.repo.Create(fx.ctx, &CreateParams{
			Title:       "Batch Delete",
			Description: "",
			Priority:    PriorityLow,
			Type:        TypeIncident,
			RequesterID: fx.user.ID,
		}, fx.tenant.ID)
		ids = append(ids, tkt.ID)
	}

	err := fx.repo.BatchDelete(fx.ctx, ids, fx.tenant.ID)
	require.NoError(t, err)

	for _, id := range ids {
		_, err := fx.repo.GetByID(fx.ctx, id, fx.tenant.ID)
		assert.Error(t, err)
	}
}

func TestRepository_BatchDelete_EmptyList(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	err := fx.repo.BatchDelete(fx.ctx, []int{}, fx.tenant.ID)
	assert.NoError(t, err)
}

func TestRepository_BatchDelete_TenantIsolation(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Batch Tenant",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	_ = fx.repo.BatchDelete(fx.ctx, []int{created.ID}, 99999)

	_, err := fx.repo.GetByID(fx.ctx, created.ID, fx.tenant.ID)
	require.NoError(t, err)
}

// =====================================================================
// Exists
// =====================================================================

func TestRepository_Exists(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Exists Check",
		Description: "",
		Priority:    PriorityLow,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	exists, err := fx.repo.Exists(fx.ctx, created.ID, fx.tenant.ID)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = fx.repo.Exists(fx.ctx, 99999, fx.tenant.ID)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestRepository_Exists_WrongTenant(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Exists Tenant",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	exists, err := fx.repo.Exists(fx.ctx, created.ID, 99999)
	require.NoError(t, err)
	assert.False(t, exists)
}

// =====================================================================
// UpdateStatus
// =====================================================================

func TestRepository_UpdateStatus(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Status Update",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	updated, err := fx.repo.UpdateStatus(fx.ctx, created.ID, StatusOpen, fx.tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusOpen, updated.Status)
}

func TestRepository_UpdateStatus_NotFound(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	_, err := fx.repo.UpdateStatus(fx.ctx, 99999, StatusOpen, fx.tenant.ID)
	assert.Error(t, err)
}

// =====================================================================
// AssignTicket
// =====================================================================

func TestRepository_AssignTicket(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	created, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Assign Test",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	updated, err := fx.repo.AssignTicket(fx.ctx, created.ID, fx.user.ID, fx.tenant.ID)
	require.NoError(t, err)
	assert.NotNil(t, updated.AssigneeID)
	assert.Equal(t, fx.user.ID, *updated.AssigneeID)
}

func TestRepository_AssignTicket_NotFound(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	_, err := fx.repo.AssignTicket(fx.ctx, 99999, fx.user.ID, fx.tenant.ID)
	assert.Error(t, err)
}

// =====================================================================
// CountByStatus / CountByPriority
// =====================================================================

func TestRepository_CountByStatus(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	for i := 0; i < 3; i++ {
		tkt, _ := fx.repo.Create(fx.ctx, &CreateParams{
			Title:       "Count Status",
			Description: "",
			Priority:    PriorityMedium,
			Type:        TypeIncident,
			RequesterID: fx.user.ID,
		}, fx.tenant.ID)
		fx.repo.UpdateStatus(fx.ctx, tkt.ID, StatusOpen, fx.tenant.ID)
	}

	fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Count Status New",
		Description: "",
		Priority:    PriorityLow,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	counts, err := fx.repo.CountByStatus(fx.ctx, fx.tenant.ID)
	require.NoError(t, err)
	assert.Contains(t, counts, StatusNew)
	assert.Contains(t, counts, StatusOpen)
}

func TestRepository_CountByPriority(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Priority Count",
		Description: "",
		Priority:    PriorityCritical,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	counts, err := fx.repo.CountByPriority(fx.ctx, fx.tenant.ID)
	require.NoError(t, err)
	assert.Contains(t, counts, PriorityCritical)
}

// =====================================================================
// GenerateTicketNumber
// =====================================================================

func TestRepository_GenerateTicketNumber(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	num, err := fx.repo.GenerateTicketNumber(fx.ctx, fx.tenant.ID)
	require.NoError(t, err)
	assert.Contains(t, num, "TKT-")
	assert.NotEmpty(t, num)
}

func TestRepository_GenerateTicketNumber_Uniqueness(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	seen := make(map[string]struct{})
	prev := ""
	// GenerateTicketNumber is count-based: each call without creating a ticket
	// returns the same counter value. To exercise uniqueness, create a ticket
	// between calls so the count increments.
	for i := 0; i < 20; i++ {
		_, _ = fx.repo.Create(fx.ctx, &CreateParams{
			Title:       "Num Uniq",
			Description: "",
			Priority:    PriorityLow,
			Type:        TypeIncident,
			RequesterID: fx.user.ID,
		}, fx.tenant.ID)

		num, err := fx.repo.GenerateTicketNumber(fx.ctx, fx.tenant.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, num)
		assert.Contains(t, num, "TKT-")
		if prev != "" {
			assert.NotEqual(t, prev, num, "numbers should differ after creating a ticket")
		}
		prev = num
		_, exists := seen[num]
		assert.False(t, exists, "generated ticket number should be unique: %s", num)
		seen[num] = struct{}{}
	}
}

// =====================================================================
// FindByAssignee / FindByRequester
// =====================================================================

func TestRepository_FindByAssignee(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	tkt, _ := fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Assign Find",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)
	fx.repo.AssignTicket(fx.ctx, tkt.ID, fx.user.ID, fx.tenant.ID)

	tickets, err := fx.repo.FindByAssignee(fx.ctx, fx.user.ID, fx.tenant.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tickets), 1)
}

func TestRepository_FindByRequester(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Requester Find",
		Description: "",
		Priority:    PriorityLow,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	tickets, err := fx.repo.FindByRequester(fx.ctx, fx.user.ID, fx.tenant.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tickets), 1)
}

func TestRepository_FindByRequester_WrongTenant(t *testing.T) {
	fx := newRepoFixture(t)
	defer fx.client.Close()

	fx.repo.Create(fx.ctx, &CreateParams{
		Title:       "Requester Tenant",
		Description: "",
		Priority:    PriorityMedium,
		Type:        TypeIncident,
		RequesterID: fx.user.ID,
	}, fx.tenant.ID)

	tickets, err := fx.repo.FindByRequester(fx.ctx, fx.user.ID, 99999)
	require.NoError(t, err)
	assert.Empty(t, tickets)
}

// =====================================================================
// State Machine (domain model)
// =====================================================================

func TestTicketModel_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from     Status
		to       Status
		expected bool
	}{
		{StatusNew, StatusOpen, true},
		{StatusNew, StatusCancelled, true},
		{StatusNew, StatusResolved, false},
		{StatusOpen, StatusInProgress, true},
		{StatusOpen, StatusPending, true},
		{StatusOpen, StatusResolved, true},
		{StatusInProgress, StatusPending, true},
		{StatusInProgress, StatusResolved, true},
		{StatusResolved, StatusClosed, true},
		{StatusResolved, StatusOpen, true}, // Reopen
		{StatusClosed, StatusOpen, false},  // Cannot reopen from closed
		{StatusCancelled, StatusOpen, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + "_to_" + string(tt.to)
		t.Run(name, func(t *testing.T) {
			model := &Ticket{Status: tt.from}
			result := model.CanTransitionTo(tt.to)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTicketModel_IsFinalState(t *testing.T) {
	assert.True(t, (&Ticket{Status: StatusClosed}).IsFinalState())
	assert.True(t, (&Ticket{Status: StatusCancelled}).IsFinalState())
	assert.False(t, (&Ticket{Status: StatusNew}).IsFinalState())
	assert.False(t, (&Ticket{Status: StatusOpen}).IsFinalState())
	assert.False(t, (&Ticket{Status: StatusResolved}).IsFinalState())
}

func TestTicketModel_StateError(t *testing.T) {
	err := &StateError{CurrentStatus: StatusNew, Message: "cannot resolve ticket from current status"}
	assert.Contains(t, err.Error(), "cannot resolve ticket")
}

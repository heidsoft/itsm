package email_intake

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"itsm-backend/ent"
	"itsm-backend/ent/customerbranch"
	"itsm-backend/ent/externalcontractreference"
	"itsm-backend/ent/group"
	"itsm-backend/ent/oncallschedule"
	"itsm-backend/ent/oncallshift"
	"itsm-backend/ent/servicecustomer"
	"itsm-backend/ent/sourceorganization"
	"itsm-backend/ent/supportcontract"
	"itsm-backend/ent/user"
)

const (
	ResolutionVerified      = "VERIFIED"
	ResolutionNeedInfo      = "NEED_INFORMATION"
	ResolutionAmbiguous     = "AMBIGUOUS"
	ResolutionRejected      = "REJECTED"
	ResolutionManualReview  = "MANUAL_REVIEW"
	DefaultConfidenceCutoff = 0.80
)

var nonWord = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func NormalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)))
}

func NormalizeContractNumber(value string) string {
	return nonWord.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "")
}

type IntakeFields struct {
	SourceOrganizationName string     `json:"sourceOrganizationName"`
	CustomerName           string     `json:"customerName"`
	BranchName             string     `json:"branchName"`
	ReportedContractNumber string     `json:"reportedContractNumber"`
	Title                  string     `json:"title"`
	Description            string     `json:"description"`
	OccurredAt             *time.Time `json:"occurredAt,omitempty"`
	Impact                 string     `json:"impact"`
	Urgency                string     `json:"urgency"`
	Confidence             float64    `json:"confidence"`
}

type Resolution struct {
	Status               string   `json:"status"`
	CustomerID           int      `json:"customerId,omitempty"`
	BranchID             int      `json:"branchId,omitempty"`
	SupportContractID    int      `json:"supportContractId,omitempty"`
	SourceOrganizationID int      `json:"sourceOrganizationId,omitempty"`
	MissingFields        []string `json:"missingFields"`
	Reasons              []string `json:"reasons"`
}

type Resolver struct{ client *ent.Client }

func NewResolver(client *ent.Client) *Resolver { return &Resolver{client: client} }

func (r *Resolver) Resolve(ctx context.Context, tenantID int, fields IntakeFields) (Resolution, error) {
	result := Resolution{}
	if strings.TrimSpace(fields.CustomerName) == "" {
		result.MissingFields = append(result.MissingFields, "customerName")
	}
	if strings.TrimSpace(fields.ReportedContractNumber) == "" {
		result.MissingFields = append(result.MissingFields, "reportedContractNumber")
	}
	if len(result.MissingFields) > 0 {
		result.Status = ResolutionNeedInfo
		return result, nil
	}

	customers, err := r.client.ServiceCustomer.Query().
		Where(servicecustomer.TenantIDEQ(tenantID), servicecustomer.StatusEQ("active")).All(ctx)
	if err != nil {
		return result, fmt.Errorf("query service customers: %w", err)
	}
	needle := NormalizeName(fields.CustomerName)
	matches := make([]*ent.ServiceCustomer, 0, 1)
	for _, candidate := range customers {
		if NormalizeName(candidate.Name) == needle || NormalizeName(candidate.ShortName) == needle || containsNormalized(candidate.Aliases, needle) || containsNormalized(candidate.HistoricalNames, needle) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 0 {
		result.Status = ResolutionManualReview
		result.Reasons = []string{"customer_not_exactly_matched"}
		return result, nil
	}
	if len(matches) > 1 {
		result.Status = ResolutionAmbiguous
		result.Reasons = []string{"multiple_customer_matches"}
		return result, nil
	}
	result.CustomerID = matches[0].ID

	if strings.TrimSpace(fields.SourceOrganizationName) != "" {
		sources, sourceErr := r.client.SourceOrganization.Query().Where(
			sourceorganization.TenantIDEQ(tenantID),
			sourceorganization.NormalizedNameEQ(NormalizeName(fields.SourceOrganizationName)),
			sourceorganization.StatusEQ("active"),
		).All(ctx)
		if sourceErr != nil {
			return result, fmt.Errorf("query source organizations: %w", sourceErr)
		}
		if len(sources) == 1 {
			result.SourceOrganizationID = sources[0].ID
		} else {
			result.Status = ResolutionManualReview
			result.Reasons = []string{"source_organization_not_verified"}
			return result, nil
		}
	}

	branches, err := r.client.CustomerBranch.Query().Where(
		customerbranch.TenantIDEQ(tenantID), customerbranch.CustomerIDEQ(result.CustomerID), customerbranch.StatusEQ("active"),
	).All(ctx)
	if err != nil {
		return result, fmt.Errorf("query customer branches: %w", err)
	}
	if len(branches) > 0 {
		branchNeedle := NormalizeName(fields.BranchName)
		if branchNeedle == "" && len(branches) > 1 {
			result.Status = ResolutionAmbiguous
			result.MissingFields = []string{"branchName"}
			result.Reasons = []string{"multiple_active_branches"}
			return result, nil
		}
		if branchNeedle == "" {
			result.BranchID = branches[0].ID
		} else {
			for _, branch := range branches {
				if NormalizeName(branch.Name) == branchNeedle || containsNormalized(branch.Aliases, branchNeedle) {
					if result.BranchID != 0 {
						result.Status = ResolutionAmbiguous
						result.Reasons = []string{"multiple_branch_matches"}
						return result, nil
					}
					result.BranchID = branch.ID
				}
			}
			if result.BranchID == 0 {
				result.Status = ResolutionManualReview
				result.Reasons = []string{"branch_not_exactly_matched"}
				return result, nil
			}
		}
	}

	contractNumber := NormalizeContractNumber(fields.ReportedContractNumber)
	contracts, err := r.client.SupportContract.Query().Where(
		supportcontract.TenantIDEQ(tenantID), supportcontract.NormalizedContractNumberEQ(contractNumber),
	).All(ctx)
	if err != nil {
		return result, fmt.Errorf("query support contracts: %w", err)
	}
	if len(contracts) == 0 && result.SourceOrganizationID != 0 {
		references, refErr := r.client.ExternalContractReference.Query().Where(
			externalcontractreference.TenantIDEQ(tenantID),
			externalcontractreference.SourceOrganizationIDEQ(result.SourceOrganizationID),
			externalcontractreference.NormalizedExternalContractNumberEQ(contractNumber),
		).WithSupportContract().All(ctx)
		if refErr != nil {
			return result, fmt.Errorf("query external contract references: %w", refErr)
		}
		for _, reference := range references {
			if reference.Edges.SupportContract != nil {
				contracts = append(contracts, reference.Edges.SupportContract)
			}
		}
	}
	if len(contracts) != 1 {
		result.Status = ResolutionManualReview
		result.Reasons = []string{"contract_not_uniquely_matched"}
		return result, nil
	}
	contract := contracts[0]
	if contract.CustomerID != result.CustomerID || (contract.BranchID != nil && *contract.BranchID != result.BranchID) {
		result.Status = ResolutionRejected
		result.Reasons = []string{"contract_customer_or_branch_mismatch"}
		return result, nil
	}
	result.SupportContractID = contract.ID
	if contract.Status != "active" {
		result.Status = ResolutionRejected
		result.Reasons = []string{"contract_not_active"}
		return result, nil
	}
	now := time.Now()
	if (contract.StartAt != nil && contract.StartAt.After(now)) || (contract.EndAt != nil && !contract.EndAt.After(now)) {
		result.Status = ResolutionRejected
		result.Reasons = []string{"contract_outside_effective_period"}
		return result, nil
	}
	result.Status = ResolutionVerified
	return result, nil
}

func containsNormalized(values []string, needle string) bool {
	for _, value := range values {
		if NormalizeName(value) == needle {
			return true
		}
	}
	return false
}

var (
	ErrOverlappingShift = errors.New("on-call shift overlaps an existing shift")
	ErrInvalidShift     = errors.New("on-call shift end must be after start")
	ErrShiftNotFound    = errors.New("on-call shift not found")
	ErrNoOnCall         = errors.New("no on-call resolver found for group")
)

type CurrentOnCall struct {
	ScheduleID int       `json:"scheduleId"`
	ShiftID    int       `json:"shiftId"`
	GroupID    int       `json:"groupId"`
	UserID     int       `json:"userId"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
}

type OnCallService struct{ client *ent.Client }

func NewOnCallService(client *ent.Client) *OnCallService { return &OnCallService{client: client} }

func (s *OnCallService) CreateShift(ctx context.Context, tenantID, scheduleID, userID int, startAt, endAt time.Time) (*ent.OnCallShift, error) {
	if !endAt.After(startAt) {
		return nil, ErrInvalidShift
	}
	schedule, err := s.client.OnCallSchedule.Query().Where(
		oncallschedule.IDEQ(scheduleID), oncallschedule.TenantIDEQ(tenantID), oncallschedule.StatusEQ("active"),
	).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("on-call schedule not found: %w", err)
	}
	member, err := s.client.Group.Query().Where(
		group.IDEQ(schedule.GroupID), group.TenantIDEQ(tenantID), group.HasMembersWith(user.IDEQ(userID), user.TenantIDEQ(tenantID), user.ActiveEQ(true)),
	).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("validate on-call member: %w", err)
	}
	if !member {
		return nil, errors.New("on-call user must be an active member of the support group")
	}
	overlap, err := s.client.OnCallShift.Query().Where(
		oncallshift.TenantIDEQ(tenantID), oncallshift.ScheduleIDEQ(scheduleID),
		oncallshift.StartAtLT(endAt), oncallshift.EndAtGT(startAt),
	).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check overlapping on-call shifts: %w", err)
	}
	if overlap {
		return nil, ErrOverlappingShift
	}
	return s.client.OnCallShift.Create().SetTenantID(tenantID).SetScheduleID(scheduleID).SetUserID(userID).SetStartAt(startAt).SetEndAt(endAt).Save(ctx)
}

func (s *OnCallService) ListShifts(ctx context.Context, tenantID, scheduleID int) ([]*ent.OnCallShift, error) {
	query := s.client.OnCallShift.Query().Where(oncallshift.TenantIDEQ(tenantID))
	if scheduleID > 0 {
		query.Where(oncallshift.ScheduleIDEQ(scheduleID))
	}
	return query.Order(ent.Desc(oncallshift.FieldStartAt)).All(ctx)
}

func (s *OnCallService) UpdateShift(ctx context.Context, tenantID, shiftID, scheduleID, userID int, startAt, endAt time.Time) (*ent.OnCallShift, error) {
	if !endAt.After(startAt) {
		return nil, ErrInvalidShift
	}
	shift, err := s.client.OnCallShift.Query().Where(oncallshift.IDEQ(shiftID), oncallshift.TenantIDEQ(tenantID)).Only(ctx)
	if err != nil {
		return nil, ErrShiftNotFound
	}
	if scheduleID != shift.ScheduleID || userID != shift.UserID || !startAt.Equal(shift.StartAt) || !endAt.Equal(shift.EndAt) {
		overlap, err := s.client.OnCallShift.Query().Where(
			oncallshift.TenantIDEQ(tenantID), oncallshift.ScheduleIDEQ(scheduleID),
			oncallshift.StartAtLT(endAt), oncallshift.EndAtGT(startAt),
			oncallshift.IDNEQ(shiftID),
		).Exist(ctx)
		if err != nil {
			return nil, fmt.Errorf("check overlapping on-call shifts: %w", err)
		}
		if overlap {
			return nil, ErrOverlappingShift
		}
	}
	return s.client.OnCallShift.UpdateOneID(shiftID).SetScheduleID(scheduleID).SetUserID(userID).SetStartAt(startAt).SetEndAt(endAt).Save(ctx)
}

func (s *OnCallService) DeleteShift(ctx context.Context, tenantID, shiftID int) error {
	deleted, err := s.client.OnCallShift.Delete().Where(oncallshift.IDEQ(shiftID), oncallshift.TenantIDEQ(tenantID)).Exec(ctx)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrShiftNotFound
	}
	return nil
}

func (s *OnCallService) CurrentResolver(ctx context.Context, tenantID, groupID int, at time.Time) (*CurrentOnCall, error) {
	shifts, err := s.client.OnCallShift.Query().Where(
		oncallshift.TenantIDEQ(tenantID), oncallshift.StartAtLTE(at), oncallshift.EndAtGT(at),
		oncallshift.HasScheduleWith(oncallschedule.TenantIDEQ(tenantID), oncallschedule.GroupIDEQ(groupID), oncallschedule.StatusEQ("active")),
		oncallshift.HasUserWith(user.TenantIDEQ(tenantID), user.ActiveEQ(true)),
	).WithSchedule().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query current on-call resolver: %w", err)
	}
	if len(shifts) == 0 {
		return nil, ErrNoOnCall
	}
	sort.Slice(shifts, func(i, j int) bool { return shifts[i].StartAt.Before(shifts[j].StartAt) })
	shift := shifts[0]
	return &CurrentOnCall{ScheduleID: shift.ScheduleID, ShiftID: shift.ID, GroupID: groupID, UserID: shift.UserID, StartAt: shift.StartAt, EndAt: shift.EndAt}, nil
}

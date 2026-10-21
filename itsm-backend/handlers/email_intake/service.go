package email_intake

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"itsm-backend/ent"
	"itsm-backend/ent/customerbranch"
	"itsm-backend/ent/emailconversation"
	"itsm-backend/ent/externalcontractreference"
	"itsm-backend/ent/group"
	oncall "itsm-backend/handlers/on_call"
	"itsm-backend/ent/servicecustomer"
	"itsm-backend/ent/sourceorganization"
	"itsm-backend/ent/supportcontract"
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

// Service is the main service struct for email_intake domain
type Service struct {
	client *ent.Client
	onCall *OnCallService
}

func NewService(client *ent.Client) *Service {
	return &Service{client: client, onCall: NewOnCallService(client)}
}

type conversationIdentity struct {
	id       int
	tenantID int
}

func updatedIdentity(updated *ent.EmailConversation) conversationIdentity {
	if updated == nil {
		return conversationIdentity{}
	}
	return conversationIdentity{id: updated.ID, tenantID: updated.TenantID}
}

func isPersistenceNotFound(err error) bool {
	return ent.IsNotFound(err)
}

func (s *Service) CreateShift(ctx context.Context, tenantID, scheduleID, userID int, startAt, endAt time.Time) (*ent.OnCallShift, error) {
	return s.onCall.CreateShift(ctx, tenantID, scheduleID, userID, startAt, endAt)
}

func (s *Service) ListShifts(ctx context.Context, tenantID, scheduleID int) ([]*ent.OnCallShift, error) {
	return s.onCall.ListShifts(ctx, tenantID, scheduleID, nil, nil)
}

func (s *Service) UpdateShift(ctx context.Context, tenantID, shiftID, scheduleID, userID int, startAt, endAt time.Time) (*ent.OnCallShift, error) {
	return s.onCall.UpdateShift(ctx, tenantID, shiftID, scheduleID, userID, startAt, endAt)
}

func (s *Service) DeleteShift(ctx context.Context, tenantID, shiftID int) error {
	return s.onCall.DeleteShift(ctx, tenantID, shiftID)
}

func (s *Service) CurrentOnCall(ctx context.Context, tenantID, groupID int, at time.Time) (*CurrentOnCall, error) {
	return s.onCall.CurrentResolver(ctx, tenantID, groupID, at)
}

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
	var contracts []*ent.SupportContract
	if result.SourceOrganizationID != 0 {
		references, refErr := r.client.ExternalContractReference.Query().Where(
			externalcontractreference.TenantIDEQ(tenantID),
			externalcontractreference.SourceOrganizationIDEQ(result.SourceOrganizationID),
			externalcontractreference.NormalizedExternalContractNumberEQ(contractNumber),
		).WithSupportContract(func(q *ent.SupportContractQuery) {
			q.Where(supportcontract.TenantIDEQ(tenantID))
		}).All(ctx)
		if refErr != nil {
			return result, fmt.Errorf("query external contract references: %w", refErr)
		}
		for _, reference := range references {
			if reference.Edges.SupportContract != nil {
				contracts = append(contracts, reference.Edges.SupportContract)
			}
		}
	} else {
		var contractErr error
		contracts, contractErr = r.client.SupportContract.Query().Where(
			supportcontract.TenantIDEQ(tenantID), supportcontract.NormalizedContractNumberEQ(contractNumber),
		).All(ctx)
		if contractErr != nil {
			return result, fmt.Errorf("query support contracts: %w", contractErr)
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

// 值班排班已迁至 handlers/on_call：排班与班次是被多个域消费的一等领域，
// 不该寄生在邮件受理里。此处保留类型/错误别名与构造函数，让 orchestrator
// 与既有 /email-intake/on-call 路由的引用面保持不变。
//
// 规范入口是 /api/v1/on-call/*（handlers/on_call）；/email-intake/on-call/*
// 作为历史路径保留，二者共用同一个 on_call.Service 实例，不存在两套语义。
type OnCallService = oncall.Service
type CurrentOnCall = oncall.CurrentOnCall

var (
	ErrOverlappingShift = oncall.ErrOverlappingShift
	ErrInvalidShift     = oncall.ErrInvalidShift
	ErrShiftNotFound    = oncall.ErrShiftNotFound
	ErrNoOnCall         = oncall.ErrNoOnCall
)

func NewOnCallService(client *ent.Client) *oncall.Service { return oncall.NewService(client) }

// CreateOnCallSchedule / ListOnCallSchedules 是历史路径的薄委托。
// 排班表的完整 CRUD（含改名、停用、删除）由 /on-call/schedules 提供。
func (s *Service) CreateOnCallSchedule(ctx context.Context, tenantID int, req *scheduleRequest) (*ent.OnCallSchedule, error) {
	return s.onCall.CreateSchedule(ctx, tenantID, oncall.ScheduleInput{
		GroupID: req.GroupID, Name: req.Name, Timezone: req.Timezone, Status: req.Status,
	})
}

func (s *Service) ListOnCallSchedules(ctx context.Context, tenantID int) ([]*ent.OnCallSchedule, error) {
	return s.onCall.ListSchedules(ctx, tenantID, 0)
}

// ─── ServiceCustomer ────────────────────────────────────────────────

func (s *Service) CreateCustomer(ctx context.Context, tenantID int, req *customerRequest) (*ent.ServiceCustomer, error) {
	status := defaultString(req.Status, "active")
	return s.client.ServiceCustomer.Create().SetTenantID(tenantID).SetName(strings.TrimSpace(req.Name)).
		SetNormalizedName(NormalizeName(req.Name)).SetShortName(req.ShortName).SetAliases(req.Aliases).
		SetHistoricalNames(req.HistoricalNames).SetStatus(status).SetNillableLinkedCustomerTenantID(req.LinkedCustomerTenantID).Save(ctx)
}

func (s *Service) ListCustomers(ctx context.Context, tenantID int) ([]*ent.ServiceCustomer, error) {
	return s.client.ServiceCustomer.Query().Where(servicecustomer.TenantIDEQ(tenantID)).Order(ent.Desc(servicecustomer.FieldUpdatedAt)).All(ctx)
}

func (s *Service) UpdateCustomer(ctx context.Context, tenantID, id int, req *customerRequest) (*ent.ServiceCustomer, error) {
	return s.client.ServiceCustomer.UpdateOneID(id).Where(servicecustomer.TenantIDEQ(tenantID)).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetShortName(req.ShortName).SetAliases(req.Aliases).SetHistoricalNames(req.HistoricalNames).SetStatus(defaultString(req.Status, "active")).SetNillableLinkedCustomerTenantID(req.LinkedCustomerTenantID).Save(ctx)
}

func (s *Service) DisableCustomer(ctx context.Context, tenantID, id int) (*ent.ServiceCustomer, error) {
	return s.client.ServiceCustomer.UpdateOneID(id).Where(servicecustomer.TenantIDEQ(tenantID)).SetStatus("inactive").Save(ctx)
}

func (s *Service) CustomerExists(ctx context.Context, tenantID, id int) (bool, error) {
	return s.client.ServiceCustomer.Query().Where(servicecustomer.IDEQ(id), servicecustomer.TenantIDEQ(tenantID)).Exist(ctx)
}

// ─── CustomerBranch ─────────────────────────────────────────────────

func (s *Service) CreateBranch(ctx context.Context, tenantID int, req *branchRequest) (*ent.CustomerBranch, error) {
	status := defaultString(req.Status, "active")
	return s.client.CustomerBranch.Create().SetTenantID(tenantID).SetCustomerID(req.CustomerID).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetAliases(req.Aliases).SetStatus(status).Save(ctx)
}

func (s *Service) ListBranches(ctx context.Context, tenantID, customerID int) ([]*ent.CustomerBranch, error) {
	query := s.client.CustomerBranch.Query().Where(customerbranch.TenantIDEQ(tenantID))
	if customerID > 0 {
		query.Where(customerbranch.CustomerIDEQ(customerID))
	}
	return query.All(ctx)
}

func (s *Service) UpdateBranch(ctx context.Context, tenantID, id int, req *branchRequest) (*ent.CustomerBranch, error) {
	return s.client.CustomerBranch.UpdateOneID(id).Where(customerbranch.TenantIDEQ(tenantID)).SetCustomerID(req.CustomerID).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetAliases(req.Aliases).SetStatus(defaultString(req.Status, "active")).Save(ctx)
}

func (s *Service) DisableBranch(ctx context.Context, tenantID, id int) (*ent.CustomerBranch, error) {
	return s.client.CustomerBranch.UpdateOneID(id).Where(customerbranch.TenantIDEQ(tenantID)).SetStatus("inactive").Save(ctx)
}

func (s *Service) BranchExistsForCustomer(ctx context.Context, tenantID, branchID, customerID int) (bool, error) {
	return s.client.CustomerBranch.Query().Where(customerbranch.IDEQ(branchID), customerbranch.CustomerIDEQ(customerID), customerbranch.TenantIDEQ(tenantID)).Exist(ctx)
}

// ─── SourceOrganization ─────────────────────────────────────────────

func (s *Service) CreateSourceOrganization(ctx context.Context, tenantID int, req *sourceOrganizationRequest) (*ent.SourceOrganization, error) {
	status := defaultString(req.Status, "active")
	return s.client.SourceOrganization.Create().SetTenantID(tenantID).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetEmailAddresses(req.EmailAddresses).SetEmailDomains(req.EmailDomains).SetStatus(status).Save(ctx)
}

func (s *Service) ListSourceOrganizations(ctx context.Context, tenantID int) ([]*ent.SourceOrganization, error) {
	return s.client.SourceOrganization.Query().Where(sourceorganization.TenantIDEQ(tenantID)).All(ctx)
}

func (s *Service) UpdateSourceOrganization(ctx context.Context, tenantID, id int, req *sourceOrganizationRequest) (*ent.SourceOrganization, error) {
	return s.client.SourceOrganization.UpdateOneID(id).Where(sourceorganization.TenantIDEQ(tenantID)).SetName(strings.TrimSpace(req.Name)).SetNormalizedName(NormalizeName(req.Name)).SetEmailAddresses(req.EmailAddresses).SetEmailDomains(req.EmailDomains).SetStatus(defaultString(req.Status, "active")).Save(ctx)
}

func (s *Service) DisableSourceOrganization(ctx context.Context, tenantID, id int) (*ent.SourceOrganization, error) {
	return s.client.SourceOrganization.UpdateOneID(id).Where(sourceorganization.TenantIDEQ(tenantID)).SetStatus("inactive").Save(ctx)
}

// ─── SupportContract ────────────────────────────────────────────────

func (s *Service) CreateSupportContract(ctx context.Context, tenantID int, req *supportContractRequest) (*ent.SupportContract, error) {
	status := defaultString(req.Status, "active")
	return s.client.SupportContract.Create().SetTenantID(tenantID).SetCustomerID(req.CustomerID).SetNillableBranchID(req.BranchID).SetContractNumber(strings.TrimSpace(req.ContractNumber)).SetNormalizedContractNumber(NormalizeContractNumber(req.ContractNumber)).SetStatus(status).SetNillableStartAt(req.StartAt).SetNillableEndAt(req.EndAt).Save(ctx)
}

func (s *Service) ListSupportContracts(ctx context.Context, tenantID int) ([]*ent.SupportContract, error) {
	return s.client.SupportContract.Query().Where(supportcontract.TenantIDEQ(tenantID)).All(ctx)
}

func (s *Service) UpdateSupportContract(ctx context.Context, tenantID, id int, req *supportContractRequest) (*ent.SupportContract, error) {
	return s.client.SupportContract.UpdateOneID(id).Where(supportcontract.TenantIDEQ(tenantID)).SetCustomerID(req.CustomerID).SetNillableBranchID(req.BranchID).SetContractNumber(strings.TrimSpace(req.ContractNumber)).SetNormalizedContractNumber(NormalizeContractNumber(req.ContractNumber)).SetStatus(defaultString(req.Status, "active")).SetNillableStartAt(req.StartAt).SetNillableEndAt(req.EndAt).Save(ctx)
}

func (s *Service) TerminateSupportContract(ctx context.Context, tenantID, id int) (*ent.SupportContract, error) {
	return s.client.SupportContract.UpdateOneID(id).Where(supportcontract.TenantIDEQ(tenantID)).SetStatus("terminated").Save(ctx)
}

func (s *Service) SupportContractExists(ctx context.Context, tenantID, id int) (bool, error) {
	return s.client.SupportContract.Query().Where(supportcontract.IDEQ(id), supportcontract.TenantIDEQ(tenantID)).Exist(ctx)
}

// ─── ExternalContractReference ──────────────────────────────────────

func (s *Service) CreateExternalContractReference(ctx context.Context, tenantID int, req *externalReferenceRequest) (*ent.ExternalContractReference, error) {
	contract, err := s.client.SupportContract.Query().Where(supportcontract.IDEQ(req.SupportContractID), supportcontract.TenantIDEQ(tenantID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return s.client.ExternalContractReference.Create().SetTenantID(tenantID).SetSourceOrganizationID(req.SourceOrganizationID).SetSupportContractID(contract.ID).SetCustomerID(contract.CustomerID).SetNillableBranchID(contract.BranchID).SetExternalContractNumber(strings.TrimSpace(req.ExternalContractNumber)).SetNormalizedExternalContractNumber(NormalizeContractNumber(req.ExternalContractNumber)).Save(ctx)
}

func (s *Service) ListExternalContractReferences(ctx context.Context, tenantID int) ([]*ent.ExternalContractReference, error) {
	return s.client.ExternalContractReference.Query().Where(externalcontractreference.TenantIDEQ(tenantID)).All(ctx)
}

func (s *Service) UpdateExternalContractReference(ctx context.Context, tenantID, id int, req *externalReferenceRequest) (*ent.ExternalContractReference, error) {
	contract, err := s.client.SupportContract.Query().Where(supportcontract.IDEQ(req.SupportContractID), supportcontract.TenantIDEQ(tenantID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return s.client.ExternalContractReference.UpdateOneID(id).Where(externalcontractreference.TenantIDEQ(tenantID)).SetSourceOrganizationID(req.SourceOrganizationID).SetSupportContractID(req.SupportContractID).SetCustomerID(contract.CustomerID).SetNillableBranchID(contract.BranchID).SetExternalContractNumber(strings.TrimSpace(req.ExternalContractNumber)).SetNormalizedExternalContractNumber(NormalizeContractNumber(req.ExternalContractNumber)).Save(ctx)
}

func (s *Service) DeleteExternalContractReference(ctx context.Context, tenantID, id int) (int, error) {
	return s.client.ExternalContractReference.Delete().Where(externalcontractreference.IDEQ(id), externalcontractreference.TenantIDEQ(tenantID)).Exec(ctx)
}

// ─── SourceOrganization existence (shared) ──────────────────────────

func (s *Service) SourceOrganizationExists(ctx context.Context, tenantID, id int) (bool, error) {
	return s.client.SourceOrganization.Query().Where(sourceorganization.IDEQ(id), sourceorganization.TenantIDEQ(tenantID)).Exist(ctx)
}

// 排班表的 CRUD 已归 handlers/on_call，本域不再保留第二份实现。

func (s *Service) GroupExistsInTenant(ctx context.Context, tenantID, groupID int) (bool, error) {
	return s.client.Group.Query().Where(group.IDEQ(groupID), group.TenantIDEQ(tenantID)).Exist(ctx)
}

// ─── EmailConversation ──────────────────────────────────────────────

func (s *Service) ListConversations(ctx context.Context, tenantID int, status string, page, pageSize int) ([]*ent.EmailConversation, int, error) {
	query := s.client.EmailConversation.Query().Where(emailconversation.TenantIDEQ(tenantID))
	if status != "" {
		query.Where(emailconversation.StatusEQ(status))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.
		WithCustomer().WithBranch().WithSupportContract().WithIncidents().
		// last_message_at 不是唯一键（schema 只给它建了 (tenant_id,status,last_message_at)
		// 复合索引），只按它排序时页边界不确定，同一条会话可能出现在两页或被跳过。
		Order(ent.Desc(emailconversation.FieldLastMessageAt), ent.Asc(emailconversation.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetConversation(ctx context.Context, tenantID, id int) (*ent.EmailConversation, error) {
	return s.client.EmailConversation.Query().Where(emailconversation.IDEQ(id), emailconversation.TenantIDEQ(tenantID)).WithCustomer().WithBranch().WithSupportContract().WithMessages().WithAnalyses().WithOutboundMessages().WithIncidents().Only(ctx)
}

func (s *Service) ReloadConversation(ctx context.Context, id, tenantID int) (*ent.EmailConversation, error) {
	return s.client.EmailConversation.Query().Where(emailconversation.IDEQ(id), emailconversation.TenantIDEQ(tenantID)).WithCustomer().WithBranch().WithSupportContract().WithIncidents().Only(ctx)
}

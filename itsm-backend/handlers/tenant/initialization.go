package tenant

import (
	"context"
	"fmt"

	"itsm-backend/common/tenantctx"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/operationalcommand"
	"itsm-backend/ent/systemconfig"
	"itsm-backend/internal/commandbus"

	"go.uber.org/zap"
)

// BaselineInstaller installs the product baseline into one tenant. It is the
// narrow dependency this command handler needs from the seeder package; the
// HTTP layer must not depend on the seeder's internals.
type BaselineInstaller interface {
	ProvisionTenant(ctx context.Context, tenantID int, templateVersion string) error
}

// BaselineVerifier reports a tenant's baseline state without writing anything.
// It is a func so the HTTP layer never has to name the seeder's result type.
type BaselineVerifier func(ctx context.Context, tenantID int) ([]ComponentVerification, error)

// ComponentVerification is one component's read-only verification result.
type ComponentVerification struct {
	Component string
	Verified  bool
	Error     string
}

// InitializationService reads the tenant initialization state: what the
// version marker says was installed, what the outbox command is doing, and
// what the live baseline verification reports. It never writes.
type InitializationService struct {
	client        *ent.Client
	verifier      BaselineVerifier
	templateLabel string
	logger        *zap.SugaredLogger
}

func NewInitializationService(client *ent.Client, verifier BaselineVerifier, templateVersion string, logger *zap.SugaredLogger) *InitializationService {
	return &InitializationService{client: client, verifier: verifier, templateLabel: templateVersion, logger: logger}
}

func (s *InitializationService) Status(ctx context.Context, tenantID int) (*dto.TenantInitializationStatusResponse, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant id must be positive")
	}
	if s.verifier == nil {
		return nil, fmt.Errorf("initialization verifier is not configured")
	}
	if _, err := s.client.Tenant.Get(ctx, tenantID); err != nil {
		return nil, fmt.Errorf("load target tenant: %w", err)
	}

	response := &dto.TenantInitializationStatusResponse{
		TenantID:        tenantID,
		TemplateVersion: s.templateLabel,
		CommandStatus:   "none",
		Ready:           true,
		Components:      []dto.TenantComponentVerificationResponse{},
	}

	verifications, err := s.verifier(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, verification := range verifications {
		response.Components = append(response.Components, dto.TenantComponentVerificationResponse{
			Component: verification.Component,
			Verified:  verification.Verified,
			Error:     verification.Error,
		})
		if !verification.Verified {
			response.Ready = false
		}
	}

	// 历史版本标记：仅作兼容读取，真实状态来自逐组件验证。
	marker, err := s.client.SystemConfig.Query().
		Where(
			systemconfig.KeyEQ(fmt.Sprintf("tenant.bootstrap.version.%d", tenantID)),
			systemconfig.DeletedAtIsNil(),
		).
		Only(systemContextFor(ctx, tenantID))
	if err == nil {
		response.RecordedVersion = marker.Value
		response.RecordedAt = &marker.UpdatedAt
	} else if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("read tenant template version marker: %w", err)
	}

	command, err := s.client.OperationalCommand.Query().
		Where(
			operationalcommand.TenantIDEQ(tenantID),
			operationalcommand.CommandTypeEQ(commandbus.CommandTenantBootstrapInstall),
			operationalcommand.AggregateTypeEQ("tenant"),
			operationalcommand.AggregateIDEQ(tenantID),
		).
		Order(ent.Desc(operationalcommand.FieldID)).
		First(systemContextFor(ctx, tenantID))
	switch {
	case err == nil:
		response.CommandStatus = command.Status
		response.CommandAttempts = command.Attempt
		if command.LastError != "" {
			response.CommandError = command.LastError
		}
	case ent.IsNotFound(err):
		// 旧租户在命令机制上线前创建，没有命令记录。
	default:
		return nil, fmt.Errorf("read tenant bootstrap command: %w", err)
	}

	return response, nil
}

func systemContextFor(ctx context.Context, tenantID int) context.Context {
	return tenantctx.SystemContext(
		ctx,
		"tenant:initialization_status",
		fmt.Sprintf("read-only initialization status for tenant %d", tenantID),
	)
}

// ProvisioningCommandHandler installs the tenant product baseline when the
// outbox delivers a tenant.bootstrap.install command.
type ProvisioningCommandHandler struct {
	client          *ent.Client
	installer       BaselineInstaller
	templateVersion string
	logger          *zap.SugaredLogger
}

func NewProvisioningCommandHandler(client *ent.Client, installer BaselineInstaller, templateVersion string, logger *zap.SugaredLogger) *ProvisioningCommandHandler {
	return &ProvisioningCommandHandler{client: client, installer: installer, templateVersion: templateVersion, logger: logger}
}

func (h *ProvisioningCommandHandler) Handle(ctx context.Context, cmd *ent.OperationalCommand) error {
	if cmd == nil ||
		cmd.CommandType != commandbus.CommandTenantBootstrapInstall ||
		cmd.AggregateType != "tenant" ||
		cmd.AggregateID <= 0 ||
		cmd.TenantID != cmd.AggregateID {
		return fmt.Errorf("invalid tenant bootstrap command")
	}
	if h.installer == nil {
		return fmt.Errorf("tenant baseline installer is not configured")
	}

	// 消费者不得信任 payload 自报身份：按命令的租户重新加载权威租户行，
	// 不存在或已停用就不安装，并让命令进入可观察失败。
	systemCtx := tenantctx.SystemContext(
		ctx,
		"commandbus:tenant-bootstrap",
		fmt.Sprintf("install product baseline for tenant %d", cmd.AggregateID),
	)
	target, err := h.client.Tenant.Get(systemCtx, cmd.AggregateID)
	if err != nil {
		return fmt.Errorf("load tenant %d for baseline install: %w", cmd.AggregateID, err)
	}
	if target.Code == protectedSystemTenantCode {
		// 平台租户的基线由 bootstrap DAG 负责，这里重复执行会跑乱步骤顺序。
		h.logger.Debugw("tenant bootstrap skipped for platform tenant", "tenant_id", target.ID)
		return nil
	}
	if target.Status == "deleted" {
		return fmt.Errorf("tenant %d is deleted; refusing baseline install", target.ID)
	}

	h.logger.Infow("tenant baseline install started", "tenant_id", target.ID, "template_version", h.templateVersion)
	if err := h.installer.ProvisionTenant(systemCtx, target.ID, h.templateVersion); err != nil {
		return fmt.Errorf("install product baseline for tenant %d: %w", target.ID, err)
	}
	h.logger.Infow("tenant baseline install completed", "tenant_id", target.ID, "template_version", h.templateVersion)
	return nil
}

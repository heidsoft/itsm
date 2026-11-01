package bootstrap

import (
	"context"
	"database/sql"
	"errors"

	"go.uber.org/zap"

	"itsm-backend/handlers/cloudsignal"
	"itsm-backend/handlers/cmdb"
	cloudruntime "itsm-backend/service/cloud"
)

// 云信号同步只接了阿里云：CMS/SAS 客户端与凭据解析都是阿里云口径。
const cloudSignalProvider = "aliyun"

// cloudSignalRuntime 汇总云监控告警、安全漏洞与建单桥三个后台任务，
// 以及 HTTP 层需要的仓储实例，避免两处各自 new 一份。
type cloudSignalRuntime struct {
	alarms          *cloudsignal.AlarmSyncService
	vulnerabilities *cloudsignal.VulnerabilitySyncService
	bridge          *cloudsignal.IncidentBridge
	states          cloudsignal.SyncStateRepository
	findings        cloudsignal.FindingRepository
	alerts          cloudsignal.AlertRepository
	handler         *cloudsignal.Handler
	logger          *zap.SugaredLogger
}

func newCloudSignalRuntime(
	db *sql.DB,
	accounts cloudsignal.CloudAccountSource,
	creator cloudsignal.IncidentCreator,
	reporterID int,
	resolver cloudruntime.TenantSecretResolver,
	logger *zap.SugaredLogger,
) *cloudSignalRuntime {
	if db == nil {
		return nil
	}
	if logger == nil {
		logger = zap.NewNop().Sugar()
	}
	states := cloudsignal.NewSyncStateRepository(db)
	findings := cloudsignal.NewFindingRepository(db)
	alerts := cloudsignal.NewAlertRepository(db)
	return &cloudSignalRuntime{
		alarms:          cloudsignal.NewAlarmSyncService(states, alerts, accounts, resolver, logger),
		vulnerabilities: cloudsignal.NewVulnerabilitySyncService(states, findings, accounts, resolver, logger),
		bridge:          cloudsignal.NewIncidentBridge(alerts, cloudsignal.NewBridgeRepository(db), creator, reporterID, logger),
		states:          states,
		findings:        findings,
		alerts:          alerts,
		handler:         cloudsignal.NewHandler(cloudsignal.NewService(alerts, findings, states, logger)),
		logger:          logger,
	}
}

// HTTPHandler 返回路由层要用的处理器；运行时未装配时返回 nil，对应路由不注册。
func (r *cloudSignalRuntime) HTTPHandler() *cloudsignal.Handler {
	if r == nil {
		return nil
	}
	return r.handler
}

// runAlarmRound 先拉一轮告警再建单：同一分钟内的告警本轮就能变成事件。
func (r *cloudSignalRuntime) runAlarmRound(ctx context.Context, tenantID int) {
	summary, err := r.alarms.SyncTenant(ctx, tenantID)
	if err != nil {
		r.logger.Warnw("cloud monitor alarm sync failed", "tenant_id", tenantID, "error", err)
	} else if summary.Persisted > 0 {
		r.logger.Infow("cloud monitor alarm sync completed", "tenant_id", tenantID,
			"accounts", summary.Accounts, "fetched", summary.Fetched,
			"created", summary.Created, "persisted", summary.Persisted)
	}

	bridgeResult, err := r.bridge.RunTenant(ctx, tenantID)
	if errors.Is(err, cloudsignal.ErrBridgeReporterUnconfigured) {
		return
	}
	if err != nil {
		r.logger.Warnw("cloud alert bridging failed", "tenant_id", tenantID, "error", err)
		return
	}
	if bridgeResult.Created > 0 {
		r.logger.Infow("cloud alerts bridged to incidents", "tenant_id", tenantID,
			"claimed", bridgeResult.Claimed, "created", bridgeResult.Created,
			"reused", bridgeResult.Reused, "failed", bridgeResult.Failed)
	}
}

func (r *cloudSignalRuntime) runVulnerabilityRound(ctx context.Context, tenantID int) {
	summary, err := r.vulnerabilities.SyncTenant(ctx, tenantID)
	if err != nil {
		r.logger.Warnw("security vulnerability sync failed", "tenant_id", tenantID, "error", err)
		return
	}
	if summary.Persisted > 0 || summary.Resolved > 0 {
		r.logger.Infow("security vulnerability sync completed", "tenant_id", tenantID,
			"accounts", summary.Accounts, "persisted", summary.Persisted,
			"created", summary.Created, "resolved", summary.Resolved, "linked", summary.Linked)
	}
}

// cmdbCloudAccountSource 把 CMDB 的云账号仓储收窄成同步需要的投影。
// 放在装配层，cloudsignal 不直接依赖其他领域的仓储实现。
type cmdbCloudAccountSource struct {
	repo interface {
		ListCloudAccounts(ctx context.Context, tenantID int, provider string) ([]*cmdb.CloudAccount, error)
	}
}

func (s cmdbCloudAccountSource) ListActiveCloudAccounts(ctx context.Context, tenantID int) ([]cloudsignal.CloudAccountRef, error) {
	accounts, err := s.repo.ListCloudAccounts(ctx, tenantID, cloudSignalProvider)
	if err != nil {
		return nil, err
	}
	refs := make([]cloudsignal.CloudAccountRef, 0, len(accounts))
	for _, account := range accounts {
		if account == nil || !account.IsActive || account.Provider != cloudSignalProvider {
			continue
		}
		refs = append(refs, cloudsignal.CloudAccountRef{
			ID:            int64(account.ID),
			TenantID:      account.TenantID,
			Provider:      account.Provider,
			AccountName:   account.AccountName,
			CredentialRef: account.CredentialRef,
		})
	}
	return refs, nil
}

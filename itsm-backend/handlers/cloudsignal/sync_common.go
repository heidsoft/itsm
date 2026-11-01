package cloudsignal

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"itsm-backend/service/cloud"
)

// resolveAccountCredential 把凭据引用解析成阿里云运行时凭据。
// 引用既可以是 secret://（由部署注入的解析器取值），也可以是 env://（试点的固定环境变量）。
func resolveAccountCredential(ctx context.Context, resolver cloud.TenantSecretResolver, account CloudAccountRef) (*cloud.ResolvedCredential, error) {
	if resolver == nil {
		return nil, fmt.Errorf("cloud credential resolver is not configured")
	}
	resolved, err := resolver.Resolve(ctx, account.TenantID, account.CredentialRef)
	if err != nil {
		return nil, fmt.Errorf("resolve cloud credential: %w", err)
	}
	cred, err := cloud.ResolveAliyunCredential(ctx, resolved)
	if err != nil {
		return nil, fmt.Errorf("parse aliyun credential: %w", err)
	}
	return cred, nil
}

// recordSignalFailure 把一次同步失败写进状态表。写状态本身失败只记日志，
// 不覆盖调用方正在返回的原始错误。
func recordSignalFailure(ctx context.Context, states SyncStateRepository, logger *zap.SugaredLogger,
	tenantID int, accountID int64, kind SignalKind, code string, cause error, now time.Time) {
	message := truncateRunes(cause.Error(), 500)
	if err := states.MarkFailure(ctx, tenantID, accountID, kind, SyncStateUnavailable, code, message, now); err != nil {
		logger.Warnw("record cloud signal sync failure failed",
			"tenant_id", tenantID, "account_id", accountID, "signal_kind", string(kind), "error", err)
	}
}

func markSignalUnconfigured(ctx context.Context, states SyncStateRepository, logger *zap.SugaredLogger,
	tenantID int, accountID int64, kind SignalKind, now time.Time) {
	if err := states.MarkUnconfigured(ctx, tenantID, accountID, kind, now); err != nil {
		logger.Warnw("mark cloud signal sync unconfigured failed",
			"tenant_id", tenantID, "account_id", accountID, "signal_kind", string(kind), "error", err)
	}
}

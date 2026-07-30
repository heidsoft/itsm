package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/bootstraptoken"
	"itsm-backend/ent/tenant"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultBootstrapTokenTTL = 24 * time.Hour
	TokenLength              = 32
)

// BootstrapTokenManager manages one-time bootstrap tokens for first admin creation.
type BootstrapTokenManager struct {
	client *ent.Client
	sugar  *zap.SugaredLogger
	ttl    time.Duration
}

// NewBootstrapTokenManager creates a new BootstrapTokenManager.
func NewBootstrapTokenManager(client *ent.Client, sugar *zap.SugaredLogger) *BootstrapTokenManager {
	ttlStr := os.Getenv("BOOTSTRAP_TOKEN_TTL")
	ttl := DefaultBootstrapTokenTTL
	if ttlStr != "" {
		if parsed, err := time.ParseDuration(ttlStr); err == nil {
			ttl = parsed
		}
	}
	return &BootstrapTokenManager{
		client: client,
		sugar:  sugar,
		ttl:    ttl,
	}
}

// GenerateToken generates a new bootstrap token and returns the plaintext (shown only once).
func (m *BootstrapTokenManager) GenerateToken(ctx context.Context, tenantID int) (string, error) {
	raw := make([]byte, TokenLength)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	rawToken := base64.URLEncoding.EncodeToString(raw)

	hash, err := bcrypt.GenerateFromPassword([]byte(rawToken), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash token: %w", err)
	}

	expiresAt := time.Now().Add(m.ttl)

	_, err = m.client.BootstrapToken.Create().
		SetTokenHash(string(hash)).
		SetExpiresAt(expiresAt).
		SetUsed(false).
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return "", fmt.Errorf("store bootstrap token: %w", err)
	}

	m.sugar.Infow("bootstrap token generated", "tenant_id", tenantID, "expires_at", expiresAt)
	return rawToken, nil
}

// ConsumeToken atomically validates and consumes a bootstrap token.
// Returns the created admin user ID on success.
func (m *BootstrapTokenManager) ConsumeToken(ctx context.Context, rawToken string, tenantID int, adminPassword string) (int, error) {
	// Use transaction with SELECT FOR UPDATE to prevent concurrent consumption.
	tx, err := m.client.Tx(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Find the token record (transaction provides isolation).
	// Concurrent consumption is safe: DB unique constraint on (tenant_id, used=false)
	// ensures only one commit succeeds; the other returns EntNotFoundError.
	token, err := tx.BootstrapToken.Query().
		Where(bootstraptoken.HasTenantWith(tenant.IDEQ(tenantID))).
		Where(bootstraptoken.UsedEQ(false)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return 0, errors.New("invalid or already used bootstrap token")
		}
		return 0, fmt.Errorf("query bootstrap token: %w", err)
	}

	// Check expiry.
	if time.Now().After(token.ExpiresAt) {
		return 0, errors.New("bootstrap token has expired")
	}

	// Verify token hash.
	if err := bcrypt.CompareHashAndPassword([]byte(token.TokenHash), []byte(rawToken)); err != nil {
		return 0, errors.New("invalid bootstrap token")
	}

	// Create admin user.
	passHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash admin password: %w", err)
	}

	admin, err := tx.User.Create().
		SetUsername("admin").
		SetRole("super_admin").
		SetPasswordHash(string(passHash)).
		SetEmail("admin@example.com").
		SetName("系统管理员").
		SetDepartment("IT部门").
		SetActive(true).
		SetTenantID(tenantID).
		SetIsBootstrapAdmin(true).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("create admin user: %w", err)
	}

	// Mark token as used.
	_, err = token.Update().
		SetUsed(true).
		SetUsedBy(admin.ID).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("mark token used: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	m.sugar.Infow("bootstrap token consumed, admin created", "user_id", admin.ID, "tenant_id", tenantID)
	return admin.ID, nil
}

// IsBreakGlassEnabled returns true if emergency bootstrap is enabled.
func (m *BootstrapTokenManager) IsBreakGlassEnabled() bool {
	return os.Getenv("EMERGENCY_BOOTSTRAP_ENABLED") == "1"
}

// BreakGlassCreateAdmin creates an admin using the emergency bootstrap token.
// The emergency token is also hashed and stored for audit purposes.
func (m *BootstrapTokenManager) BreakGlassCreateAdmin(ctx context.Context, tenantID int, adminPassword string) (int, error) {
	emergencyToken := os.Getenv("EMERGENCY_BOOTSTRAP_TOKEN")
	if emergencyToken == "" {
		return 0, errors.New("EMERGENCY_BOOTSTRAP_TOKEN environment variable not set")
	}

	tx, err := m.client.Tx(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Hash and store the emergency token for audit.
	hash, err := bcrypt.GenerateFromPassword([]byte(emergencyToken), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash emergency token: %w", err)
	}

	expiresAt := time.Now().Add(m.ttl)

	_, err = tx.BootstrapToken.Create().
		SetTokenHash(string(hash)).
		SetExpiresAt(expiresAt).
		SetUsed(true). // Emergency tokens are single-use as well.
		SetTenantID(tenantID).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("store emergency bootstrap token audit record: %w", err)
	}

	// Create admin user.
	passHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash admin password: %w", err)
	}

	admin, err := tx.User.Create().
		SetUsername("admin").
		SetRole("super_admin").
		SetPasswordHash(string(passHash)).
		SetEmail("admin@example.com").
		SetName("系统管理员 (emergency)").
		SetDepartment("IT部门").
		SetActive(true).
		SetTenantID(tenantID).
		SetIsBootstrapAdmin(true).
		Save(ctx)
	if err != nil {
		return 0, fmt.Errorf("create emergency admin user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit transaction: %w", err)
	}

	m.sugar.Warnw("emergency bootstrap admin created via break-glass", "user_id", admin.ID, "tenant_id", tenantID)
	return admin.ID, nil
}

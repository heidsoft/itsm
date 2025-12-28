package common

import (
	"context"
	"fmt"
	"time"

	"itsm-backend/ent"
	enttenant "itsm-backend/ent/tenant"
	"itsm-backend/middleware"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      Repository
	jwtSecret string
	logger    *zap.SugaredLogger
	client    *ent.Client // For legacy integrations if needed
}

func NewService(repo Repository, jwtSecret string, logger *zap.SugaredLogger, client *ent.Client) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
		logger:    logger,
		client:    client,
	}
}

// Auth

func (s *Service) Login(ctx context.Context, username, password string, tenantID int, tenantCode string) (*AuthResult, error) {
	// Resolve tenant
	if tenantID == 0 && tenantCode != "" {
		t, err := s.client.Tenant.Query().Where(enttenant.CodeEQ(tenantCode)).First(ctx)
		if err == nil {
			tenantID = t.ID
		}
	}
	if tenantID == 0 {
		// fallback to default tenant if exists
		if t, err := s.client.Tenant.Query().Where(enttenant.CodeEQ("default")).First(ctx); err == nil {
			tenantID = t.ID
		}
	}
	u, err := s.repo.GetUserByUsername(ctx, username, tenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password (need to get hash from DB, but domain User doesn't have it)
	// I'll fetch the ent user directly for password verification to keep domain User clean
	entUser, err := s.client.User.Get(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(entUser.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !u.Active {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Generate tokens
	accessToken, err := middleware.GenerateAccessToken(u.ID, u.Username, u.Role, u.TenantID, s.jwtSecret, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	refreshToken, err := middleware.GenerateRefreshToken(u.ID, s.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         u,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := middleware.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// regenerate tokens
	accessToken, err := middleware.GenerateAccessToken(user.ID, user.Username, user.Role, user.TenantID, s.jwtSecret, 15*time.Minute)
	if err != nil {
		return nil, err
	}

	newRefresh, err := middleware.GenerateRefreshToken(user.ID, s.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		User:         user,
	}, nil
}

// User Management

func (s *Service) GetUser(ctx context.Context, id int) (*User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) ListUsers(ctx context.Context, tenantID int) ([]*User, error) {
	return s.repo.ListUsers(ctx, tenantID)
}

// Organization Management

func (s *Service) GetDepartmentTree(ctx context.Context, tenantID int) ([]*Department, error) {
	return s.repo.GetDepartmentTree(ctx, tenantID)
}

func (s *Service) CreateDepartment(ctx context.Context, d *Department) (*Department, error) {
	return s.repo.CreateDepartment(ctx, d)
}

func (s *Service) ListTeams(ctx context.Context, tenantID int) ([]*Team, error) {
	return s.repo.ListTeams(ctx, tenantID)
}

func (s *Service) CreateTeam(ctx context.Context, t *Team) (*Team, error) {
	return s.repo.CreateTeam(ctx, t)
}

func (s *Service) AddTeamMember(ctx context.Context, teamID int, userID int) error {
	return s.repo.AddTeamMember(ctx, teamID, userID)
}

// Tags

func (s *Service) ListTags(ctx context.Context, tenantID int) ([]*Tag, error) {
	return s.repo.ListTags(ctx, tenantID)
}

func (s *Service) CreateTag(ctx context.Context, t *Tag) (*Tag, error) {
	return s.repo.CreateTag(ctx, t)
}

// Auditing

func (s *Service) LogActivity(ctx context.Context, log *AuditLog) error {
	return s.repo.CreateAuditLog(ctx, log)
}

func (s *Service) GetAuditLogs(ctx context.Context, tenantID int, userID int) ([]*AuditLog, error) {
	return s.repo.ListAuditLogs(ctx, tenantID, userID, 100)
}

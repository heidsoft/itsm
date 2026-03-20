package common

import (
	"context"
)

// Repository interface for Common domain
type Repository interface {
	// Auth & User
	GetUserByUsername(ctx context.Context, username string, tenantID int) (*User, error)
	GetUserByID(ctx context.Context, id int) (*User, error)
	ListUsers(ctx context.Context, tenantID int) ([]*User, error)
	CreateUser(ctx context.Context, u *User) (*User, error)
	UpdateUser(ctx context.Context, u *User) (*User, error)

	// Department
	CreateDepartment(ctx context.Context, d *Department) (*Department, error)
	GetDepartment(ctx context.Context, id int, tenantID int) (*Department, error)
	ListDepartments(ctx context.Context, tenantID int) ([]*Department, error)
	GetDepartmentTree(ctx context.Context, tenantID int) ([]*Department, error)
	UpdateDepartment(ctx context.Context, d *Department) (*Department, error)
	DeleteDepartment(ctx context.Context, id int, tenantID int) error

	// Team
	CreateTeam(ctx context.Context, t *Team) (*Team, error)
	GetTeam(ctx context.Context, id int, tenantID int) (*Team, error)
	ListTeams(ctx context.Context, tenantID int) ([]*Team, error)
	UpdateTeam(ctx context.Context, t *Team) (*Team, error)
	DeleteTeam(ctx context.Context, id int, tenantID int) error
	AddTeamMember(ctx context.Context, teamID int, userID int) error

	// Tag
	CreateTag(ctx context.Context, t *Tag) (*Tag, error)
	ListTags(ctx context.Context, tenantID int) ([]*Tag, error)
	DeleteTag(ctx context.Context, id int, tenantID int) error

	// Audit Log
	CreateAuditLog(ctx context.Context, l *AuditLog) error
	ListAuditLogs(ctx context.Context, tenantID int, userID int, limit int) ([]*AuditLog, error)
}

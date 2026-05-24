package service

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/stretchr/testify/assert"
	"itsm-backend/ent/enttest"
)

func TestTenantAwareRepository_ValidateTenantAccess(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	repo := NewTenantAwareRepository(client, 100)

	tests := []struct {
		name           string
		entityTenantID int
		wantErr        bool
	}{
		{"same tenant", 100, false},
		{"different tenant", 200, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.ValidateTenantAccess(context.Background(), tt.entityTenantID)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "cross-tenant access denied")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

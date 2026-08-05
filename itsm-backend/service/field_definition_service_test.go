package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldDefinitionSchema_RoundTrip(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:field_definition_schema?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	created, err := client.FieldDefinition.Create().
		SetTenantID(1).
		SetEntityType("ticket_template").
		SetEntityID(4).
		SetName("office_location").
		SetLabel("办公地点").
		SetFieldType("text").
		SetRequired(true).
		SetSortOrder(0).
		Save(ctx)
	require.NoError(t, err)
	assert.Equal(t, "office_location", created.Name)

	fetched, err := client.FieldDefinition.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "办公地点", fetched.Label)
}

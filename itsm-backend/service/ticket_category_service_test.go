package service

import (
	"testing"

	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestMoveCategoryUpdatesParentSortOrderAndDescendantLevels(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_category_move?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx := t.Context()
	svc := NewTicketCategoryService(client)

	root, err := svc.CreateCategory(ctx, &CreateCategoryRequest{
		Name:      "根分类",
		Code:      "root",
		SortOrder: 1,
		IsActive:  true,
		TenantID:  1,
	})
	require.NoError(t, err)

	child, err := svc.CreateCategory(ctx, &CreateCategoryRequest{
		Name:      "子分类",
		Code:      "child",
		ParentID:  root.ID,
		SortOrder: 1,
		IsActive:  true,
		TenantID:  1,
	})
	require.NoError(t, err)

	grandchild, err := svc.CreateCategory(ctx, &CreateCategoryRequest{
		Name:      "孙分类",
		Code:      "grandchild",
		ParentID:  child.ID,
		SortOrder: 1,
		IsActive:  true,
		TenantID:  1,
	})
	require.NoError(t, err)

	sortOrder := 7
	newParentID := 0
	moved, err := svc.MoveCategory(ctx, child.ID, &MoveCategoryRequest{
		NewParentID:  &newParentID,
		NewSortOrder: &sortOrder,
	}, 1)
	require.NoError(t, err)
	require.Equal(t, 0, moved.ParentID)
	require.Equal(t, 1, moved.Level)
	require.Equal(t, 7, moved.SortOrder)

	reloadedGrandchild, err := svc.GetCategory(ctx, grandchild.ID, 1)
	require.NoError(t, err)
	require.Equal(t, 2, reloadedGrandchild.Level)
}

func TestMoveCategoryRejectsMovingUnderDescendant(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ticket_category_cycle?mode=memory&cache=shared&_fk=1")
	t.Cleanup(func() { client.Close() })

	ctx := t.Context()
	svc := NewTicketCategoryService(client)

	root, err := svc.CreateCategory(ctx, &CreateCategoryRequest{
		Name:     "根分类",
		Code:     "cycle-root",
		IsActive: true,
		TenantID: 1,
	})
	require.NoError(t, err)

	child, err := svc.CreateCategory(ctx, &CreateCategoryRequest{
		Name:     "子分类",
		Code:     "cycle-child",
		ParentID: root.ID,
		IsActive: true,
		TenantID: 1,
	})
	require.NoError(t, err)

	newParentID := child.ID
	_, err = svc.MoveCategory(ctx, root.ID, &MoveCategoryRequest{NewParentID: &newParentID}, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "子分类")
}

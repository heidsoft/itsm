package service

import (
	"context"
	"testing"

	"itsm-backend/ent/enttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagService_CreateTag(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tagService := NewTagService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		nameVal       string
		codeVal       string
		description   string
		color         string
		tenantID      int
		expectedError bool
	}{
		{
			name:          "成功创建标签",
			nameVal:       "紧急",
			codeVal:       "URGENT",
			description:   "紧急事项",
			color:         "#FF0000",
			tenantID:      testTenant.ID,
			expectedError: false,
		},
		{
			name:          "重复标签代码",
			nameVal:       "另一个紧急",
			codeVal:       "URGENT",
			description:   "重复代码",
			color:         "#FF0000",
			tenantID:      testTenant.ID,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, err := tagService.CreateTag(ctx, tt.nameVal, tt.codeVal, tt.description, tt.color, tt.tenantID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, tag)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tag)
				assert.Equal(t, tt.nameVal, tag.Name)
				assert.Equal(t, tt.codeVal, tag.Code)
			}
		})
	}
}

func TestTagService_ListTags(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tagService := NewTagService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 为租户创建标签
	_, err = client.Tag.Create().
		SetName("标签A").
		SetCode("TAG-A").
		SetColor("#FF0000").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.Tag.Create().
		SetName("标签B").
		SetCode("TAG-B").
		SetColor("#00FF00").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 测试获取标签列表
	tags, err := tagService.ListTags(ctx, testTenant.ID)
	assert.NoError(t, err)
	assert.Len(t, tags, 2)
}

func TestTagService_UpdateTag(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tagService := NewTagService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建标签
	testTag, err := client.Tag.Create().
		SetName("原始标签").
		SetCode("ORIGINAL").
		SetColor("#FF0000").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 测试更新
	newName := "新标签名"
	newColor := "#00FF00"
	tag, err := tagService.UpdateTag(ctx, testTag.ID, &newName, nil, nil, &newColor, testTenant.ID)
	assert.NoError(t, err)
	assert.NotNil(t, tag)
	assert.Equal(t, newName, tag.Name)
	assert.Equal(t, newColor, tag.Color)
}

func TestTagService_DeleteTag(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	tagService := NewTagService(client)

	ctx := context.Background()

	// 创建测试租户
	testTenant, err := client.Tenant.Create().
		SetName("Test Tenant").
		SetCode("TEST").
		SetDomain("test.com").
		SetStatus("active").
		Save(ctx)
	require.NoError(t, err)

	// 创建标签
	testTag, err := client.Tag.Create().
		SetName("待删除标签").
		SetCode("DELETE-ME").
		SetColor("#FF0000").
		SetTenantID(testTenant.ID).
		Save(ctx)
	require.NoError(t, err)

	// 删除标签
	err = tagService.DeleteTag(ctx, testTag.ID, testTenant.ID)
	assert.NoError(t, err)
}

package service

import (
	"context"
	"strings"
	"testing"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/enttest"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap/zaptest"
)

// createTestAttrDefinition 创建属性定义测试夹具
func createTestAttrDefinition(
	ctx context.Context, client *ent.Client, tenantID, ciTypeID int,
	name, displayName, attrType string,
	required, unique bool, defaultValue, validationRules string,
) (*ent.CIAttributeDefinition, error) {
	create := client.CIAttributeDefinition.Create().
		SetName(name).
		SetDisplayName(displayName).
		SetType(attrType).
		SetRequired(required).
		SetUnique(unique).
		SetCiTypeID(ciTypeID).
		SetTenantID(tenantID)
	if defaultValue != "" {
		create.SetDefaultValue(defaultValue)
	}
	if validationRules != "" {
		create.SetValidationRules(validationRules)
	}
	return create.Save(ctx)
}

func TestNormalizeAttributesTableDriven(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:attr_validator_table?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant, err := createCMDBTestTenant(ctx, client, "Attr Tenant", "attr-tenant", "attr.example.com")
	if err != nil {
		t.Fatal(err)
	}
	ciType, err := createTestCIType(ctx, client, tenant.ID, "服务器")
	if err != nil {
		t.Fatal(err)
	}

	// owner：必填字符串；cpu_cores：整数 + 数值范围；env：枚举；tier：带默认值
	if _, err := createTestAttrDefinition(ctx, client, tenant.ID, ciType.ID,
		"owner", "负责人", "string", true, false, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := createTestAttrDefinition(ctx, client, tenant.ID, ciType.ID,
		"cpu_cores", "CPU核数", "integer", false, false, "", `{"min":1,"max":128}`); err != nil {
		t.Fatal(err)
	}
	if _, err := createTestAttrDefinition(ctx, client, tenant.ID, ciType.ID,
		"env", "环境", "string", false, false, "", `{"enum_values":["prod","staging"]}`); err != nil {
		t.Fatal(err)
	}
	if _, err := createTestAttrDefinition(ctx, client, tenant.ID, ciType.ID,
		"tier", "层级", "string", false, false, "standard", ""); err != nil {
		t.Fatal(err)
	}

	validator := NewCIAttributeValidator(client)

	tests := []struct {
		name       string
		attributes map[string]interface{}
		wantErr    string
		check      func(t *testing.T, got map[string]interface{})
	}{
		{
			name:       "必填属性缺失",
			attributes: map[string]interface{}{},
			wantErr:    "必填",
		},
		{
			name:       "必填属性为空白字符串",
			attributes: map[string]interface{}{"owner": "   "},
			wantErr:    "必填",
		},
		{
			name:       "整数字符串强转为整数",
			attributes: map[string]interface{}{"owner": "张三", "cpu_cores": "8"},
			check: func(t *testing.T, got map[string]interface{}) {
				if got["cpu_cores"] != 8 {
					t.Fatalf("cpu_cores = %#v, want int 8", got["cpu_cores"])
				}
			},
		},
		{
			name:       "整数类型非法",
			attributes: map[string]interface{}{"owner": "张三", "cpu_cores": "abc"},
			wantErr:    "需要整数",
		},
		{
			name:       "数值超出范围",
			attributes: map[string]interface{}{"owner": "张三", "cpu_cores": 999},
			wantErr:    "不能大于",
		},
		{
			name:       "枚举值合法",
			attributes: map[string]interface{}{"owner": "张三", "env": "prod"},
		},
		{
			name:       "枚举值非法",
			attributes: map[string]interface{}{"owner": "张三", "env": "dev"},
			wantErr:    "不在允许范围内",
		},
		{
			name:       "默认值自动填充",
			attributes: map[string]interface{}{"owner": "张三"},
			check: func(t *testing.T, got map[string]interface{}) {
				if got["tier"] != "standard" {
					t.Fatalf("tier = %#v, want standard", got["tier"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validator.NormalizeAttributes(ctx, tenant.ID, ciType.ID, tt.attributes, 0)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want contains %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestNormalizeAttributesUnique(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:attr_validator_unique?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant, err := createCMDBTestTenant(ctx, client, "Unique Tenant", "unique-tenant", "unique.example.com")
	if err != nil {
		t.Fatal(err)
	}
	ciType, err := createTestCIType(ctx, client, tenant.ID, "服务器")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createTestAttrDefinition(ctx, client, tenant.ID, ciType.ID,
		"serial", "序列号", "string", false, true, "", ""); err != nil {
		t.Fatal(err)
	}

	existing, err := client.ConfigurationItem.Create().
		SetName("已有CI").SetCiType(ciType.Name).SetCiTypeID(ciType.ID).
		SetStatus("active").SetTenantID(tenant.ID).
		SetAttributes(map[string]interface{}{"serial": "SN-001"}).
		Save(ctx)
	if err != nil {
		t.Fatal(err)
	}

	validator := NewCIAttributeValidator(client)

	// 创建场景：与已有 CI 属性值冲突
	if _, err := validator.NormalizeAttributes(ctx, tenant.ID, ciType.ID,
		map[string]interface{}{"serial": "SN-001"}, 0); err == nil || !strings.Contains(err.Error(), "唯一") {
		t.Fatalf("err = %v, want unique violation", err)
	}
	// 更新场景：排除自身后不冲突
	if _, err := validator.NormalizeAttributes(ctx, tenant.ID, ciType.ID,
		map[string]interface{}{"serial": "SN-001"}, existing.ID); err != nil {
		t.Fatalf("self-exclusion failed: %v", err)
	}
	// 不同值不冲突
	if _, err := validator.NormalizeAttributes(ctx, tenant.ID, ciType.ID,
		map[string]interface{}{"serial": "SN-002"}, 0); err != nil {
		t.Fatalf("distinct value rejected: %v", err)
	}
}

func TestNormalizeAttributesInheritance(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:attr_validator_inherit?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()

	tenant, err := createCMDBTestTenant(ctx, client, "Inherit Tenant", "inherit-tenant", "inherit.example.com")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := createTestCIType(ctx, client, tenant.ID, "计算资源")
	if err != nil {
		t.Fatal(err)
	}
	child, err := client.CIType.Create().
		SetName("物理服务器").SetTenantID(tenant.ID).SetParentTypeID(parent.ID).
		Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 必填定义挂在父类型上，子类型应继承
	if _, err := createTestAttrDefinition(ctx, client, tenant.ID, parent.ID,
		"owner", "负责人", "string", true, false, "", ""); err != nil {
		t.Fatal(err)
	}

	validator := NewCIAttributeValidator(client)

	if _, err := validator.NormalizeAttributes(ctx, tenant.ID, child.ID,
		map[string]interface{}{}, 0); err == nil || !strings.Contains(err.Error(), "必填") {
		t.Fatalf("err = %v, want inherited required violation", err)
	}
	if _, err := validator.NormalizeAttributes(ctx, tenant.ID, child.ID,
		map[string]interface{}{"owner": "张三"}, 0); err != nil {
		t.Fatalf("inherited attribute rejected: %v", err)
	}
}

// TestCreateCIRejectsInvalidAttributes 校验接入线上路径：CreateCI 前置属性校验
func TestCreateCIRejectsInvalidAttributes(t *testing.T) {
	client := enttest.Open(t, "sqlite3", "file:attr_validator_createci?mode=memory&cache=shared&_fk=1")
	defer client.Close()
	ctx := context.Background()
	logger := zaptest.NewLogger(t).Sugar()

	tenant, err := createCMDBTestTenant(ctx, client, "CreateCI Tenant", "createci-tenant", "createci.example.com")
	if err != nil {
		t.Fatal(err)
	}
	ciType, err := createTestCIType(ctx, client, tenant.ID, "服务器")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createTestAttrDefinition(ctx, client, tenant.ID, ciType.ID,
		"owner", "负责人", "string", true, false, "", ""); err != nil {
		t.Fatal(err)
	}

	historySvc := NewCIHistoryService(client, logger)
	tagSvc := NewCITagService(client, logger)
	svc := NewConfigurationItemService(client, logger, historySvc, tagSvc)

	// 缺失必填属性应被拒绝
	_, err = svc.CreateCI(ctx, &dto.CreateCIRequest{
		Name:     "web-01",
		CITypeID: ciType.ID,
		Status:   "active",
	}, tenant.ID)
	if err == nil || !strings.Contains(err.Error(), "必填") {
		t.Fatalf("err = %v, want required attribute violation", err)
	}

	// 合法属性创建成功且属性完成归一化
	resp, err := svc.CreateCI(ctx, &dto.CreateCIRequest{
		Name:       "web-01",
		CITypeID:   ciType.ID,
		Status:     "active",
		Attributes: map[string]interface{}{"owner": "张三"},
	}, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Attributes["owner"] != "张三" {
		t.Fatalf("owner = %#v, want 张三", resp.Attributes["owner"])
	}
}

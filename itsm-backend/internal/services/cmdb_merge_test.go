package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultManagedFields(t *testing.T) {
	fields := defaultManagedFields()
	assert.NotEmpty(t, fields)

	// 核心字段必须在托管清单中，且策略符合三方合并语义
	byName := make(map[string]ManagedField, len(fields))
	for _, f := range fields {
		_, dup := byName[f.FieldName]
		assert.False(t, dup, "duplicate managed field: %s", f.FieldName)
		byName[f.FieldName] = f
	}

	name, ok := byName["Name"]
	assert.True(t, ok)
	assert.Equal(t, AlwaysUseIncoming, name.Strategy)
	assert.False(t, name.CanBeEmpty)

	assigned, ok := byName["AssignedTo"]
	assert.True(t, ok)
	assert.Equal(t, UseExistingOrDefault, assigned.Strategy)

	serial, ok := byName["SerialNumber"]
	assert.True(t, ok)
	assert.Equal(t, UseIncomingOrExistingOrDefault, serial.Strategy)
}

func TestResolveField_OwnershipProtectedFields(t *testing.T) {
	mf := ManagedField{FieldName: "OwnedBy", Strategy: AlwaysUseIncoming, CanBeEmpty: true}

	tests := []struct {
		name          string
		ownershipMode string
		incoming      interface{}
		existing      interface{}
		defaultVal    interface{}
		want          interface{}
	}{
		{
			name:          "customer 模式保护存量归属人",
			ownershipMode: OwnershipModeCustomer,
			incoming:      "discovered-user",
			existing:      "customer-user",
			defaultVal:    "sla-user",
			want:          "customer-user",
		},
		{
			name:          "customer 模式存量为空时回退默认值",
			ownershipMode: OwnershipModeCustomer,
			incoming:      "discovered-user",
			existing:      "",
			defaultVal:    "sla-user",
			want:          "sla-user",
		},
		{
			name:          "sla 模式默认值优先",
			ownershipMode: OwnershipModeSLA,
			incoming:      "discovered-user",
			existing:      "customer-user",
			defaultVal:    "sla-user",
			want:          "sla-user",
		},
		{
			name:          "managed 模式来源值优先",
			ownershipMode: OwnershipModeManaged,
			incoming:      "discovered-user",
			existing:      "customer-user",
			defaultVal:    "sla-user",
			want:          "discovered-user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveField(mf, tt.ownershipMode, tt.incoming, tt.existing, tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveField_Strategies(t *testing.T) {
	tests := []struct {
		name       string
		mf         ManagedField
		incoming   interface{}
		existing   interface{}
		defaultVal interface{}
		want       interface{}
	}{
		{
			name:     "AlwaysUseIncoming 可空字段直接采用来源值（包括空值）",
			mf:       ManagedField{FieldName: "Description", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
			incoming: "", existing: "old", defaultVal: "def",
			want: "",
		},
		{
			name:     "AlwaysUseIncoming 非空字段来源为空时回退存量",
			mf:       ManagedField{FieldName: "Status", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
			incoming: "", existing: "active", defaultVal: "planned",
			want: "active",
		},
		{
			name:     "UseIncomingOrExistingOrDefault 逐级回退",
			mf:       ManagedField{FieldName: "Model", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
			incoming: "", existing: "", defaultVal: "default-model",
			want: "default-model",
		},
		{
			name:     "UseExistingOrDefault 忽略来源值",
			mf:       ManagedField{FieldName: "Custom", Strategy: UseExistingOrDefault, CanBeEmpty: true},
			incoming: "incoming", existing: "existing", defaultVal: "def",
			want: "existing",
		},
		{
			name:     "UseSLAOrDefault 默认值优先",
			mf:       ManagedField{FieldName: "Custom", Strategy: UseSLAOrDefault, CanBeEmpty: true},
			incoming: "incoming", existing: "existing", defaultVal: "sla-default",
			want: "sla-default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveField(tt.mf, OwnershipModeManaged, tt.incoming, tt.existing, tt.defaultVal)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPickFirstNonEmpty(t *testing.T) {
	assert.Equal(t, "a", pickFirstNonEmpty("a", "b"))
	assert.Equal(t, "b", pickFirstNonEmpty("", "b"))
	assert.Equal(t, "c", pickFirstNonEmpty(nil, "", "c"))
	assert.Nil(t, pickFirstNonEmpty(nil, "", 0))
}

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		val  interface{}
		want bool
	}{
		{"nil 为空", nil, true},
		{"空字符串为空", "", true},
		{"非空字符串", "x", false},
		{"零值 int 为空", 0, true},
		{"非零 int", 1, false},
		{"空切片为空", []interface{}{}, true},
		{"非空切片", []interface{}{1}, false},
		{"空 map 为空", map[string]interface{}{}, true},
		{"非空 map", map[string]interface{}{"k": "v"}, false},
		{"bool 不视为空", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isEmpty(tt.val))
		})
	}
}

func TestValuesEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b interface{}
		want bool
	}{
		{"双 nil 相等", nil, nil, true},
		{"nil 与非 nil 不等", nil, "a", false},
		{"字符串相等", "a", "a", true},
		{"字符串不等", "a", "b", false},
		{"int 相等", 3, 3, true},
		{"int64 相等", int64(3), int64(3), true},
		{"类型不同不等", 3, int64(3), false},
		{"bool 相等", true, true, true},
		{"切片深比较相等", []interface{}{"a", 1}, []interface{}{"a", 1}, true},
		{"切片长度不同不等", []interface{}{"a"}, []interface{}{"a", 1}, false},
		{"map 深比较相等", map[string]interface{}{"k": "v"}, map[string]interface{}{"k": "v"}, true},
		{"map 值不同不等", map[string]interface{}{"k": "v"}, map[string]interface{}{"k": "x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, valuesEqual(tt.a, tt.b))
		})
	}
}

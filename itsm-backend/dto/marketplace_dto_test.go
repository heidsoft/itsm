package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/marketplaceitem"
)

// TestMarketplaceItemResponse_CamelCaseContract 锁定 API 字段命名契约：
// 前端 httpClient.get<{ items: [...] }>(...) 依赖这些 camelCase key；
// 任何 snake_case 渗入 JSON 即视为契约回归。
func TestMarketplaceItemResponse_CamelCaseContract(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	item := &ent.MarketplaceItem{
		ID:               7,
		Name:             "feishu-approval-bridge",
		Type:             marketplaceitem.TypeConnector,
		Title:            "飞书审批增强",
		Provider:         "feishu-official",
		Description:      "把 BPMN 审批任务实时同步到飞书工作台",
		LongDescription:  "完整描述",
		IconURL:          "https://example.com/icon.png",
		Tags:             []string{"feishu", "approval"},
		Rating:           4.7,
		InstallCount:     123,
		LatestVersion:    "1.2.0",
		MinSystemVersion: "v1.6.0",
		Status:           marketplaceitem.StatusPublished,
		IsOfficial:       true,
		IsFree:           true,
		Price:            0,
		Category:         "办公协同",
		Capabilities:     []string{"approval", "notification"},
		RequiredPermissions: []string{"approval:write"},
		AuthorID:         "u-001",
		AuthorName:       "Feishu Team",
		Homepage:         "https://example.com",
		Repository:       "https://git.example.com/feishu",
		License:          "Apache-2.0",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	resp := ToMarketplaceItemResponse(item)
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	payload := string(raw)

	// 禁止出现的 snake_case key —— 一旦命中即视为契约破坏。
	forbidden := []string{
		"long_description", "install_count", "icon_url",
		"latest_version", "min_system_version",
		"is_official", "is_free", "required_permissions",
		"author_id", "author_name",
	}
	for _, k := range forbidden {
		if strings.Contains(payload, `"`+k+`"`) {
			t.Fatalf("DTO JSON contains forbidden snake_case key %q: %s", k, payload)
		}
	}

	// 必出现的 camelCase key —— 任何缺失都视为前端 lookup 失败源。
	required := []string{
		"longDescription", "installCount", "iconUrl",
		"latestVersion", "minSystemVersion",
		"isOfficial", "isFree", "requiredPermissions",
		"authorId", "authorName",
	}
	for _, k := range required {
		if !strings.Contains(payload, `"`+k+`"`) {
			t.Fatalf("DTO JSON missing required camelCase key %q: %s", k, payload)
		}
	}

	if resp.Status != "published" {
		t.Fatalf("expected status=published, got %q", resp.Status)
	}
	if resp.Type != "connector" {
		t.Fatalf("expected type=connector, got %q", resp.Type)
	}
}

// TestMarketplaceItemResponse_NilGuard 显式 nil 保护：
// ListItems 在 ent.NotFound 时 Service 返回 (nil, 0, err)，Handler 需要 DTO guard
// 防止 nil 指针解引用与 panic。
func TestMarketplaceItemResponse_NilGuard(t *testing.T) {
	if got := ToMarketplaceItemResponse(nil); got != nil {
		t.Fatalf("expected nil for nil ent item, got %+v", got)
	}
	// 列表中夹杂 nil 也不应 panic
	mixed := []*ent.MarketplaceItem{nil, {
		ID:   1,
		Name: "x", Type: marketplaceitem.TypeSkill,
		Title: "x", Provider: "p", LatestVersion: "1",
		Status: marketplaceitem.StatusPublished,
	}}
	out := ToMarketplaceItemResponseList(mixed)
	if len(out) != 1 {
		t.Fatalf("expected 1 non-nil element, got %d", len(out))
	}
	if out[0].Name != "x" {
		t.Fatalf("expected first item name=x, got %q", out[0].Name)
	}
	if out[0].Type != "skill" {
		t.Fatalf("expected type=skill, got %q", out[0].Type)
	}
}

// TestMarketplaceItemResponse_OmitemptyDotOptional 锁定空字段被 omitempty 隐藏：
// - description / longDescription / iconUrl 等可选字段无值时不应进入 JSON；
// - 状态机：published 状态字符串原样透传。
func TestMarketplaceItemResponse_OmitemptyOptional(t *testing.T) {
	item := &ent.MarketplaceItem{
		ID:   99,
		Name: "dingtalk-ai-assistant",
		// Type 缺省 ""
		Title:        "钉钉 AI 助手",
		Provider:     "dingtalk-official",
		LatestVersion: "0.1.0",
		Status:       marketplaceitem.StatusPublished,
		// Description 缺省 ""
		// IconURL 缺省 ""
		// Category 缺省 ""
	}
	raw, err := json.Marshal(ToMarketplaceItemResponse(item))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	payload := string(raw)
	for _, k := range []string{"description", "longDescription", "iconUrl", "category"} {
		if strings.Contains(payload, `"`+k+`"`) {
			t.Fatalf("empty optional field %q leaked into JSON: %s", k, payload)
		}
	}
}
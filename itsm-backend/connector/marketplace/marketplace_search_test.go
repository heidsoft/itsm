package marketplace

import (
	"context"
	"testing"

	"itsm-backend/connector"
)

// docFakeC 带完整文案字段（标题/描述/供应商/标签）的连接器，
// 专门用于覆盖 marketplace 的"文案"匹配分支（Title/Name/Description/Provider/Tags）。
type docFakeC struct{}

func (f *docFakeC) Manifest() connector.Manifest {
	return connector.Manifest{
		Name:        "mailer",
		Title:       "邮件告警",
		Description: "用于发送邮件告警通知",
		Provider:    "Acme",
		Type:        connector.TypeIM,
		Tags:        []string{"mail", "alert"},
	}
}
func (f *docFakeC) Init(_ context.Context, _ connector.Config) error   { return nil }
func (f *docFakeC) Send(_ context.Context, _ *connector.Message) error { return nil }
func (f *docFakeC) HealthCheck(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{OK: true}
}
func (f *docFakeC) Close() error { return nil }

// fakeSource 模拟一个远程市场数据源
type fakeSource struct {
	manifests []connector.Manifest
}

func (s *fakeSource) Fetch() ([]connector.Manifest, error) {
	return s.manifests, nil
}

func newMarketWithMailer() *Market {
	r := connector.NewRegistry()
	r.Register(func() connector.Connector { return &docFakeC{} })
	m := New()
	_ = m.Refresh(r)
	return m
}

// TestMarketplace_Search_CopyBranches 覆盖 matchAny 里对 文案 各字段的匹配分支：
// Title / Name / Description / Provider / Tags，以及大小写不敏感与分类过滤。
func TestMarketplace_Search_CopyBranches(t *testing.T) {
	m := newMarketWithMailer()

	// Title / Description 命中（中文）
	if got := m.Search("邮件", ""); len(got) != 1 {
		t.Fatalf("search '邮件' should match Title/Description, got %d", len(got))
	}
	// Name 命中
	if got := m.Search("mailer", ""); len(got) != 1 {
		t.Fatalf("search 'mailer' should match Name, got %d", len(got))
	}
	// Provider 命中（大小写不敏感：录入 "Acme"，查询 "acme"）
	if got := m.Search("acme", ""); len(got) != 1 {
		t.Fatalf("search 'acme' should case-insensitively match Provider, got %d", len(got))
	}
	// Tags 命中（大小写不敏感：录入 "alert"，查询 "ALERT"）
	if got := m.Search("ALERT", ""); len(got) != 1 {
		t.Fatalf("search 'ALERT' should case-insensitively match Tags, got %d", len(got))
	}
	// 未命中
	if got := m.Search("nothing", ""); len(got) != 0 {
		t.Fatalf("search 'nothing' should return 0, got %d", len(got))
	}
	// 空查询返回全部
	if got := m.Search("", ""); len(got) != 1 {
		t.Fatalf("empty query should return all, got %d", len(got))
	}
	// 分类过滤命中（Refresh 把 manifest.Type 映射为 Category）
	if got := m.Search("", string(connector.TypeIM)); len(got) != 1 {
		t.Fatalf("category filter 'im' should match, got %d", len(got))
	}
	// 分类过滤未命中（查询词命中但分类不匹配 => 0）
	if got := m.Search("邮件", "webhook"); len(got) != 0 {
		t.Fatalf("category mismatch should return 0 even if query matches, got %d", len(got))
	}
}

// TestMarketplace_equalsFold 覆盖 equalsFold 的大小写归一与长度分支
func TestMarketplace_equalsFold(t *testing.T) {
	if !equalsFold("AbC", "aBc") {
		t.Fatal("equalsFold should be case-insensitive true")
	}
	if !equalsFold("ab", "ab") {
		t.Fatal("equalsFold identical should be true")
	}
	if equalsFold("abc", "ab") {
		t.Fatal("equalsFold different length should be false")
	}
}

// TestMarketplace_contains 覆盖 contains 的子串/空子串/超长分支
func TestMarketplace_contains(t *testing.T) {
	if !contains("Hello World", "WORLD") {
		t.Fatal("contains should be case-insensitive (WORLD in Hello World)")
	}
	if contains("Hello", "xyz") {
		t.Fatal("contains should return false for non-substring")
	}
	if !contains("Hello", "") {
		t.Fatal("contains empty substring should be true")
	}
	if contains("Hi", "toolong") {
		t.Fatal("contains longer-than-haystack should be false")
	}
}

// TestMarketplace_Refresh_RemoteOverride 覆盖 Refresh 中
// "远程源覆盖本地同名元信息" 与 "远程独有项标记 Local=false" 两个分支。
func TestMarketplace_Refresh_RemoteOverride(t *testing.T) {
	r := connector.NewRegistry()
	r.Register(func() connector.Connector { return &docFakeC{} })

	src := &fakeSource{manifests: []connector.Manifest{
		// 与本地同名，但 Title 被远程覆盖
		{Name: "mailer", Title: "Remote Mailer", Type: connector.TypeIM},
		// 远程独有项
		{Name: "remote-only", Title: "Remote Only", Type: connector.TypeWebhook},
	}}

	m := New()
	if err := m.Refresh(r, src); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	if e, ok := m.entries["mailer"]; !ok || e.Manifest.Title != "Remote Mailer" {
		t.Fatalf("remote should override local Title, got %+v", e)
	}
	if e, ok := m.entries["remote-only"]; !ok {
		t.Fatal("remote-only entry should be present")
	} else if e.Local {
		t.Fatal("remote-only entry should have Local=false")
	}
}

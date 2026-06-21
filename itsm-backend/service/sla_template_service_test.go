package service

import (
	"context"
	"sync"
	"testing"
)

// TestSLATemplateService_ListTemplates 验证能列出全部 6 个预置模板
func TestSLATemplateService_ListTemplates(t *testing.T) {
	s := NewSLATemplateService(nil, nil)
	templates := s.ListTemplates()

	if len(templates) != 6 {
		t.Fatalf("expected 6 templates, got %d", len(templates))
	}

	// 验证已知模板 key 存在
	expectedKeys := map[string]bool{
		"incident_p1_critical":      false,
		"incident_p2_high":          false,
		"incident_p3_medium":        false,
		"change_normal":             false,
		"change_emergency":          false,
		"service_request_standard":  false,
	}

	for _, tmpl := range templates {
		if _, ok := expectedKeys[tmpl.Key]; ok {
			expectedKeys[tmpl.Key] = true
			// 校验基础字段
			if tmpl.Name == "" {
				t.Errorf("template %s has empty name", tmpl.Key)
			}
			if tmpl.ResponseTime <= 0 {
				t.Errorf("template %s has invalid ResponseTime: %d", tmpl.Key, tmpl.ResponseTime)
			}
			if tmpl.ResolutionTime <= 0 {
				t.Errorf("template %s has invalid ResolutionTime: %d", tmpl.Key, tmpl.ResolutionTime)
			}
			if tmpl.ResponseTime > tmpl.ResolutionTime {
				t.Errorf("template %s: ResponseTime (%d) should be <= ResolutionTime (%d)",
					tmpl.Key, tmpl.ResponseTime, tmpl.ResolutionTime)
			}
		}
	}

	for key, found := range expectedKeys {
		if !found {
			t.Errorf("missing expected template key: %s", key)
		}
	}
}

// TestSLATemplateService_GetTemplate 验证 GetTemplate 能正确返回 / 报错
func TestSLATemplateService_GetTemplate(t *testing.T) {
	s := NewSLATemplateService(nil, nil)

	// 存在的 key
	tmpl, err := s.GetTemplate("incident_p1_critical")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tmpl.Key != "incident_p1_critical" {
		t.Errorf("expected key 'incident_p1_critical', got %q", tmpl.Key)
	}
	if tmpl.Priority != "critical" {
		t.Errorf("expected priority 'critical', got %q", tmpl.Priority)
	}
	if tmpl.ResponseTime != 15 {
		t.Errorf("expected ResponseTime=15, got %d", tmpl.ResponseTime)
	}
	if tmpl.ResolutionTime != 240 {
		t.Errorf("expected ResolutionTime=240, got %d", tmpl.ResolutionTime)
	}

	// 不存在的 key
	_, err = s.GetTemplate("not_existing_template")
	if err == nil {
		t.Error("expected error for non-existing template, got nil")
	}
}

// TestSLATemplateService_GetTemplate_EdgeCase 验证 key 为空/特殊字符
func TestSLATemplateService_GetTemplate_EdgeCase(t *testing.T) {
	s := NewSLATemplateService(nil, nil)

	cases := []string{"", "INVALID_KEY", "incident_p1", "../etc/passwd"}
	for _, c := range cases {
		_, err := s.GetTemplate(c)
		if err == nil {
			t.Errorf("expected error for key=%q, got nil", c)
		}
	}
}

// TestSLATemplateService_IsInstalled 验证 IsInstalled 在未安装/已安装场景下的返回
func TestSLATemplateService_IsInstalled(t *testing.T) {
	s := NewSLATemplateService(nil, nil)

	if s.IsInstalled(1, "incident_p1_critical") {
		t.Error("expected false before install")
	}

	// 模拟安装状态
	s.mu.Lock()
	s.installed["1:incident_p1_critical"] = 42
	s.mu.Unlock()

	if !s.IsInstalled(1, "incident_p1_critical") {
		t.Error("expected true after manual install record")
	}

	// 不同租户应互不影响
	if s.IsInstalled(2, "incident_p1_critical") {
		t.Error("tenant 2 should not see tenant 1 install record")
	}
}

// TestSLATemplateService_ConcurrentAccess 验证并发安全
func TestSLATemplateService_ConcurrentAccess(t *testing.T) {
	s := NewSLATemplateService(nil, nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = s.ListTemplates()
			_, _ = s.GetTemplate("incident_p1_critical")
			_ = s.IsInstalled(i, "incident_p1_critical")
		}(i)
	}
	wg.Wait()
}

// TestSLATemplateService_EscalationRules 验证升级规则矩阵覆盖 P1/P2/P3
func TestSLATemplateService_EscalationRules(t *testing.T) {
	s := NewSLATemplateService(nil, nil)
	templates := s.ListTemplates()

	for _, tmpl := range templates {
		levels, ok := tmpl.EscalationRules["levels"].([]map[string]interface{})
		if !ok {
			t.Errorf("template %s: escalation levels not found", tmpl.Key)
			continue
		}
		if len(levels) == 0 {
			t.Errorf("template %s: escalation levels is empty", tmpl.Key)
		}
		for i, lvl := range levels {
			if _, ok := lvl["level"]; !ok {
				t.Errorf("template %s: level[%d] missing 'level'", tmpl.Key, i)
			}
			if _, ok := lvl["threshold_minutes"]; !ok {
				t.Errorf("template %s: level[%d] missing 'threshold_minutes'", tmpl.Key, i)
			}
			if _, ok := lvl["notify_roles"]; !ok {
				t.Errorf("template %s: level[%d] missing 'notify_roles'", tmpl.Key, i)
			}
		}
	}
}

// TestSLATemplateService_InstallTemplate_NilClient 验证 nil client 调用安全
//（用于覆盖 service 不依赖 DB 的方法）
func TestSLATemplateService_InstallTemplate_NilClient(t *testing.T) {
	s := NewSLATemplateService(nil, nil)

	// 未知 key 应直接报错
	_, err := s.InstallTemplate(context.Background(), "not_existing", 1)
	if err == nil {
		t.Error("expected error for non-existing template key")
	}
}
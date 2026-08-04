package bpmn

import "testing"

func TestValidateWebhookURLBlocksSSRF(t *testing.T) {
	blocked := []string{
		"file:///etc/passwd",
		"http://localhost/admin",
		"http://127.0.0.1/admin",
		"http://10.0.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]/",
	}
	for _, target := range blocked {
		if err := validateWebhookURL(target); err == nil {
			t.Errorf("expected %q to be blocked", target)
		}
	}
	if err := validateWebhookURL("https://hooks.example.com/event"); err != nil {
		t.Fatalf("expected public HTTPS URL to be accepted: %v", err)
	}
}

package config

import (
	"testing"
)

func TestLoadContestDefaultsDoNotRequireModelCredentials(t *testing.T) {
	for _, key := range []string{
		"GEMINI_API_KEY", "GOOGLE_CLOUD_PROJECT", "GCP_PROJECT", "FLOWPROOF_VERTEX_AI", "FLOWPROOF_DEMO_MODE",
	} {
		t.Setenv(key, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() requires an unrelated model credential: %v", err)
	}
	if cfg.Port == "" || cfg.StatePath == "" || cfg.StepTimeout <= 0 || cfg.RunTimeout <= 0 {
		t.Fatalf("invalid contest defaults: %#v", cfg)
	}
}

func TestValidateTargetAllowsOnlyBuiltInOriginOrExactAllowlist(t *testing.T) {
	serverOrigin := "http://127.0.0.1:8080"
	allowed := []string{"example.com"}

	for _, raw := range []string{
		"http://127.0.0.1:8080/demo-store",
		"https://example.com/checkout",
	} {
		if err := ValidateTarget(raw, serverOrigin, allowed); err != nil {
			t.Fatalf("ValidateTarget(%q) = %v, want allowed", raw, err)
		}
	}

	for _, raw := range []string{
		"ftp://example.com/file",
		"https://user:secret@example.com/",
		"http://127.0.0.1:9090/demo-store",
		"http://localhost:8080/demo-store",
		"http://10.0.0.8/internal",
		"http://169.254.169.254/latest/meta-data",
		"https://sub.example.com/checkout",
		"https://not-example.com/checkout",
	} {
		if err := ValidateTarget(raw, serverOrigin, allowed); err == nil {
			t.Fatalf("ValidateTarget(%q) unexpectedly allowed", raw)
		}
	}
}

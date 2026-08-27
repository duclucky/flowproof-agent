package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port         string
	StatePath    string
	ChromePath   string
	WebDir       string
	AllowedHosts []string
	StepTimeout  time.Duration
	RunTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:        getenv("PORT", "8080"),
		StatePath:   getenv("FLOWPROOF_STATE_PATH", "data/runs.json"),
		ChromePath:  strings.TrimSpace(os.Getenv("FLOWPROOF_CHROME_PATH")),
		WebDir:      getenv("FLOWPROOF_WEB_DIR", "web/dist"),
		StepTimeout: 4 * time.Second,
		RunTimeout:  45 * time.Second,
	}

	var err error
	if cfg.StepTimeout, err = envDuration("FLOWPROOF_STEP_TIMEOUT", cfg.StepTimeout); err != nil {
		return Config{}, err
	}
	if cfg.RunTimeout, err = envDuration("FLOWPROOF_RUN_TIMEOUT", cfg.RunTimeout); err != nil {
		return Config{}, err
	}
	if raw := strings.TrimSpace(os.Getenv("FLOWPROOF_ALLOWED_HOSTS")); raw != "" {
		seen := make(map[string]struct{})
		for _, host := range strings.Split(raw, ",") {
			host = strings.ToLower(strings.TrimSpace(host))
			if host == "" {
				continue
			}
			if _, ok := seen[host]; ok {
				continue
			}
			seen[host] = struct{}{}
			cfg.AllowedHosts = append(cfg.AllowedHosts, host)
		}
	}
	return cfg, nil
}

func ValidateTarget(rawURL, serverOrigin string, allowedHosts []string) error {
	target, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || target.Scheme == "" || target.Host == "" {
		return fmt.Errorf("invalid target URL")
	}
	if target.Scheme != "http" && target.Scheme != "https" {
		return fmt.Errorf("target URL must use http or https")
	}
	if target.User != nil {
		return fmt.Errorf("target URL must not contain credentials")
	}

	origin, err := url.Parse(strings.TrimSpace(serverOrigin))
	if err == nil && origin.Scheme != "" && origin.Host != "" && sameOrigin(target, origin) {
		return nil
	}

	host := strings.ToLower(strings.TrimSuffix(target.Hostname(), "."))
	if host == "" || unsafeHost(host) {
		return fmt.Errorf("target host is not permitted")
	}
	for _, allowed := range allowedHosts {
		if host == strings.ToLower(strings.TrimSuffix(strings.TrimSpace(allowed), ".")) {
			return nil
		}
	}
	return fmt.Errorf("target host %q is not in FLOWPROOF_ALLOWED_HOSTS", host)
}

func sameOrigin(a, b *url.URL) bool {
	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func unsafeHost(host string) bool {
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() || !ip.IsGlobalUnicast()
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive Go duration", key)
	}
	return value, nil
}

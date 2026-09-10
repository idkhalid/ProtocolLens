package config

import (
	"reflect"
	"testing"
	"time"
)

func TestReplayConfigDefaultsAndSafeFallbacks(t *testing.T) {
	t.Setenv("PROTOCOLLENS_REPLAY_ENABLED", "definitely-not-true")
	t.Setenv("PROTOCOLLENS_REPLAY_ALLOWED_PORTS", "bad")
	t.Setenv("PROTOCOLLENS_REPLAY_TIMEOUT", "999h")
	t.Setenv("PROTOCOLLENS_REPLAY_MAX_REQUEST_SIZE", "-1")
	t.Setenv("PROTOCOLLENS_REPLAY_MAX_RESPONSE_SIZE", "0")
	t.Setenv("PROTOCOLLENS_REPLAY_MAX_CONCURRENT", "-1")
	t.Setenv("PROTOCOLLENS_LOCAL_CAPTURE_ENABLED", "definitely-not-true")

	cfg := Load()
	if cfg.ReplayEnabled || !reflect.DeepEqual(cfg.ReplayAllowedPorts, []uint16{80, 443}) || cfg.ReplayTimeout != 60*time.Second || cfg.ReplayMaxRequestBytes != 1<<20 || cfg.ReplayMaxResponseBytes != 2<<20 || cfg.ReplayMaxConcurrent != 4 || cfg.LocalCaptureEnabled {
		t.Fatalf("replay config = %#v", cfg)
	}
}

func TestBooleanConfigSemantics(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"false", false},
		{"TRUE", true},
		{"FALSE", false},
		{"1", true},
		{"0", false},
		{"yes", true},
		{"no", false},
		{"garbage", false},
		{"", false},
	} {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("PROTOCOLLENS_REPLAY_ENABLED", tt.value)
			t.Setenv("PROTOCOLLENS_LOCAL_CAPTURE_ENABLED", tt.value)
			cfg := Load()
			if cfg.ReplayEnabled != tt.want || cfg.LocalCaptureEnabled != tt.want {
				t.Fatalf("value %q: replay=%v capture=%v want=%v", tt.value, cfg.ReplayEnabled, cfg.LocalCaptureEnabled, tt.want)
			}
		})
	}
}

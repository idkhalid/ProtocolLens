package config

import (
	"reflect"
	"testing"
	"time"
)

func TestReplayConfigDefaultsAndSafeFallbacks(t *testing.T) {
	t.Setenv("PROTOCOLLENS_REPLAY_ENABLED", "")
	t.Setenv("PROTOCOLLENS_REPLAY_ALLOWED_PORTS", "bad")
	t.Setenv("PROTOCOLLENS_REPLAY_TIMEOUT", "999h")
	t.Setenv("PROTOCOLLENS_REPLAY_MAX_REQUEST_SIZE", "-1")
	t.Setenv("PROTOCOLLENS_REPLAY_MAX_RESPONSE_SIZE", "0")
	t.Setenv("PROTOCOLLENS_REPLAY_MAX_CONCURRENT", "-1")

	cfg := Load()
	if cfg.ReplayEnabled || !reflect.DeepEqual(cfg.ReplayAllowedPorts, []uint16{80, 443}) || cfg.ReplayTimeout != 60*time.Second || cfg.ReplayMaxRequestBytes != 1<<20 || cfg.ReplayMaxResponseBytes != 2<<20 || cfg.ReplayMaxConcurrent != 4 {
		t.Fatalf("replay config = %#v", cfg)
	}
}

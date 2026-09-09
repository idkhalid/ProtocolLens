package config

import "testing"

func TestLocalCaptureBindAllowed(t *testing.T) {
	allowed := []string{"localhost", "localhost:8080", "127.0.0.1", "127.9.8.7:8080", "[::1]", "[::1]:8080"}
	for _, addr := range allowed {
		if !LocalCaptureBindAllowed(addr) {
			t.Fatalf("expected %q to be allowed", addr)
		}
	}
	rejected := []string{"", ":8080", "0.0.0.0", "0.0.0.0:8080", "::", "[::]:8080", "192.168.1.10:8080", "8.8.8.8:8080", "example.com:8080"}
	for _, addr := range rejected {
		if LocalCaptureBindAllowed(addr) {
			t.Fatalf("expected %q to be rejected", addr)
		}
	}
}

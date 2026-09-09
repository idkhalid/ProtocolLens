package replay

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type resolver map[string][]net.IPAddr

func (r resolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r["host"], nil
}

func TestDestinationPolicyBlocksUnsafeDestinations(t *testing.T) {
	policy := NewDestinationPolicy([]uint16{80, 443})
	for _, raw := range []string{
		"http://127.0.0.1",
		"http://localhost",
		"http://10.0.0.1",
		"http://172.16.0.1",
		"http://192.168.1.1",
		"http://169.254.169.254",
		"http://100.64.0.1",
		"http://0.1.2.3",
		"http://192.0.2.1",
		"http://198.18.0.1",
		"http://203.0.113.1",
		"http://240.0.0.1",
		"http://[::1]",
		"http://[::ffff:127.0.0.1]",
	} {
		addr, _ := netip.ParseAddr(host(raw))
		if addr.IsValid() && policy.ValidateAddr(addr) == nil {
			t.Fatalf("allowed unsafe address %s", raw)
		}
	}
}

func TestDestinationPolicyValidatesURL(t *testing.T) {
	policy := NewDestinationPolicy([]uint16{80, 443})
	policy.Resolver = resolver{"host": {{IP: net.ParseIP("93.184.216.34")}}}
	if _, err := policy.ValidateURL(context.Background(), "https://example.com/api"); err != nil {
		t.Fatalf("public URL rejected: %v", err)
	}
	cases := map[string]error{
		"file:///etc/passwd":        ErrUnsupportedScheme,
		"https://u:p@example.com/":  ErrUserinfo,
		"https://example.com:8443/": ErrPortBlocked,
		"https://localhost/":        ErrLocalHostname,
	}
	for raw, want := range cases {
		if _, err := policy.ValidateURL(context.Background(), raw); !errors.Is(err, want) {
			t.Fatalf("%s error = %v, want %v", raw, err, want)
		}
	}
}

func TestDestinationPolicyRejectsMixedDNSResults(t *testing.T) {
	policy := NewDestinationPolicy([]uint16{443})
	policy.Resolver = resolver{"host": {{IP: net.ParseIP("93.184.216.34")}, {IP: net.ParseIP("10.0.0.1")}}}
	if _, err := policy.ValidateURL(context.Background(), "https://example.com/"); !errors.Is(err, ErrDestinationBlocked) {
		t.Fatalf("mixed DNS error = %v", err)
	}
}

func host(raw string) string {
	for _, prefix := range []string{"http://", "https://"} {
		raw = strings.TrimPrefix(raw, prefix)
	}
	raw, _, _ = strings.Cut(raw, "/")
	return strings.Trim(raw, "[]")
}

func TestReplayTransportDisablesEnvironmentProxyAndCompression(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	transport := NewDestinationPolicy([]uint16{443}).Transport(time.Second)
	if transport.Proxy != nil || !transport.DisableCompression || transport.TLSHandshakeTimeout != time.Second || transport.ResponseHeaderTimeout != time.Second {
		t.Fatalf("transport = %#v", transport)
	}
}

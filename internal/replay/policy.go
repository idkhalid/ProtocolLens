package replay

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrUnsupportedScheme  = errors.New("unsupported_scheme")
	ErrUserinfo           = errors.New("invalid_userinfo")
	ErrLocalHostname      = errors.New("destination_blocked")
	ErrPortBlocked        = errors.New("port_blocked")
	ErrDestinationBlocked = errors.New("destination_blocked")
)

type DestinationPolicy struct {
	AllowedPorts map[uint16]struct{}
	Resolver     Resolver
}

type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

func NewDestinationPolicy(ports []uint16) DestinationPolicy {
	allowed := make(map[uint16]struct{}, len(ports))
	for _, port := range ports {
		allowed[port] = struct{}{}
	}
	return DestinationPolicy{AllowedPorts: allowed, Resolver: net.DefaultResolver}
}

func (p DestinationPolicy) ValidateURL(ctx context.Context, raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" {
		return nil, ErrUnsupportedScheme
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrUnsupportedScheme
	}
	if u.Host == "" {
		return nil, ErrDestinationBlocked
	}
	if u.User != nil {
		return nil, ErrUserinfo
	}
	if isLocalHostname(u.Hostname()) {
		return nil, ErrLocalHostname
	}
	if _, ok := p.AllowedPorts[defaultPort(u)]; !ok {
		return nil, ErrPortBlocked
	}
	return u, p.ValidateHost(ctx, u.Hostname())
}

func (p DestinationPolicy) ValidateHost(ctx context.Context, host string) error {
	if isLocalHostname(host) {
		return ErrLocalHostname
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return p.ValidateAddr(addr)
	}
	resolver := p.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addrs, err := resolver.LookupIPAddr(ctx, host)
	if err != nil || len(addrs) == 0 {
		return ErrDestinationBlocked
	}
	for _, addr := range addrs {
		ip, ok := netip.AddrFromSlice(addr.IP)
		if !ok || !SafeAddr(ip.Unmap()) {
			return ErrDestinationBlocked
		}
	}
	return nil
}

func (p DestinationPolicy) ValidateAddr(addr netip.Addr) error {
	if !SafeAddr(addr.Unmap()) {
		return ErrDestinationBlocked
	}
	return nil
}

func SafeAddr(addr netip.Addr) bool {
	if !addr.IsValid() || addr.IsUnspecified() || addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || addr.IsMulticast() {
		return false
	}
	if inPrefix(addr, "0.0.0.0/8") || inPrefix(addr, "100.64.0.0/10") || inPrefix(addr, "169.254.0.0/16") || inPrefix(addr, "192.0.0.0/24") || inPrefix(addr, "192.0.2.0/24") || inPrefix(addr, "198.18.0.0/15") || inPrefix(addr, "198.51.100.0/24") || inPrefix(addr, "203.0.113.0/24") || inPrefix(addr, "240.0.0.0/4") || inPrefix(addr, "fc00::/7") || inPrefix(addr, "fe80::/10") || inPrefix(addr, "ff00::/8") {
		return false
	}
	return true
}

func Port(rawURL string) (uint16, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0, err
	}
	return defaultPort(u), nil
}

func defaultPort(u *url.URL) uint16 {
	if port := u.Port(); port != "" {
		parsed, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return 0
		}
		return uint16(parsed)
	}
	if u.Scheme == "http" {
		return 80
	}
	if u.Scheme == "https" {
		return 443
	}
	return 0
}

func (p DestinationPolicy) Transport(connectTimeout time.Duration) *http.Transport {
	dialer := &net.Dialer{Timeout: connectTimeout}
	return &http.Transport{
		Proxy:                 nil,
		DisableCompression:    true,
		TLSHandshakeTimeout:   connectTimeout,
		ResponseHeaderTimeout: connectTimeout,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, ErrDestinationBlocked
			}
			parsedPort, err := strconv.ParseUint(port, 10, 16)
			if err != nil {
				return nil, ErrPortBlocked
			}
			if _, ok := p.AllowedPorts[uint16(parsedPort)]; !ok {
				return nil, ErrPortBlocked
			}
			resolver := p.Resolver
			if resolver == nil {
				resolver = net.DefaultResolver
			}
			addrs, err := resolver.LookupIPAddr(ctx, host)
			if err != nil || len(addrs) == 0 {
				return nil, ErrDestinationBlocked
			}
			for _, addr := range addrs {
				ip, ok := netip.AddrFromSlice(addr.IP)
				if !ok || !SafeAddr(ip.Unmap()) {
					return nil, ErrDestinationBlocked
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].IP.String(), port))
		},
	}
}

func inPrefix(addr netip.Addr, cidr string) bool {
	prefix := netip.MustParsePrefix(cidr)
	return prefix.Contains(addr)
}

func isLocalHostname(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	return host == "localhost" || strings.HasSuffix(host, ".localhost")
}

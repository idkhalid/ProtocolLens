package generator

import (
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

type Target string

const (
	TargetCurl   Target = "curl"
	TargetPython Target = "python"
	TargetGo     Target = "go"
)

type Input struct {
	Method      string
	URL         string
	Query       url.Values
	Headers     http.Header
	Body        string
	BodyStruct  any
	ContentType string
}

type Output struct {
	Target               Target   `json:"target"`
	Language             string   `json:"language"`
	Code                 string   `json:"code"`
	EnvironmentVariables []string `json:"environmentVariables"`
}

var ErrUnsupportedTarget = errors.New("unsupported_target")

// EnvName computes a deterministic environment variable name.
func EnvName(name string) string {
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ToUpper(name)
	return name
}

// FilterHeaders removes transport/browser noise and T5-forbidden headers.
func FilterHeaders(h http.Header) http.Header {
	clean := http.Header{}
	for k, v := range h {
		k = http.CanonicalHeaderKey(k)
		// Explicit small policy for noise/forbidden
		switch k {
		case "Host", "Proxy-Authorization", "Proxy-Connection", "Keep-Alive", "Te", "Trailer", "Upgrade",
			"Priority", "Upgrade-Insecure-Requests", "Accept-Encoding", "Content-Length", "Connection", "Transfer-Encoding", "Dnt":
			continue
		}
		if strings.HasPrefix(k, "Sec-Fetch-") || strings.HasPrefix(k, "Sec-Ch-Ua") {
			continue
		}
		clean[k] = v
	}
	return clean
}

// SortedHeaders returns headers deterministically.
func SortedHeaders(h http.Header) [][2]string {
	var keys []string
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out [][2]string
	for _, k := range keys {
		out = append(out, [2]string{k, h.Get(k)})
	}
	return out
}

func AppendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

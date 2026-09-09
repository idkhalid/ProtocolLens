package curl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"protocollens/internal/generator"
	"protocollens/internal/replay"
)

func Generate(in generator.Input) (generator.Output, error) {
	var envs []string
	var out strings.Builder

	out.WriteString("curl \\\n")
	if in.Method != "GET" {
		out.WriteString(fmt.Sprintf("  --request %s \\\n", in.Method))
	}

	u, _ := url.Parse(in.URL)
	query := u.Query()
	for k, v := range in.Query {
		query[k] = v
	}

	for k, vals := range query {
		for i, v := range vals {
			if v == replay.Redacted {
				env := generator.EnvName(k)
				envs = generator.AppendUnique(envs, env)
				query[k][i] = fmt.Sprintf("${%s}", env)
			}
		}
	}
	u.RawQuery = query.Encode()

	// url.Values.Encode() escapes $ and { }. Unescape them for shell variables
	rawURL := u.String()
	rawURL = strings.ReplaceAll(rawURL, "%24%7B", "${")
	rawURL = strings.ReplaceAll(rawURL, "%7D", "}")

	out.WriteString(fmt.Sprintf("  %s \\\n", shellEscape(rawURL)))

	headers := generator.FilterHeaders(in.Headers)
	for _, h := range generator.SortedHeaders(headers) {
		val := h[1]
		if strings.Contains(val, replay.Redacted) {
			env := generator.EnvName(h[0])
			envs = generator.AppendUnique(envs, env)
			val = strings.ReplaceAll(val, replay.Redacted, fmt.Sprintf("${%s}", env))
		}
		out.WriteString(fmt.Sprintf("  --header %s \\\n", shellEscape(fmt.Sprintf("%s: %s", h[0], val))))
	}

	if in.BodyStruct != nil {
		switch v := in.BodyStruct.(type) {
		case url.Values:
			// Deterministic form values
			var keys []string
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, name := range keys {
				for _, item := range v[name] {
					if item == replay.Redacted {
						env := generator.EnvName(name)
						envs = generator.AppendUnique(envs, env)
						item = fmt.Sprintf("${%s}", env)
					}
					out.WriteString(fmt.Sprintf("  --data-urlencode %s \\\n", shellEscape(fmt.Sprintf("%s=%s", name, item))))
				}
			}
		default:
			replacedJSON, newEnvs := replaceJSON(v)
			for _, e := range newEnvs {
				envs = generator.AppendUnique(envs, e)
			}
			out.WriteString(fmt.Sprintf("  --data %s \\\n", shellEscape(replacedJSON)))
		}
	} else if in.Body != "" {
		out.WriteString(fmt.Sprintf("  --data %s \\\n", shellEscape(in.Body)))
	}

	res := strings.TrimSuffix(out.String(), " \\\n")

	if envs == nil {
		envs = []string{}
	}
	sort.Strings(envs)

	return generator.Output{
		Target:               generator.TargetCurl,
		Language:             "bash",
		Code:                 res,
		EnvironmentVariables: envs,
	}, nil
}

// shellEscape safely escapes strings for POSIX shell.
// If it contains a placeholder like ${VAR}, it uses double quotes and escapes $, `, ", \ appropriately,
// but leaves the placeholder intact.
// If it doesn't contain placeholders, it uses single quotes.
func shellEscape(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.Contains(s, "${") {
		return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
	}
	
	// Has variables. Use double quotes. Escape \, ", `, and $ (except when part of ${...})
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '"', '`':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '$':
			// Check if it's our placeholder
			if i+1 < len(s) && s[i+1] == '{' {
				// It's likely our placeholder. Let it through without escaping.
				b.WriteByte('$')
			} else {
				// Escape literal $
				b.WriteByte('\\')
				b.WriteByte('$')
			}
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func replaceJSON(val any) (string, []string) {
	var envs []string
	var walk func(v any) any
	walk = func(v any) any {
		switch x := v.(type) {
		case map[string]any:
			m := make(map[string]any)
			for k, val := range x {
				if val == replay.Redacted {
					env := generator.EnvName(k)
					envs = generator.AppendUnique(envs, env)
					m[k] = fmt.Sprintf("${%s}", env)
				} else {
					m[k] = walk(val)
				}
			}
			return m
		case []any:
			a := make([]any, len(x))
			for i, val := range x {
				a[i] = walk(val)
			}
			return a
		default:
			return x
		}
	}
	res := walk(val)
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(res)
	return strings.TrimSuffix(out.String(), "\n"), envs
}

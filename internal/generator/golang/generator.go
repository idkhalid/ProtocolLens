package golang

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"net/url"
	"sort"
	"strings"

	"protocollens/internal/generator"
	"protocollens/internal/replay"
)

func Generate(in generator.Input) (generator.Output, error) {
	var envs []string
	var imports []string
	addImport := func(pkg string) {
		imports = generator.AppendUnique(imports, pkg)
	}

	addImport("net/http")
	addImport("log")

	var buf strings.Builder

	buf.WriteString("func main() {\n")

	u, _ := url.Parse(in.URL)
	query := u.Query()
	for k, v := range in.Query {
		query[k] = v
	}

	if len(query) > 0 {
		addImport("net/url")
		buf.WriteString("	params := url.Values{}\n")
		var qKeys []string
		for k := range query {
			qKeys = append(qKeys, k)
		}
		sort.Strings(qKeys)

		for _, k := range qKeys {
			for _, v := range query[k] {
				if v == replay.Redacted {
					env := generator.EnvName(k)
					envs = generator.AppendUnique(envs, env)
					addImport("os")
					buf.WriteString(fmt.Sprintf("	params.Add(%q, os.Getenv(%q))\n", k, env))
				} else {
					buf.WriteString(fmt.Sprintf("	params.Add(%q, %q)\n", k, v))
				}
			}
		}
		u.RawQuery = ""
		buf.WriteString(fmt.Sprintf("	rawURL := %q + \"?\" + params.Encode()\n\n", u.String()))
	} else {
		buf.WriteString(fmt.Sprintf("	rawURL := %q\n\n", u.String()))
	}

	var bodyCreation string
	var reqBodyArg = "nil"

	if in.BodyStruct != nil {
		switch v := in.BodyStruct.(type) {
		case url.Values:
			addImport("net/url")
			addImport("strings")
			buf.WriteString("	values := url.Values{}\n")
			var keys []string
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				for _, item := range v[k] {
					if item == replay.Redacted {
						env := generator.EnvName(k)
						envs = generator.AppendUnique(envs, env)
						addImport("os")
						buf.WriteString(fmt.Sprintf("	values.Add(%q, os.Getenv(%q))\n", k, env))
					} else {
						buf.WriteString(fmt.Sprintf("	values.Add(%q, %q)\n", k, item))
					}
				}
			}
			buf.WriteString("	bodyReader := strings.NewReader(values.Encode())\n\n")
			reqBodyArg = "bodyReader"
		default:
			code, newEnvs := goDictCode(v, 1)
			for _, e := range newEnvs {
				envs = generator.AppendUnique(envs, e)
			}
			if len(newEnvs) > 0 {
				addImport("os")
			}
			buf.WriteString(fmt.Sprintf("	payload := %s\n", code))
			addImport("encoding/json")
			addImport("bytes")
			buf.WriteString("	bodyBytes, err := json.Marshal(payload)\n")
			buf.WriteString("	if err != nil {\n		log.Fatal(err)\n	}\n")
			buf.WriteString("	bodyReader := bytes.NewReader(bodyBytes)\n\n")
			reqBodyArg = "bodyReader"
		}
	} else if in.Body != "" {
		addImport("strings")
		buf.WriteString(fmt.Sprintf("	bodyReader := strings.NewReader(%q)\n\n", in.Body))
		reqBodyArg = "bodyReader"
	}
	buf.WriteString(bodyCreation)

	buf.WriteString(fmt.Sprintf("	req, err := http.NewRequest(%q, rawURL, %s)\n", in.Method, reqBodyArg))
	buf.WriteString("	if err != nil {\n		log.Fatal(err)\n	}\n\n")

	headers := generator.FilterHeaders(in.Headers)
	for _, h := range generator.SortedHeaders(headers) {
		val := h[1]
		if strings.Contains(val, replay.Redacted) {
			env := generator.EnvName(h[0])
			envs = generator.AppendUnique(envs, env)
			addImport("os")
			
			// Simple naive replacement for header
			parts := strings.SplitN(val, replay.Redacted, 2)
			if parts[0] == "" && parts[1] == "" {
				buf.WriteString(fmt.Sprintf("	req.Header.Set(%q, os.Getenv(%q))\n", h[0], env))
			} else if parts[1] == "" {
				buf.WriteString(fmt.Sprintf("	req.Header.Set(%q, %q + os.Getenv(%q))\n", h[0], parts[0], env))
			} else {
				buf.WriteString(fmt.Sprintf("	req.Header.Set(%q, %q + os.Getenv(%q) + %q)\n", h[0], parts[0], env, parts[1]))
			}
		} else {
			buf.WriteString(fmt.Sprintf("	req.Header.Set(%q, %q)\n", h[0], val))
		}
	}

	addImport("io")
	buf.WriteString("\n	client := &http.Client{}\n")
	buf.WriteString("	resp, err := client.Do(req)\n")
	buf.WriteString("	if err != nil {\n		log.Fatal(err)\n	}\n")
	buf.WriteString("	defer resp.Body.Close()\n\n")
	buf.WriteString("	bodyText, err := io.ReadAll(resp.Body)\n")
	buf.WriteString("	if err != nil {\n		log.Fatal(err)\n	}\n")
	buf.WriteString("	log.Printf(\"Status: %s\", resp.Status)\n")
	buf.WriteString("	log.Printf(\"Body: %s\", bodyText)\n")
	buf.WriteString("}\n")

	sort.Strings(imports)
	var finalCode strings.Builder
	finalCode.WriteString("package main\n\nimport (\n")
	for _, imp := range imports {
		finalCode.WriteString(fmt.Sprintf("\t%q\n", imp))
	}
	finalCode.WriteString(")\n\n")
	finalCode.WriteString(buf.String())

	formatted, err := format.Source([]byte(finalCode.String()))
	if err != nil {
		return generator.Output{}, err
	}

	if envs == nil {
		envs = []string{}
	}
	sort.Strings(envs)

	return generator.Output{
		Target:               generator.TargetGo,
		Language:             "go",
		Code:                 string(formatted),
		EnvironmentVariables: envs,
	}, nil
}

func goDictCode(v any, indent int) (string, []string) {
	var envs []string
	prefix := strings.Repeat("\t", indent)

	switch x := v.(type) {
	case map[string]any:
		if len(x) == 0 {
			return "map[string]any{}", envs
		}
		var keys []string
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var b bytes.Buffer
		b.WriteString("map[string]any{\n")
		for _, k := range keys {
			val := x[k]
			if val == replay.Redacted {
				env := generator.EnvName(k)
				envs = generator.AppendUnique(envs, env)
				b.WriteString(fmt.Sprintf("%s\t%q: os.Getenv(%q),\n", prefix, k, env))
			} else {
				subCode, subEnvs := goDictCode(val, indent+1)
				for _, e := range subEnvs {
					envs = generator.AppendUnique(envs, e)
				}
				b.WriteString(fmt.Sprintf("%s\t%q: %s,\n", prefix, k, subCode))
			}
		}
		b.WriteString(prefix + "}")
		return b.String(), envs
	case []any:
		if len(x) == 0 {
			return "[]any{}", envs
		}
		var b bytes.Buffer
		b.WriteString("[]any{\n")
		for _, val := range x {
			subCode, subEnvs := goDictCode(val, indent+1)
			for _, e := range subEnvs {
				envs = generator.AppendUnique(envs, e)
			}
			b.WriteString(fmt.Sprintf("%s\t%s,\n", prefix, subCode))
		}
		b.WriteString(prefix + "}")
		return b.String(), envs
	case string:
		return fmt.Sprintf("%q", x), envs
	default:
		// numbers, booleans, null
		if x == nil {
			return "nil", envs
		}
		b, _ := json.Marshal(x)
		return string(b), envs
	}
}

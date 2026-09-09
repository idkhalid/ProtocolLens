package python

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

	out.WriteString("import httpx\n")

	// Determine if we need 'os'
	needsOS := false
	
	// Collect query params and envs
	var paramsCode []string
	u, _ := url.Parse(in.URL)
	query := u.Query()
	for k, v := range in.Query {
		query[k] = v
	}

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
				needsOS = true
				paramsCode = append(paramsCode, fmt.Sprintf("    %q: os.environ[%q],", k, env))
			} else {
				paramsCode = append(paramsCode, fmt.Sprintf("    %q: %q,", k, v))
			}
		}
	}
	u.RawQuery = ""

	headers := generator.FilterHeaders(in.Headers)
	var headerCode []string
	for _, h := range generator.SortedHeaders(headers) {
		val := h[1]
		if strings.Contains(val, replay.Redacted) {
			env := generator.EnvName(h[0])
			envs = generator.AppendUnique(envs, env)
			needsOS = true
			valCode := fmt.Sprintf(`f%q`, strings.ReplaceAll(val, replay.Redacted, fmt.Sprintf("{os.environ['%s']}", env)))
			headerCode = append(headerCode, fmt.Sprintf("    %q: %s,", h[0], valCode))
		} else {
			headerCode = append(headerCode, fmt.Sprintf("    %q: %q,", h[0], val))
		}
	}

	var bodyCode string
	bodyKwarg := ""
	if in.BodyStruct != nil {
		switch v := in.BodyStruct.(type) {
		case url.Values:
			var keys []string
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var formCode []string
			for _, k := range keys {
				for _, item := range v[k] {
					if item == replay.Redacted {
						env := generator.EnvName(k)
						envs = generator.AppendUnique(envs, env)
						needsOS = true
						formCode = append(formCode, fmt.Sprintf("    %q: os.environ[%q],", k, env))
					} else {
						formCode = append(formCode, fmt.Sprintf("    %q: %q,", k, item))
					}
				}
			}
			if len(formCode) > 0 {
				bodyCode = "data = {\n" + strings.Join(formCode, "\n") + "\n}\n"
				bodyKwarg = ", data=data"
			}
		default:
			code, newEnvs := pythonDictCode(v, 1)
			for _, e := range newEnvs {
				envs = generator.AppendUnique(envs, e)
			}
			if len(newEnvs) > 0 {
				needsOS = true
			}
			bodyCode = "payload = " + code + "\n"
			bodyKwarg = ", json=payload"
		}
	} else if in.Body != "" {
		bodyCode = fmt.Sprintf("content = %q\n", in.Body)
		bodyKwarg = ", content=content"
	}

	if needsOS {
		out.WriteString("import os\n")
	}
	out.WriteString("\n")

	out.WriteString(fmt.Sprintf("url = %q\n\n", u.String()))

	if len(paramsCode) > 0 {
		out.WriteString("params = {\n" + strings.Join(paramsCode, "\n") + "\n}\n\n")
	}
	if len(headerCode) > 0 {
		out.WriteString("headers = {\n" + strings.Join(headerCode, "\n") + "\n}\n\n")
	}
	if bodyCode != "" {
		out.WriteString(bodyCode + "\n")
	}

	args := []string{"url"}
	if len(paramsCode) > 0 {
		args = append(args, "params=params")
	}
	if len(headerCode) > 0 {
		args = append(args, "headers=headers")
	}
	args = append(args, "timeout=10.0")

	callArgs := strings.Join(args, ", ")
	callArgs += bodyKwarg

	method := strings.ToLower(in.Method)
	out.WriteString(fmt.Sprintf("with httpx.Client() as client:\n"))
	out.WriteString(fmt.Sprintf("    response = client.%s(%s)\n", method, callArgs))
	out.WriteString("    print(response.status_code)\n")
	out.WriteString("    print(response.text)\n")

	if envs == nil {
		envs = []string{}
	}
	sort.Strings(envs)

	return generator.Output{
		Target:               generator.TargetPython,
		Language:             "python",
		Code:                 out.String(),
		EnvironmentVariables: envs,
	}, nil
}

func pythonDictCode(v any, indent int) (string, []string) {
	var envs []string
	prefix := strings.Repeat("    ", indent)

	switch x := v.(type) {
	case map[string]any:
		if len(x) == 0 {
			return "{}", envs
		}
		var keys []string
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		var b bytes.Buffer
		b.WriteString("{\n")
		for _, k := range keys {
			val := x[k]
			if val == replay.Redacted {
				env := generator.EnvName(k)
				envs = generator.AppendUnique(envs, env)
				b.WriteString(fmt.Sprintf("%s    %q: os.environ[%q],\n", prefix, k, env))
			} else {
				subCode, subEnvs := pythonDictCode(val, indent+1)
				for _, e := range subEnvs {
					envs = generator.AppendUnique(envs, e)
				}
				b.WriteString(fmt.Sprintf("%s    %q: %s,\n", prefix, k, subCode))
			}
		}
		b.WriteString(prefix + "}")
		return b.String(), envs
	case []any:
		if len(x) == 0 {
			return "[]", envs
		}
		var b bytes.Buffer
		b.WriteString("[\n")
		for _, val := range x {
			subCode, subEnvs := pythonDictCode(val, indent+1)
			for _, e := range subEnvs {
				envs = generator.AppendUnique(envs, e)
			}
			b.WriteString(fmt.Sprintf("%s    %s,\n", prefix, subCode))
		}
		b.WriteString(prefix + "]")
		return b.String(), envs
	case string:
		return fmt.Sprintf("%q", x), envs
	default:
		// numbers, booleans, null
		if x == nil {
			return "None", envs
		}
		b, _ := json.Marshal(x)
		if string(b) == "true" {
			return "True", envs
		}
		if string(b) == "false" {
			return "False", envs
		}
		return string(b), envs
	}
}

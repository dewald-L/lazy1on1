package store

import "strings"

// splitFrontMatter splits a file's raw content into its "---" delimited
// front matter block (key: value lines, one per line, no nesting) and the
// remaining body. It intentionally supports only the tiny subset of YAML
// this app actually needs, so the app has no external YAML dependency.
func splitFrontMatter(raw string) (fm map[string]string, body string) {
	fm = map[string]string{}
	lines := strings.Split(raw, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return fm, raw
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return fm, raw
	}

	for _, l := range lines[1:end] {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) == "" {
			continue
		}
		idx := strings.Index(l, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(l[:idx])
		val := strings.TrimSpace(l[idx+1:])
		fm[key] = val
	}

	rest := lines[end+1:]
	// Drop a single leading blank line right after the closing fence, so
	// round-tripping doesn't accumulate blank lines.
	if len(rest) > 0 && strings.TrimSpace(rest[0]) == "" {
		rest = rest[1:]
	}
	return fm, strings.Join(rest, "\n")
}

// splitList parses a simple "a, b, c" value into a trimmed, non-empty slice.
func splitList(v string) []string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "[")
	v = strings.TrimSuffix(v, "]")
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinList(items []string) string {
	return strings.Join(items, ", ")
}

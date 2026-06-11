package hrns

import (
	"regexp"
	"sort"
	"strings"
)

func filter(values []string, keep func(string) bool) []string {
	out := []string{}
	for _, value := range values {
		if keep(value) {
			out = append(out, value)
		}
	}
	return out
}

func indexOf(values []string, needle string) int {
	for i, value := range values {
		if value == needle {
			return i
		}
	}
	return -1
}

func keys[M ~map[string]V, V any](m M) []string {
	out := make([]string, 0, len(m))
	for key := range m {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func nonCommentLines(text string) []string {
	out := []string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "//") {
			out = append(out, line)
		}
	}
	return out
}

func words(text string) []string {
	return regexp.MustCompile(`[a-z0-9가-힣_:-]+`).FindAllString(strings.ToLower(text), -1)
}

func longestWordRun(left, right []string) int {
	best := 0
	prev := make([]int, len(right)+1)
	curr := make([]int, len(right)+1)
	for i := 1; i <= len(left); i++ {
		for j := 1; j <= len(right); j++ {
			if left[i-1] == right[j-1] {
				curr[j] = prev[j-1] + 1
				if curr[j] > best {
					best = curr[j]
				}
			} else {
				curr[j] = 0
			}
		}
		copy(prev, curr)
		for j := range curr {
			curr[j] = 0
		}
	}
	return best
}

func proposalList(raw map[string]any) []map[string]any {
	arr, ok := raw["proposals"].([]any)
	if !ok {
		return []map[string]any{raw}
	}
	out := []map[string]any{}
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}

func firstHeading(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func headings(text string, limit int) []string {
	out := []string{}
	re := regexp.MustCompile(`^#{1,4}\s+(.+)$`)
	for _, line := range strings.Split(text, "\n") {
		if m := re.FindStringSubmatch(line); m != nil {
			out = append(out, strings.TrimSpace(m[1]))
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}

func hasPrefixAny(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func lines(text string) []string {
	out := []string{}
	for _, line := range strings.Split(strings.TrimSpace(text), "\n") {
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return out
}

func normalizeSpace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

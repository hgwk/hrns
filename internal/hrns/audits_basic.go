package hrns

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func RunLineAudit(cfg Config) error {
	max := cfg.LineAudit.MaxLines
	if max == 0 {
		max = 300
	}
	if env := os.Getenv("LINE_AUDIT_MAX"); env != "" {
		fmt.Sscanf(env, "%d", &max)
	}
	exts := stringSet(cfg.LineAudit.Extensions)
	files := ListFiles(cfg.LineAudit.Roots, exts)
	if len(files) == 0 {
		return fmt.Errorf("line-audit: FAIL (0 files scanned; check lineAudit.roots and extensions)")
	}
	type item struct {
		file  string
		lines int
	}
	var over []item
	for _, file := range files {
		lines := countLines(ReadText(file))
		if lines > max {
			over = append(over, item{file, lines})
		}
	}
	if len(over) > 0 {
		sort.Slice(over, func(i, j int) bool { return over[i].lines > over[j].lines })
		for _, item := range over {
			fmt.Fprintf(os.Stderr, "- %s: %d lines\n", item.file, item.lines)
		}
		return fmt.Errorf("line-audit: FAIL (max %d lines)", max)
	}
	fmt.Printf("line-audit: PASS (max %d lines, %d files scanned)\n", max, len(files))
	return nil
}

func RunDocsSymbolSync(cfg Config) error {
	a := NewAudit("verify-docs-symbol-sync")
	linkRe := regexp.MustCompile(`\[[^\]]+\]\(([^)#]+)(?:#[^)]+)?\)`)
	for _, file := range ListFiles(cfg.Docs.Roots, nil) {
		if !strings.HasSuffix(file, ".md") {
			continue
		}
		for _, match := range linkRe.FindAllStringSubmatch(stripFencedCode(ReadText(file)), -1) {
			ref := strings.Trim(match[1], "<>")
			if strings.HasPrefix(ref, "http:") || strings.HasPrefix(ref, "https:") ||
				strings.HasPrefix(ref, "mailto:") || strings.HasPrefix(ref, "#") {
				continue
			}
			if strings.HasPrefix(ref, "/") || strings.Contains(ref, "*") {
				continue
			}
			base := filepath.Dir(file)
			if !Exists(filepath.Clean(filepath.Join(base, ref))) {
				a.Fail(file+": missing referenced artifact", ref)
			}
		}
	}
	return a.Finish()
}

func stripFencedCode(text string) string {
	var out []string
	inFence := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func RunSensitiveConfig(cfg Config) error {
	a := NewAudit("verify-sensitive-config-placeholders")
	patterns := map[string]*regexp.Regexp{
		"openai_key":     regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`),
		"anthropic_key":  regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`),
		"aws_access_key": regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		"agent_bearer":   regexp.MustCompile(`agz_live_[A-Za-z0-9_-]{30,}`),
		"private_key":    regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`),
	}
	for _, file := range ListFiles([]string{".env.example", ".github", "docs", "infra"}, nil) {
		text := ReadText(file)
		for name, re := range patterns {
			if m := re.FindString(text); m != "" {
				a.Fail(file+": "+name, truncate(m, 80))
			}
		}
	}
	return a.Finish()
}

func RunNoOrphanFixtures(cfg Config) error {
	a := NewAudit("verify-no-orphan-fixtures")
	fixtures := filter(ListFiles(cfg.Fixtures.Roots, nil), func(file string) bool {
		return strings.Contains(file, "/fixtures/")
	})
	tests := filter(ListFiles(cfg.Fixtures.TestRoots, nil), func(file string) bool {
		return regexp.MustCompile(`\.(test|spec)\.(ts|tsx|js|mjs)$`).MatchString(file)
	})
	haystack := ""
	for _, file := range tests {
		haystack += "\n" + ReadText(file)
	}
	for _, fixture := range fixtures {
		parts := strings.Split(fixture, "/")
		idx := indexOf(parts, "fixtures")
		root := ""
		if idx >= 0 && idx+1 < len(parts) {
			root = strings.Join(parts[:idx+2], "/")
		}
		dir := filepath.ToSlash(filepath.Dir(fixture))
		leaf := filepath.Base(fixture)
		scenario := ""
		if len(parts) >= 2 {
			scenario = parts[len(parts)-2]
		}
		if !strings.Contains(haystack, root) && !strings.Contains(haystack, dir) &&
			!strings.Contains(haystack, leaf) && !strings.Contains(haystack, scenario) {
			a.Fail(fixture + ": fixture is not referenced by active tests")
		}
	}
	return a.Finish()
}

func RunDuplicateHelpers(cfg Config) error {
	a := NewAudit("verify-duplicate-helpers")
	names := map[string]struct{}{"asNumber": {}, "clamp": {}, "coerceNumber": {}, "parseBoolean": {}, "parseEnv": {}, "sleep": {}, "toNumber": {}}
	re := regexp.MustCompile(`\b(?:export\s+)?(?:function|const)\s+([A-Za-z0-9_]+)\b`)
	loc := map[string]map[string]struct{}{}
	for _, file := range ListFiles([]string{"packages", "scripts"}, nil) {
		if !regexp.MustCompile(`\.(ts|tsx|js|mjs)$`).MatchString(file) || strings.Contains(file, ".test.") {
			continue
		}
		for _, m := range re.FindAllStringSubmatch(ReadText(file), -1) {
			if _, ok := names[m[1]]; !ok {
				continue
			}
			if loc[m[1]] == nil {
				loc[m[1]] = map[string]struct{}{}
			}
			loc[m[1]][file] = struct{}{}
		}
	}
	for name, files := range loc {
		if len(files) > 1 {
			a.Fail(fmt.Sprintf("duplicate helper %q", name), strings.Join(keys(files), ", "))
		}
	}
	return a.Finish()
}

func RunThinBarrelModules(cfg Config) error {
	a := NewAudit("verify-thin-barrel-modules")
	for _, file := range ListFiles([]string{"packages"}, nil) {
		if !strings.HasSuffix(file, "/index.ts") {
			continue
		}
		lines := nonCommentLines(ReadText(file))
		exports := 0
		for _, line := range lines {
			if strings.HasPrefix(line, "export ") {
				exports++
			}
		}
		if len(lines) > 0 && len(lines) == exports && exports <= 1 {
			a.Fail(fmt.Sprintf("%s: thin barrel has %d export line(s)", file, exports))
		}
	}
	return a.Finish()
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	if strings.HasSuffix(text, "\n") {
		return len(strings.Split(text, "\n")) - 1
	}
	return len(strings.Split(text, "\n"))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

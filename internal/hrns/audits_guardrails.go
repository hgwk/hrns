package hrns

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func RunJSONDuplicateKeys(cfg Config) error {
	a := NewAudit("verify-json-duplicate-keys")
	var findings []Finding
	for _, file := range ListFiles(cfg.JSONDuplicateKeys.Roots, stringSet([]string{".json"})) {
		dups, err := duplicateJSONKeys(ReadText(file))
		if err != nil {
			findings = append(findings, Finding{Message: file + ": invalid JSON", Detail: err.Error()})
			continue
		}
		for _, dup := range dups {
			findings = append(findings, Finding{Message: file + ": duplicate JSON key", Detail: dup})
		}
	}
	return finishByMode(a, findings, cfg.JSONDuplicateKeys.Mode)
}

func RunForbiddenReferences(cfg Config) error {
	a := NewAudit("verify-forbidden-references")
	allow := compilePatterns(cfg.ForbiddenRefs.AllowPaths)
	rules := make([]struct {
		re      *regexp.Regexp
		message string
	}, 0, len(cfg.ForbiddenRefs.Rules))
	for _, rule := range cfg.ForbiddenRefs.Rules {
		if rule.Pattern == "" {
			continue
		}
		rules = append(rules, struct {
			re      *regexp.Regexp
			message string
		}{regexp.MustCompile(rule.Pattern), rule.Message})
	}
	var findings []Finding
	if len(rules) == 0 {
		return finishByMode(a, findings, cfg.ForbiddenRefs.Mode)
	}
	for _, file := range ListFiles(cfg.ForbiddenRefs.Roots, nil) {
		if matchesAny(allow, file) {
			continue
		}
		text := ReadText(file)
		for _, rule := range rules {
			if rule.re.MatchString(text) {
				msg := rule.message
				if msg == "" {
					msg = "forbidden reference: " + rule.re.String()
				}
				findings = append(findings, Finding{Message: file + ": " + msg})
			}
		}
	}
	return finishByMode(a, findings, cfg.ForbiddenRefs.Mode)
}

func RunMagicNumbers(cfg Config) error {
	a := NewAudit("verify-magic-numbers")
	allowPaths := compilePatterns(cfg.MagicNumbers.AllowPaths)
	allowed := stringSet(cfg.MagicNumbers.AllowedValues)
	numberRe := regexp.MustCompile(`(?:^|[^\w.])(-?\d+(?:\.\d+)?)(?:$|[^\w.])`)
	var findings []Finding
	for _, file := range ListFiles(cfg.MagicNumbers.Roots, nil) {
		if matchesAny(allowPaths, file) || !isCodeFile(file) || strings.HasSuffix(file, ".tsx") {
			continue
		}
		for i, line := range strings.Split(ReadText(file), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
				continue
			}
			if regexp.MustCompile(`\bconst\s+[A-Z0-9_]+\s*=`).MatchString(line) {
				continue
			}
			for _, match := range numberRe.FindAllStringSubmatch(line, -1) {
				value := match[1]
				if _, ok := allowed[value]; ok {
					continue
				}
				findings = append(findings, Finding{
					Message: fmt.Sprintf("%s:%d inline numeric policy value %s", file, i+1, value),
					Detail:  truncate(trimmed, 140),
				})
			}
		}
	}
	return finishByMode(a, findings, cfg.MagicNumbers.Mode)
}

func RunStructureRatchet(cfg Config) error {
	a := NewAudit("verify-structure-ratchet")
	var findings []Finding
	for _, item := range cfg.StructureRatchet.Files {
		if item.Path == "" || !Exists(item.Path) {
			continue
		}
		text := ReadText(item.Path)
		if item.MaxLines > 0 {
			lines := countLines(text)
			if lines > item.MaxLines {
				findings = append(findings, Finding{
					Message: item.Path + ": line budget exceeded",
					Detail:  fmt.Sprintf("%d > %d", lines, item.MaxLines),
				})
			}
		}
		for _, metric := range item.Metrics {
			if metric.Pattern == "" || metric.Max < 0 {
				continue
			}
			count := len(regexp.MustCompile(metric.Pattern).FindAllString(text, -1))
			if count > metric.Max {
				name := metric.Name
				if name == "" {
					name = metric.Pattern
				}
				findings = append(findings, Finding{
					Message: item.Path + ": ratchet metric exceeded: " + name,
					Detail:  fmt.Sprintf("%d > %d", count, metric.Max),
				})
			}
		}
	}
	return finishByMode(a, findings, cfg.StructureRatchet.Mode)
}

func RunNoPlaceholderRoutes(cfg Config) error {
	a := NewAudit("verify-no-placeholder-routes")
	var findings []Finding
	routeRe := regexp.MustCompile(`(?i)(route\.(ts|js)$|/api/|/routes?/|/pages/api/)`)
	statusRe := regexp.MustCompile(`(?i)(status\s*[:=]\s*501|SERVICE_UNAVAILABLE|not implemented|coming soon|placeholder)`)
	for _, file := range ListFiles(cfg.PlaceholderRoutes.Roots, nil) {
		if !isCodeFile(file) || !routeRe.MatchString(filepath.ToSlash(file)) {
			continue
		}
		if statusRe.MatchString(ReadText(file)) {
			findings = append(findings, Finding{Message: file + ": stable route contains placeholder/unimplemented response"})
		}
	}
	return finishByMode(a, findings, cfg.PlaceholderRoutes.Mode)
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern != "" {
			out = append(out, regexp.MustCompile(pattern))
		}
	}
	return out
}

func matchesAny(patterns []*regexp.Regexp, value string) bool {
	value = filepath.ToSlash(value)
	return slices.ContainsFunc(patterns, func(re *regexp.Regexp) bool { return re.MatchString(value) })
}

func isCodeFile(file string) bool {
	switch filepath.Ext(file) {
	case ".go", ".js", ".mjs", ".cjs", ".ts", ".tsx", ".rs", ".sql":
		return true
	default:
		return false
	}
}

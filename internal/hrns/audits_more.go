package hrns

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func RunEnvExampleSync(cfg Config) error {
	a := NewAudit("verify-env-example-symbol-sync")
	if len(cfg.Env.RequiredPrefixes) == 0 {
		fmt.Println("verify-env-example-symbol-sync: PASS (no required prefixes configured)")
		return nil
	}
	example := ReadText(cfg.Env.Example)
	exampleKeys := map[string]struct{}{}
	for _, m := range regexp.MustCompile(`(?m)^#?\s*([A-Z][A-Z0-9_]+)=`).FindAllStringSubmatch(example, -1) {
		exampleKeys[m[1]] = struct{}{}
	}
	ignored := stringSet(cfg.Env.Ignored)
	envRe := regexp.MustCompile(`process\.env(?:\[['"]([A-Z0-9_]+)['"]\]|\.([A-Z0-9_]+))`)
	seen := map[string]struct{}{}
	for _, file := range ListFiles(cfg.Env.Roots, nil) {
		for _, m := range envRe.FindAllStringSubmatch(ReadText(file), -1) {
			key := m[1]
			if key == "" {
				key = m[2]
			}
			seen[key] = struct{}{}
		}
	}
	for key := range seen {
		if _, skip := ignored[key]; skip || !hasPrefixAny(key, cfg.Env.RequiredPrefixes) {
			continue
		}
		if _, ok := exampleKeys[key]; !ok {
			a.Fail(key + " is read from code but missing in " + cfg.Env.Example)
		}
	}
	return a.Finish()
}

func RunMainDiffScope(cfg Config) error {
	a := NewAudit("verify-main-diff-scope")
	base := cfg.MainDiff.Base
	mergeBase := strings.TrimSpace(git("merge-base", "HEAD", base))
	if mergeBase == "" {
		return finishByMode(a, []Finding{{Message: "cannot find merge base with " + base}}, cfg.MainDiff.Mode)
	}
	riskyPatterns, findings := compilePatterns("riskyPatterns", cfg.MainDiff.RiskyPatterns)
	names := lines(git("diff", "--name-only", mergeBase, "HEAD"))
	stat := lines(git("diff", "--numstat", mergeBase, "HEAD"))
	changed := 0
	for _, line := range stat {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			changed += numeric(parts[0]) + numeric(parts[1])
		}
	}
	if len(names) > cfg.MainDiff.MaxFiles {
		findings = append(findings, Finding{"diff touches too many files", fmt.Sprintf("%d > %d", len(names), cfg.MainDiff.MaxFiles)})
	}
	if changed > cfg.MainDiff.MaxChangedLines {
		findings = append(findings, Finding{"diff changes too many lines", fmt.Sprintf("%d > %d", changed, cfg.MainDiff.MaxChangedLines)})
	}
	for _, file := range names {
		for _, pattern := range riskyPatterns {
			if pattern.MatchString(file) {
				findings = append(findings, Finding{"diff touches risky/generated path", file})
			}
		}
	}
	return finishByMode(a, findings, cfg.MainDiff.Mode)
}

func RunStopRule(cfg Config) error {
	a := NewAudit("verify-stop-rule")
	text := ""
	for _, path := range cfg.StopRule.LogPaths {
		if Exists(path) {
			text += "\n" + ReadText(path)
		}
	}
	counts := map[string]int{}
	for _, m := range regexp.MustCompile(`(?i)(?:FAIL|ERROR|failed|error):?\s+(.{8,120})`).FindAllStringSubmatch(text, -1) {
		counts[normalizeSpace(strings.ToLower(m[1]))]++
	}
	var findings []Finding
	threshold := cfg.StopRule.RepeatedFailureThreshold
	if threshold == 0 {
		threshold = 2
	}
	for failure, count := range counts {
		if count >= threshold {
			findings = append(findings, Finding{"repeated failure pattern should trigger replanning", fmt.Sprintf("%dx %s", count, failure)})
		}
	}
	return finishByMode(a, findings, cfg.StopRule.Mode)
}

func RunEleganceReview(cfg Config) error {
	a := NewAudit("verify-elegance-review")
	base := cfg.Elegance.Base
	mergeBase := strings.TrimSpace(git("merge-base", "HEAD", base))
	if mergeBase == "" {
		return finishByMode(a, []Finding{{Message: "cannot find merge base with " + base}}, cfg.Elegance.Mode)
	}
	status := lines(git("diff", "--name-status", mergeBase, "HEAD"))
	newFiles := 0
	for _, line := range status {
		if strings.HasPrefix(line, "A\t") {
			newFiles++
		}
	}
	var findings []Finding
	if newFiles > cfg.Elegance.MaxNewFiles {
		findings = append(findings, Finding{"large number of new files; consider a smaller change boundary", fmt.Sprintf("%d > %d", newFiles, cfg.Elegance.MaxNewFiles)})
	}
	large := 0
	for _, line := range lines(git("diff", "--numstat", mergeBase, "HEAD")) {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 && numeric(parts[0])+numeric(parts[1]) >= cfg.Elegance.LargeFileLineThreshold {
			large++
		}
	}
	if large > cfg.Elegance.MaxLargeFiles {
		findings = append(findings, Finding{"many large changed files; consider splitting/refactoring", fmt.Sprintf("%d > %d", large, cfg.Elegance.MaxLargeFiles)})
	}
	patch := git("diff", mergeBase, "HEAD")
	for _, pattern := range cfg.Elegance.SmellPatterns {
		count := len(regexp.MustCompile(`(?im)^\+.*`+regexp.QuoteMeta(pattern)).FindAllString(patch, -1))
		if count > 0 {
			findings = append(findings, Finding{"new patch contains unresolved smell marker", fmt.Sprintf("%s: %d", pattern, count)})
		}
	}
	return finishByMode(a, findings, cfg.Elegance.Mode)
}

func git(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func numeric(value string) int {
	n, _ := strconv.Atoi(value)
	return n
}

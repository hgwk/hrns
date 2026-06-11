package hrns

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func RunScopeDrift(cfg Config) error {
	a := NewAudit("verify-scope-drift")
	mergeBase := mergeBaseFor(cfg.ScopeDrift.Base)
	if mergeBase == "" {
		return finishByMode(a, []Finding{{Message: "cannot find merge base with " + cfg.ScopeDrift.Base}}, cfg.ScopeDrift.Mode)
	}
	claimed := ledgerClaimPaths()
	if len(claimed) == 0 {
		fmt.Println("verify-scope-drift: PASS (no active ledger claim paths)")
		return nil
	}
	var findings []Finding
	for _, file := range changedFiles(mergeBase) {
		if isLedgerOrInstructionFile(file) || pathCoveredByClaims(file, claimed) {
			continue
		}
		findings = append(findings, Finding{
			Message: "changed file is outside active ledger claim paths",
			Detail:  file,
		})
	}
	return finishByMode(a, findings, cfg.ScopeDrift.Mode)
}

func RunSpeculativeAbstractions(cfg Config) error {
	a := NewAudit("verify-speculative-abstractions")
	mergeBase := mergeBaseFor(cfg.Abstractions.Base)
	if mergeBase == "" {
		return finishByMode(a, []Finding{{Message: "cannot find merge base with " + cfg.Abstractions.Base}}, cfg.Abstractions.Mode)
	}
	threshold := cfg.Abstractions.SingleUseThreshold
	if threshold == 0 {
		threshold = 1
	}
	terms := cfg.Abstractions.Terms
	if len(terms) == 0 {
		terms = DefaultConfig().Abstractions.Terms
	}
	var findings []Finding
	for _, file := range changedFiles(mergeBase) {
		if !isCodeFile(file) {
			continue
		}
		text := ReadText(file)
		patch := git("diff", mergeBase, "HEAD", "--", file)
		for _, term := range terms {
			if !addedAbstraction(patch, term) {
				continue
			}
			count := len(regexp.MustCompile(`\b`+regexp.QuoteMeta(term)+`\b`).FindAllString(text, -1))
			if count <= threshold {
				findings = append(findings, Finding{
					Message: "new single-use abstraction may be speculative",
					Detail:  fmt.Sprintf("%s: %s appears %d time(s)", file, term, count),
				})
			}
		}
	}
	return finishByMode(a, findings, cfg.Abstractions.Mode)
}

func RunRegressionEvidence(cfg Config) error {
	a := NewAudit("verify-regression-evidence")
	mergeBase := mergeBaseFor(cfg.Regression.Base)
	if mergeBase == "" {
		return finishByMode(a, []Finding{{Message: "cannot find merge base with " + cfg.Regression.Base}}, cfg.Regression.Mode)
	}
	files := changedFiles(mergeBase)
	if hasTestChange(files, cfg.Regression.TestPaths) {
		return a.Finish()
	}
	text := strings.ToLower(strings.Join(files, "\n") + "\n" + git("diff", mergeBase, "HEAD"))
	for _, keyword := range cfg.Regression.BugKeywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return finishByMode(a, []Finding{{
				Message: "bugfix-looking change has no changed regression test",
				Detail:  "keyword: " + keyword,
			}}, cfg.Regression.Mode)
		}
	}
	return a.Finish()
}

func mergeBaseFor(base string) string {
	if base == "" {
		base = "main"
	}
	return strings.TrimSpace(git("merge-base", "HEAD", base))
}

func changedFiles(mergeBase string) []string {
	return lines(git("diff", "--name-only", mergeBase, "HEAD"))
}

func ledgerClaimPaths() []string {
	rows := readJSONL("ledger/tickets.jsonl")
	latest := map[string]map[string]any{}
	for _, row := range rows {
		id := stringValue(row["id"])
		if id == "" {
			id = stringValue(row["ticket"])
		}
		if id == "" {
			continue
		}
		latest[id] = row
	}
	out := []string{}
	for _, row := range latest {
		state := stringValue(row["state"])
		if state == "" {
			state = stringValue(row["status"])
		}
		if !activeClaimState(state) {
			continue
		}
		out = append(out, stringSliceAny(row["paths"])...)
	}
	return appendUnique(nil, out...)
}

func readJSONL(path string) []map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []map[string]any
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row map[string]any
		if json.Unmarshal([]byte(line), &row) == nil {
			out = append(out, row)
		}
	}
	return out
}

func stringSliceAny(raw any) []string {
	arr, _ := raw.([]any)
	out := []string{}
	for _, item := range arr {
		if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, normalizePath(s))
		}
	}
	return out
}

func activeClaimState(state string) bool {
	switch state {
	case "ready", "doing", "blocked", "review", "rework", "open", "planned", "claimed", "in_progress", "audit_ready", "changes_requested", "review_ready":
		return true
	default:
		return false
	}
}

func pathCoveredByClaims(file string, claims []string) bool {
	file = normalizePath(file)
	for _, claim := range claims {
		claim = strings.TrimSuffix(normalizePath(claim), "/")
		if claim == "" || claim == "." {
			return true
		}
		if file == claim || strings.HasPrefix(file, claim+"/") {
			return true
		}
	}
	return false
}

func isLedgerOrInstructionFile(file string) bool {
	switch normalizePath(file) {
	case "ledger/tickets.jsonl", "ledger/worklog.jsonl", "ledger/goal.json", "ledger/config.json", "AGENTS.md", "CLAUDE.md", "hrns.config.json":
		return true
	default:
		return false
	}
}

func addedAbstraction(patch, term string) bool {
	re := regexp.MustCompile(`(?m)^\+.*\b(?:type|interface|class|struct|func|function|const|let|var)\b.*\b` + regexp.QuoteMeta(term) + `\b`)
	return re.MatchString(patch)
}

func hasTestChange(files []string, patterns []string) bool {
	for _, file := range files {
		lower := strings.ToLower(filepath.ToSlash(file))
		for _, pattern := range patterns {
			if strings.Contains(lower, strings.ToLower(pattern)) {
				return true
			}
		}
	}
	return false
}

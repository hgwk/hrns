package hrns

import (
	"fmt"
	"os"
	"path/filepath"
)

type auditInfo struct {
	Name        string
	Category    string
	Purpose     string
	Config      string
	Failure     string
	NeedsConfig func(Config) bool
	IsOff       func(Config) bool
}

var auditCatalog = map[string]auditInfo{
	"verify-line-count": {
		Name: "verify-line-count", Category: "stable",
		Purpose: "checks configured source roots for files over lineAudit.maxLines",
		Config:  "lineAudit.roots, lineAudit.extensions, lineAudit.maxLines",
		Failure: "a source file exceeds the configured line budget",
	},
	"verify-docs-symbol-sync": {
		Name: "verify-docs-symbol-sync", Category: "stable",
		Purpose: "checks Markdown relative links and referenced local docs paths",
		Config:  "docs.roots",
		Failure: "a Markdown link points at a missing local file",
	},
	"verify-sensitive-config-placeholders": {
		Name: "verify-sensitive-config-placeholders", Category: "stable",
		Purpose: "checks example/docs/infra files for secret-looking literal values",
		Config:  "env.example, env.roots",
		Failure: "a placeholder file appears to contain a real secret",
	},
	"verify-no-orphan-fixtures": {
		Name: "verify-no-orphan-fixtures", Category: "stable",
		Purpose: "checks fixture files under configured roots are referenced by tests",
		Config:  "fixtures.roots, fixtures.testRoots",
		Failure: "a fixture appears unused",
	},
	"verify-duplicate-helpers": {
		Name: "verify-duplicate-helpers", Category: "stable",
		Purpose: "checks for repeated small helper/function names across source files",
		Config:  "lineAudit.roots",
		Failure: "similar helper names are repeatedly defined",
	},
	"verify-thin-barrel-modules": {
		Name: "verify-thin-barrel-modules", Category: "stable",
		Purpose: "checks index/barrel modules that are too thin to justify ownership",
		Config:  "lineAudit.roots",
		Failure: "a barrel file only re-exports a very small surface",
	},
	"verify-env-example-symbol-sync": configurable("verify-env-example-symbol-sync", "checks process.env usage against .env.example", "env.example, env.roots, env.requiredPrefixes", "an env variable is used but missing from the example file", func(c Config) bool {
		return c.Env.Example == "" || len(c.Env.Roots) == 0 || !pathExists(c.Env.Example) || !anyPathExists(c.Env.Roots)
	}, nil),
	"verify-agent-instruction-drift": configurable("verify-agent-instruction-drift", "checks repeated long instruction text across agent policy files", "agentInstructions.files, agentInstructions.mode", "instruction files duplicate long policy text", nil, func(c Config) bool { return c.AgentInstructions.Mode == "off" }),
	"verify-docs-duplication":        configurable("verify-docs-duplication", "checks Markdown documents for near-duplicate content", "docsDuplication.roots, docsDuplication.threshold, docsDuplication.mode", "two docs overlap enough that one may belong in the other", nil, func(c Config) bool { return c.DocsDuplication.Mode == "off" }),
	"verify-doc-proposal": configurable("verify-doc-proposal", "checks a proposed Markdown document before creating it", "docsProposal.proposalPath, docsProposal.roots, docsProposal.mode", "a proposed doc overlaps an existing doc", func(c Config) bool {
		return c.DocsProposal.ProposalPath == "" || !pathExists(c.DocsProposal.ProposalPath) || !anyPathExists(c.DocsProposal.Roots)
	}, func(c Config) bool { return c.DocsProposal.Mode == "off" }),
	"verify-json-duplicate-keys":      configurable("verify-json-duplicate-keys", "checks JSON files for duplicate keys before parsers keep the last value", "jsonDuplicateKeys.roots, jsonDuplicateKeys.mode", "a JSON object repeats a key", nil, func(c Config) bool { return c.JSONDuplicateKeys.Mode == "off" }),
	"verify-forbidden-references":     configurable("verify-forbidden-references", "checks configured legacy names/imports/surfaces are not referenced", "forbiddenReferences.rules, roots, allowPaths, mode", "a forbidden pattern appears outside allowlisted paths", func(c Config) bool { return len(c.ForbiddenRefs.Rules) == 0 }, func(c Config) bool { return c.ForbiddenRefs.Mode == "off" }),
	"verify-magic-numbers":            configurable("verify-magic-numbers", "checks inline numeric policy values that should be named constants", "magicNumbers.roots, allowedValues, mode", "a non-allowlisted numeric literal appears in source", func(c Config) bool { return !anyPathExists(c.MagicNumbers.Roots) }, func(c Config) bool { return c.MagicNumbers.Mode == "off" }),
	"verify-structure-ratchet":        configurable("verify-structure-ratchet", "checks per-file line and regex budgets", "structureRatchet.files, mode", "a configured file exceeds its ratchet", func(c Config) bool { return len(c.StructureRatchet.Files) == 0 }, func(c Config) bool { return c.StructureRatchet.Mode == "off" }),
	"verify-no-placeholder-routes":    configurable("verify-no-placeholder-routes", "checks stable routes do not return placeholder responses", "placeholderRoutes.roots, mode", "a route still returns placeholder/not implemented text", func(c Config) bool { return !anyPathExists(c.PlaceholderRoutes.Roots) }, func(c Config) bool { return c.PlaceholderRoutes.Mode == "off" }),
	"verify-scope-drift":              configurable("verify-scope-drift", "checks changed files against active ldgr claim paths", "scopeDrift.base, mode", "a diff changes files outside active claim scope", needsGitRepo, func(c Config) bool { return c.ScopeDrift.Mode == "off" }),
	"verify-speculative-abstractions": configurable("verify-speculative-abstractions", "checks new single-use abstraction surfaces", "speculativeAbstractions.terms, base, mode", "a likely premature abstraction was added", needsGitRepo, func(c Config) bool { return c.Abstractions.Mode == "off" }),
	"verify-regression-evidence":      configurable("verify-regression-evidence", "checks bugfix-looking diffs include regression test evidence", "regressionEvidence.bugKeywords, testPaths, mode", "a fix-like change lacks a changed test", needsGitRepo, func(c Config) bool { return c.Regression.Mode == "off" }),
	"verify-main-diff-scope":          configurable("verify-main-diff-scope", "checks total changed files/lines and risky path changes", "mainDiff.base, maxFiles, maxChangedLines, riskyPatterns, mode", "the diff is broader or riskier than configured", needsGitRepo, func(c Config) bool { return c.MainDiff.Mode == "off" }),
	"verify-stop-rule":                configurable("verify-stop-rule", "checks repeated failure logs for stop-and-replan conditions", "stopRule.logPaths, repeatedFailureThreshold, mode", "the same failure appears repeatedly", func(c Config) bool { return !anyPathExists(c.StopRule.LogPaths) }, func(c Config) bool { return c.StopRule.Mode == "off" }),
	"verify-elegance-review":          configurable("verify-elegance-review", "checks diff shape for broad churn and patch-smell markers", "elegance.base, maxNewFiles, maxLargeFiles, smellPatterns, mode", "a diff is too broad or contains patch-smell markers", needsGitRepo, func(c Config) bool { return c.Elegance.Mode == "off" }),
}

func configurable(name, purpose, config, failure string, needs func(Config) bool, off func(Config) bool) auditInfo {
	return auditInfo{Name: name, Category: "configurable", Purpose: purpose, Config: config, Failure: failure, NeedsConfig: needs, IsOff: off}
}

func auditStatus(name string, cfg Config) string {
	info, ok := auditCatalog[normalizeAuditName(name)]
	if !ok {
		return "unknown"
	}
	if info.IsOff != nil && info.IsOff(cfg) {
		return "inactive"
	}
	if info.NeedsConfig != nil && info.NeedsConfig(cfg) {
		return "needs config"
	}
	return "active"
}

func explainAudit(name string, cfg Config) error {
	info, ok := auditCatalog[normalizeAuditName(name)]
	if !ok {
		return fmt.Errorf("unknown audit: %s", name)
	}
	fmt.Printf("%s\n", info.Name)
	fmt.Printf("category: %s\n", info.Category)
	fmt.Printf("status: %s\n", auditStatus(info.Name, cfg))
	fmt.Printf("purpose: %s\n", info.Purpose)
	fmt.Printf("config: %s\n", info.Config)
	fmt.Printf("failure: %s\n", info.Failure)
	return nil
}

func anyPathExists(paths []string) bool {
	for _, path := range paths {
		if pathExists(path) {
			return true
		}
	}
	return false
}

func pathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(filepath.Clean(path))
	return err == nil
}

func needsGitRepo(Config) bool {
	return !pathExists(".git")
}

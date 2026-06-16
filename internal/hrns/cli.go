package hrns

import (
	"fmt"
	"os"
)

var stableAudits = []string{
	"verify-line-count",
	"verify-docs-symbol-sync",
	"verify-sensitive-config-placeholders",
	"verify-no-orphan-fixtures",
	"verify-duplicate-helpers",
	"verify-thin-barrel-modules",
}

var allAudits = []string{
	"verify-line-count",
	"verify-docs-symbol-sync",
	"verify-sensitive-config-placeholders",
	"verify-no-orphan-fixtures",
	"verify-duplicate-helpers",
	"verify-thin-barrel-modules",
	"verify-env-example-symbol-sync",
	"verify-agent-instruction-drift",
	"verify-docs-duplication",
	"verify-doc-proposal",
	"verify-json-duplicate-keys",
	"verify-forbidden-references",
	"verify-magic-numbers",
	"verify-structure-ratchet",
	"verify-no-placeholder-routes",
	"verify-scope-drift",
	"verify-speculative-abstractions",
	"verify-regression-evidence",
	"verify-main-diff-scope",
	"verify-stop-rule",
	"verify-elegance-review",
}

var Version = "0.1.6"

func Run(args []string) error {
	cmd := "audit"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	if isHelp(cmd) {
		printHelp()
		return nil
	}
	if cmd == "version" || cmd == "--version" || cmd == "-V" {
		if len(args) > 0 {
			return fmt.Errorf("usage: hrns version")
		}
		fmt.Printf("hrns %s\n", Version)
		return nil
	}
	if len(args) > 0 && isHelp(args[0]) {
		printCommandHelp(cmd)
		return nil
	}
	target, args, err := parseTargetArg(args)
	if err != nil {
		return err
	}
	if cmd == "init" {
		if err := os.MkdirAll(target, 0o755); err != nil {
			return fmt.Errorf("target %s: %w", target, err)
		}
	}
	restore, err := chdirTarget(target)
	if err != nil {
		return err
	}
	defer restore()
	cfg, err := LoadConfig(".")
	if err != nil {
		return err
	}
	switch cmd {
	case "list":
		if contains(args, "--json") {
			return printListJSON(cfg)
		}
		printList(cfg, contains(args, "--verbose") || contains(args, "-v"))
	case "init":
		return initCommand(args)
	case "audit":
		includeAll := contains(args, "--all")
		withLdgr := contains(args, "--with-ldgr")
		audits := cfg.AuditSets.Default
		if len(audits) == 0 {
			audits = stableAudits
		}
		if includeAll {
			audits = cfg.AuditSets.All
			if len(audits) == 0 {
				audits = allAudits
			}
		}
		if err := runMany(audits, cfg); err != nil {
			return err
		}
		if withLdgr {
			return runLdgrVerify()
		}
		return nil
	case "run":
		if len(args) == 0 {
			return fmt.Errorf("usage: hrns run <audit-name>")
		}
		return runOne(normalizeAuditName(args[0]), cfg)
	case "explain":
		if len(args) == 0 {
			return fmt.Errorf("usage: hrns explain <audit-name>")
		}
		return explainAudit(args[0], cfg)
	case "line-audit":
		return RunLineAudit(cfg)
	case "docs:index":
		return WriteDocsIndex(cfg)
	case "docs:check":
		if len(args) > 0 {
			_ = os.Setenv("HRNS_DOC_PROPOSAL", args[0])
		}
		return RunDocProposal(cfg)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
	return nil
}

func isHelp(value string) bool {
	return value == "help" || value == "--help" || value == "-h"
}

func normalizeAuditName(name string) string {
	for len(name) > 4 && name[len(name)-4:] == ".mjs" {
		name = name[:len(name)-4]
	}
	return name
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

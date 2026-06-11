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

var Version = "0.1.1"

func Run(args []string) error {
	cmd := "audit"
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}
	if cmd == "version" || cmd == "--version" || cmd == "-V" {
		if len(args) > 0 {
			return fmt.Errorf("usage: hrns version")
		}
		fmt.Printf("hrns %s\n", Version)
		return nil
	}
	cfg, err := LoadConfig(".")
	if err != nil {
		return err
	}
	switch cmd {
	case "list":
		printList(cfg)
	case "init":
		return initCommand(args)
	case "audit":
		includeAll := contains(args, "--all")
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
		return runMany(audits, cfg)
	case "run":
		if len(args) == 0 {
			return fmt.Errorf("usage: hrns run <audit-name>")
		}
		return runOne(normalizeAuditName(args[0]), cfg)
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

func printList(cfg Config) {
	fmt.Println("Stable audits:")
	for _, name := range stableAudits {
		fmt.Printf("- %s\n", name)
	}
	fmt.Println("\nConfigurable audits:")
	for _, name := range allAudits[len(stableAudits):] {
		fmt.Printf("- %s\n", name)
	}
	fmt.Println("\nConfigured default audit set:")
	audits := cfg.AuditSets.Default
	if len(audits) == 0 {
		audits = stableAudits
	}
	for _, name := range audits {
		fmt.Printf("- %s\n", normalizeAuditName(name))
	}
}

func runMany(audits []string, cfg Config) error {
	failures := 0
	for _, name := range audits {
		if err := runOne(normalizeAuditName(name), cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			failures++
		}
	}
	if failures > 0 {
		return fmt.Errorf("audit: FAIL (%d/%d audit(s) failed)", failures, len(audits))
	}
	fmt.Printf("hrns audit: PASS (%d audit(s) passed)\n", len(audits))
	return nil
}

func runOne(name string, cfg Config) error {
	switch normalizeAuditName(name) {
	case "verify-line-count":
		return RunLineAudit(cfg)
	case "verify-docs-symbol-sync":
		return RunDocsSymbolSync(cfg)
	case "verify-sensitive-config-placeholders":
		return RunSensitiveConfig(cfg)
	case "verify-no-orphan-fixtures":
		return RunNoOrphanFixtures(cfg)
	case "verify-duplicate-helpers":
		return RunDuplicateHelpers(cfg)
	case "verify-thin-barrel-modules":
		return RunThinBarrelModules(cfg)
	case "verify-env-example-symbol-sync":
		return RunEnvExampleSync(cfg)
	case "verify-agent-instruction-drift":
		return RunAgentInstructionDrift(cfg)
	case "verify-docs-duplication":
		return RunDocsDuplication(cfg)
	case "verify-doc-proposal":
		return RunDocProposal(cfg)
	case "verify-json-duplicate-keys":
		return RunJSONDuplicateKeys(cfg)
	case "verify-forbidden-references":
		return RunForbiddenReferences(cfg)
	case "verify-magic-numbers":
		return RunMagicNumbers(cfg)
	case "verify-structure-ratchet":
		return RunStructureRatchet(cfg)
	case "verify-no-placeholder-routes":
		return RunNoPlaceholderRoutes(cfg)
	case "verify-scope-drift":
		return RunScopeDrift(cfg)
	case "verify-speculative-abstractions":
		return RunSpeculativeAbstractions(cfg)
	case "verify-regression-evidence":
		return RunRegressionEvidence(cfg)
	case "verify-main-diff-scope":
		return RunMainDiffScope(cfg)
	case "verify-stop-rule":
		return RunStopRule(cfg)
	case "verify-elegance-review":
		return RunEleganceReview(cfg)
	default:
		return fmt.Errorf("unknown audit: %s", name)
	}
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

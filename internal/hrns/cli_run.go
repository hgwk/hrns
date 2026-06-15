package hrns

import (
	"fmt"
	"os"
)

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

package hrns

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func initCommand(args []string) error {
	profile, err := parseInitProfile(args)
	if err != nil {
		return err
	}
	if err := initConfig(profile); err != nil {
		return err
	}
	if contains(args, "--docs") {
		if err := initDocsProposal(); err != nil {
			return err
		}
	}
	if contains(args, "--instructions") {
		if err := initInstructions(); err != nil {
			return err
		}
	}
	return nil
}

func initConfig(profile string) error {
	target := "hrns.config.json"
	if Exists(target) {
		fmt.Println("hrns.config.json already exists")
		return nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg := ConfigForProject(wd)
	applyInitProfile(&cfg, profile)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(target, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println("created hrns.config.json")
	return nil
}

func parseInitProfile(args []string) (string, error) {
	for i, arg := range args {
		if arg == "--profile" {
			if i+1 >= len(args) {
				return "", fmt.Errorf("usage: hrns init [--docs] [--instructions] [--profile node|go|rust|next]")
			}
			return normalizeInitProfile(args[i+1])
		}
		if strings.HasPrefix(arg, "--profile=") {
			return normalizeInitProfile(strings.TrimPrefix(arg, "--profile="))
		}
	}
	return "", nil
}

func normalizeInitProfile(profile string) (string, error) {
	switch profile {
	case "", "node", "go", "rust", "next":
		return profile, nil
	default:
		return "", fmt.Errorf("unknown init profile %q", profile)
	}
}

func applyInitProfile(cfg *Config, profile string) {
	switch profile {
	case "node":
		cfg.LineAudit.Roots = []string{"src", "apps", "packages", "scripts"}
		cfg.LineAudit.Extensions = []string{".ts", ".tsx", ".mjs", ".js", ".cjs"}
		cfg.Env.Roots = []string{"src", "apps", "packages", "scripts"}
		cfg.Fixtures.Roots = []string{"src", "apps", "packages", "fixtures"}
		cfg.Fixtures.TestRoots = []string{"src", "apps", "packages", "tests", "__tests__"}
	case "next":
		cfg.LineAudit.Roots = []string{"app", "pages", "components", "src", "packages", "scripts"}
		cfg.LineAudit.Extensions = []string{".ts", ".tsx", ".mjs", ".js", ".cjs"}
		cfg.Env.Roots = []string{"app", "pages", "components", "src", "packages", "scripts"}
		cfg.Fixtures.Roots = []string{"src", "packages", "fixtures"}
		cfg.Fixtures.TestRoots = []string{"tests", "__tests__", "src", "packages"}
	case "go":
		cfg.LineAudit.Roots = []string{"cmd", "internal", "pkg", "scripts"}
		cfg.LineAudit.Extensions = []string{".go", ".sh"}
		cfg.Env.Roots = []string{"cmd", "internal", "pkg", "scripts"}
		cfg.Fixtures.Roots = []string{"testdata", "fixtures"}
		cfg.Fixtures.TestRoots = []string{"cmd", "internal", "pkg", "tests"}
	case "rust":
		cfg.LineAudit.Roots = []string{"src", "tests", "benches", "crates"}
		cfg.LineAudit.Extensions = []string{".rs"}
		cfg.Env.Roots = []string{"src", "tests", "benches", "crates"}
		cfg.Fixtures.Roots = []string{"tests", "fixtures"}
		cfg.Fixtures.TestRoots = []string{"tests", "src", "crates"}
	}
}

func initDocsProposal() error {
	path := filepath.Join(".hrns", "doc-proposal.json")
	if Exists(path) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	payload := map[string]any{
		"version": 1,
		"proposals": []map[string]any{{
			"path": "", "title": "", "purpose": "Explain why this must be a new document instead of an update.",
			"summary": "Short summary of the planned content.", "decision": "new_document", "target": "",
		}},
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Println("created .hrns/doc-proposal.json")
	return nil
}

func initInstructions() error {
	bodyPath := instructionBodyPath()
	if err := os.MkdirAll(filepath.Dir(bodyPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(bodyPath, []byte(defaultInstructions()), 0o644); err != nil {
		return err
	}
	fmt.Printf("updated %s\n", bodyPath)
	for _, file := range []string{"CLAUDE.md", "AGENTS.md"} {
		if err := injectInstructionInclude(file); err != nil {
			return err
		}
	}
	return nil
}

func injectInstructionInclude(file string) error {
	pointer := "@" + instructionBodyPath()
	if !Exists(file) {
		if err := os.WriteFile(file, []byte(pointer+"\n"), 0o644); err != nil {
			return err
		}
		fmt.Printf("created %s\n", file)
		return nil
	}
	current := ReadText(file)
	updated := upsertInstructionPointer(current, pointer)
	if updated == current {
		fmt.Printf("%s already references hrns instructions\n", file)
		return nil
	}
	if err := os.WriteFile(file, []byte(updated), 0o644); err != nil {
		return err
	}
	fmt.Printf("updated %s\n", file)
	return nil
}

func upsertInstructionPointer(current, pointer string) string {
	cleaned := strings.TrimSpace(stripLeadingSeparator(removeKnownInstructionPointers(current)))
	if strings.TrimSpace(current) == pointer {
		return current
	}
	if cleaned == "" {
		return pointer + "\n"
	}
	return pointer + "\n\n---\n\n" + cleaned + "\n"
}

func removeKnownInstructionPointers(content string) string {
	out := content
	for _, pointer := range []string{"@" + instructionBodyPath(), "@.hrns/instructions.md", "@.hrns/audit-guide.md"} {
		out = removePointerPrelude(out, pointer)
		out = stripLeadingSeparator(out)
	}
	return out
}

func removePointerPrelude(content, pointer string) string {
	lines := strings.Split(content, "\n")
	pos := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == pointer {
			pos = i
			break
		}
		if strings.TrimSpace(line) != "" {
			return content
		}
	}
	if pos < 0 {
		return content
	}
	return strings.Join(append(lines[:pos], lines[pos+1:]...), "\n")
}

func stripLeadingSeparator(content string) string {
	return strings.TrimLeft(strings.TrimPrefix(strings.TrimLeft(content, " \t\r\n"), "---"), " \t\r\n")
}

func instructionBodyPath() string {
	home := os.Getenv("HRNS_HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = filepath.Join(h, ".hrns")
		} else {
			home = ".hrns"
		}
	}
	return filepath.Join(home, "audit-guide.md")
}

func defaultInstructions() string {
	return `# hrns Instructions

## Audit Gates

- Run ` + "`hrns audit`" + ` before marking work complete.
- For broader review, run ` + "`hrns audit --all`" + ` and resolve fail-mode findings.
- Keep project-specific gate behavior in ` + "`hrns.config.json`" + ` or ` + "`package.json#hrns`" + `.
- Treat ` + "`verify-scope-drift`" + `, ` + "`verify-speculative-abstractions`" + `, and
  ` + "`verify-regression-evidence`" + ` warnings as prompts to tighten scope, remove
  premature abstractions, or add bug regression evidence.

## Task And Worklog Ownership

- hrns does not own tasks, lessons, tickets, or worklogs.
- Use ldgr for task state, lessons, append-only tickets, worklogs, and handoff records.
- Use hrns only for repository audit gates and document creation checks.

## Document Creation Gate

- Before creating a new Markdown document, write ` + "`.hrns/doc-proposal.json`" + `.
- Run ` + "`hrns docs:index`" + ` and ` + "`hrns docs:check .hrns/doc-proposal.json`" + `.
- If the proposal overlaps an existing document, do not create the new file. Update the reported existing document instead.
- To intentionally update an existing document, set ` + "`\"decision\": \"update_existing\"`" + ` and ` + "`\"target\"`" + ` to the existing document path.

## Duplicate Instruction Control

- Keep long operating instructions in one included file.
- Do not paste the same guidance into both ` + "`AGENTS.md`" + `, ` + "`CLAUDE.md`" + `, and tool-specific instruction files.
- If an include already points to this file, update this file rather than adding another prose copy.
`
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

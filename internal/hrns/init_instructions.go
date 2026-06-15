package hrns

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

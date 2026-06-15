package hrns

import "fmt"

func printHelp() {
	fmt.Println(`hrns audits repository guardrails.

Usage:
  hrns [audit]
  hrns audit [--all] [--with-ldgr] [--target PATH]
  hrns run <audit-name> [--target PATH]
  hrns explain <audit-name> [--target PATH]
  hrns init [--docs] [--instructions] [--profile node|go|rust|next] [--target PATH]
  hrns list [--verbose] [--json] [--target PATH]
  hrns version

Commands:
  audit        Run the configured default audit set
  run          Run one audit by name
  explain      Explain one audit's purpose, config, status, and failure shape
  list         List stable and configurable audits
  init         Create hrns config/docs/instruction files
  docs:index   Build the document similarity index
  docs:check   Check a proposed Markdown document
  version      Print the installed version`)
}

func printCommandHelp(cmd string) {
	switch cmd {
	case "audit":
		fmt.Println("usage: hrns audit [--all] [--with-ldgr] [--target PATH]")
	case "run":
		fmt.Println("usage: hrns run <audit-name> [--target PATH]")
	case "explain":
		fmt.Println("usage: hrns explain <audit-name> [--target PATH]")
	case "init":
		fmt.Println("usage: hrns init [--docs] [--instructions] [--profile node|go|rust|next] [--target PATH]")
	case "docs:check":
		fmt.Println("usage: hrns docs:check [proposal-json] [--target PATH]")
	case "docs:index":
		fmt.Println("usage: hrns docs:index [--target PATH]")
	case "line-audit":
		fmt.Println("usage: hrns line-audit [--target PATH]")
	case "list":
		fmt.Println("usage: hrns list [--verbose] [--json] [--target PATH]")
	case "version":
		fmt.Println("usage: hrns version")
	default:
		printHelp()
	}
}

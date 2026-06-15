package hrns

import (
	"fmt"
	"strings"
)

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

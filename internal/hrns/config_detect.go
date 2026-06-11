package hrns

import (
	"os"
	"path/filepath"
	"sort"
)

func ConfigForProject(root string) Config {
	cfg := DefaultConfig()

	codeRoots := existing(root, "src", "apps", "packages", "cmd", "internal", "scripts", "infra")
	testRoots := existing(root, "tests", "test", "e2e")
	lineRoots := appendUnique(codeRoots, testRoots...)
	if len(lineRoots) > 0 {
		cfg.LineAudit.Roots = lineRoots
		cfg.LineAudit.Extensions = projectExtensions(root, lineRoots)
	}

	docsRoots := existing(root, "docs", "README.md", "AGENTS.md", "CLAUDE.md")
	if len(docsRoots) > 0 {
		cfg.Docs.Roots = docsRoots
	}
	if dupRoots := existing(root, "docs", "README.md"); len(dupRoots) > 0 {
		cfg.DocsDuplication.Roots = dupRoots
		cfg.DocsProposal.Roots = dupRoots
	}

	if len(testRoots) > 0 {
		cfg.Fixtures.Roots = testRoots
		cfg.Fixtures.TestRoots = testRoots
	} else if len(codeRoots) > 0 {
		cfg.Fixtures.Roots = codeRoots
		cfg.Fixtures.TestRoots = codeRoots
	}

	envRoots := appendUnique(codeRoots, testRoots...)
	if len(envRoots) > 0 {
		cfg.Env.Roots = envRoots
	}

	jsonRoots := existing(root, "package.json", "tsconfig.json", "tsconfig.build.json", ".github", "messages")
	if len(jsonRoots) > 0 {
		cfg.JSONDuplicateKeys.Roots = jsonRoots
	}

	forbiddenRoots := appendUnique(codeRoots, testRoots...)
	forbiddenRoots = appendUnique(forbiddenRoots, existing(root, "docs")...)
	if len(forbiddenRoots) > 0 {
		cfg.ForbiddenRefs.Roots = forbiddenRoots
	}

	magicRoots := appendUnique(codeRoots, testRoots...)
	if len(magicRoots) > 0 {
		cfg.MagicNumbers.Roots = magicRoots
	}
	if len(testRoots) > 0 {
		cfg.MagicNumbers.AllowPaths = allowPathPrefixes(testRoots)
	}

	placeholderRoots := existing(root, "apps", "src", "pages", "app")
	if len(placeholderRoots) > 0 {
		cfg.PlaceholderRoutes.Roots = placeholderRoots
	}

	return cfg
}

func existing(root string, paths ...string) []string {
	out := []string{}
	for _, path := range paths {
		if existsAt(root, path) {
			out = append(out, path)
		}
	}
	return out
}

func existsAt(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, rel))
	return err == nil
}

func appendUnique(base []string, values ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(base)+len(values))
	for _, value := range append(base, values...) {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func allowPathPrefixes(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, "^"+normalizePath(path)+"/")
	}
	return out
}

func projectExtensions(root string, roots []string) []string {
	allowed := stringSet(DefaultConfig().LineAudit.Extensions)
	seen := map[string]struct{}{}
	for _, relRoot := range roots {
		walkRoot := filepath.Join(root, relRoot)
		_ = filepath.WalkDir(walkRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr == nil && shouldExclude(rel) {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			ext := filepath.Ext(path)
			if _, ok := allowed[ext]; ok {
				seen[ext] = struct{}{}
			}
			return nil
		})
	}
	if len(seen) == 0 {
		return DefaultConfig().LineAudit.Extensions
	}
	out := make([]string, 0, len(seen))
	for ext := range seen {
		out = append(out, ext)
	}
	sort.Strings(out)
	return out
}

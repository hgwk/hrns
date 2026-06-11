package hrns

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigForProjectDetectsTypescriptModuleShape(t *testing.T) {
	root := t.TempDir()
	mkdirs(t, root, "src", "tests", "migrations", "docs")
	writeFile(t, root, "src/index.ts", "export const value = 1\n")
	writeFile(t, root, "tests/index.test.ts", "import '../src/index'\n")
	writeFile(t, root, "migrations/001_init.sql", "select 1;\n")
	writeFile(t, root, "docs/module-design.md", "# Design\n")
	writeFile(t, root, "README.md", "# Module\n")
	writeFile(t, root, "AGENTS.md", "agent guide\n")
	writeFile(t, root, "CLAUDE.md", "claude guide\n")
	writeFile(t, root, "package.json", "{}\n")
	writeFile(t, root, "tsconfig.json", "{}\n")
	writeFile(t, root, "tsconfig.build.json", "{}\n")

	cfg := ConfigForProject(root)

	assertEqual(t, "line roots", cfg.LineAudit.Roots, []string{"src", "tests"})
	assertEqual(t, "line extensions", cfg.LineAudit.Extensions, []string{".ts"})
	assertEqual(t, "docs roots", cfg.Docs.Roots, []string{"docs", "README.md", "AGENTS.md", "CLAUDE.md"})
	assertEqual(t, "fixture roots", cfg.Fixtures.Roots, []string{"tests"})
	assertEqual(t, "env roots", cfg.Env.Roots, []string{"src", "tests"})
	assertEqual(t, "json roots", cfg.JSONDuplicateKeys.Roots, []string{"package.json", "tsconfig.json", "tsconfig.build.json"})
	assertEqual(t, "forbidden roots", cfg.ForbiddenRefs.Roots, []string{"src", "tests", "docs"})
	assertEqual(t, "magic roots", cfg.MagicNumbers.Roots, []string{"src", "tests"})
	assertEqual(t, "magic allow paths", cfg.MagicNumbers.AllowPaths, []string{"^tests/"})
	assertEqual(t, "placeholder roots", cfg.PlaceholderRoutes.Roots, []string{"src"})
}

func mkdirs(t *testing.T, root string, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(t *testing.T, root, rel, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertEqual[T any](t *testing.T, name string, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %#v, want %#v", name, got, want)
	}
}

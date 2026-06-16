package hrns

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsSymbolSyncResolvesLinksRelativeToDocument(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/architecture.md", "[Roadmap](graph-routing-roadmap.md)\n")
	writeFile(t, root, "docs/graph-routing-roadmap.md", "# Roadmap\n")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Docs.Roots = []string{"docs"}
	if err := RunDocsSymbolSync(cfg); err != nil {
		t.Fatalf("relative link should resolve beside source document: %v", err)
	}
}

func TestDocsSymbolSyncIgnoresFencedCodeLinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/plan.md", "```go\nprintln(\"[x](missing.md)\")\n```\n")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Docs.Roots = []string{"docs"}
	if err := RunDocsSymbolSync(cfg); err != nil && strings.Contains(err.Error(), "missing.md") {
		t.Fatalf("fenced code link should not be checked: %v", err)
	}
}

func TestDocsSymbolSyncStillReportsMissingLinks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "[Missing](docs/missing.md)\n")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Docs.Roots = []string{"README.md"}
	err = RunDocsSymbolSync(cfg)
	if err == nil || !strings.Contains(err.Error(), "verify-docs-symbol-sync failed") {
		t.Fatalf("missing link should fail, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "docs", "missing.md")); !os.IsNotExist(statErr) {
		t.Fatalf("test setup expected missing target, stat err=%v", statErr)
	}
}

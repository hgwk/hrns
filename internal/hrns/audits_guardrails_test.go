package hrns

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForbiddenReferencesReportsInvalidRegex(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ForbiddenRefs.Mode = "fail"
	cfg.ForbiddenRefs.Roots = []string{}
	cfg.ForbiddenRefs.AllowPaths = []string{"["}
	cfg.ForbiddenRefs.Rules = []ForbiddenRule{{Pattern: "("}}

	err := RunForbiddenReferences(cfg)
	if err == nil || !strings.Contains(err.Error(), "verify-forbidden-references failed") {
		t.Fatalf("expected invalid regex failure, got %v", err)
	}
}

func TestStructureRatchetReportsInvalidRegex(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "source.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	cfg.StructureRatchet.Mode = "fail"
	cfg.StructureRatchet.Files = []RatchetFile{{
		Path:    path,
		Metrics: []RatchetMetric{{Pattern: "["}},
	}}

	err := RunStructureRatchet(cfg)
	if err == nil || !strings.Contains(err.Error(), "verify-structure-ratchet failed") {
		t.Fatalf("expected invalid regex failure, got %v", err)
	}
}

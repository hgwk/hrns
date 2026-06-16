package hrns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunInitTarget(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new-project")
	home := t.TempDir()
	t.Setenv("HRNS_HOME", filepath.Join(home, ".hrns"))
	if err := Run([]string{"init", "--target", dir, "--profile", "go"}); err != nil {
		t.Fatalf("Run(init --target): %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "hrns.config.json"))
	if err != nil {
		t.Fatalf("expected target config: %v", err)
	}
	if !strings.Contains(string(data), `"cmd"`) {
		t.Fatalf("expected go profile config, got %s", data)
	}
	for _, file := range []string{"AGENTS.md", "CLAUDE.md"} {
		pointer, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatalf("expected %s: %v", file, err)
		}
		if !strings.Contains(string(pointer), "@"+filepath.Join(home, ".hrns", "audit-guide.md")) {
			t.Fatalf("%s missing home-local pointer: %s", file, pointer)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".hrns", "audit-guide.md")); err != nil {
		t.Fatalf("expected home-local audit guide: %v", err)
	}
}

func TestRunInitHomeArgOverridesEnvHome(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new-project")
	envHome := filepath.Join(t.TempDir(), ".hrns-env")
	flagHome := filepath.Join(t.TempDir(), ".hrns-flag")
	t.Setenv("HRNS_HOME", envHome)
	if err := Run([]string{"init", "--target", dir, "--home", flagHome}); err != nil {
		t.Fatalf("Run(init --home): %v", err)
	}
	if _, err := os.Stat(filepath.Join(flagHome, "audit-guide.md")); err != nil {
		t.Fatalf("expected flag home guide: %v", err)
	}
	if _, err := os.Stat(filepath.Join(envHome, "audit-guide.md")); !os.IsNotExist(err) {
		t.Fatalf("env home should not be used when --home is set: %v", err)
	}
	pointer, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("expected AGENTS.md: %v", err)
	}
	if !strings.HasPrefix(string(pointer), "@"+filepath.Join(flagHome, "audit-guide.md")+"\n") {
		t.Fatalf("AGENTS.md missing flag home pointer: %s", pointer)
	}
}

func TestRunInitCanSkipInstructions(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HRNS_HOME", filepath.Join(home, ".hrns"))
	if err := Run([]string{"init", "--target", dir, "--no-instructions"}); err != nil {
		t.Fatalf("Run(init --no-instructions): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("AGENTS.md should not be created with --no-instructions, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".hrns", "audit-guide.md")); !os.IsNotExist(err) {
		t.Fatalf("audit guide should not be created with --no-instructions, stat err=%v", err)
	}
}

func TestRunAuditWithLdgrRunsAdapterAfterAudits(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)
	if err := os.Mkdir("cmd", 0o755); err != nil {
		t.Fatalf("mkdir cmd: %v", err)
	}
	touch(t, "cmd/ok.go")

	called := false
	old := runLdgrVerify
	runLdgrVerify = func() error {
		called = true
		return nil
	}
	t.Cleanup(func() { runLdgrVerify = old })

	if err := Run([]string{"audit", "--with-ldgr"}); err != nil {
		t.Fatalf("Run(audit --with-ldgr): %v", err)
	}
	if !called {
		t.Fatalf("expected ldgr adapter to run")
	}
}

func TestInitProfileGoWritesGoRoots(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)

	if err := Run([]string{"init", "--profile", "go"}); err != nil {
		t.Fatalf("Run(init --profile go): %v", err)
	}
	data, err := os.ReadFile("hrns.config.json")
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !contains(cfg.LineAudit.Extensions, ".go") || contains(cfg.LineAudit.Extensions, ".tsx") {
		t.Fatalf("unexpected go profile extensions: %+v", cfg.LineAudit.Extensions)
	}
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

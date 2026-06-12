package hrns

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunVersionDoesNotRequireConfig(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	stdout := captureStdout(t, func() {
		if err := Run([]string{"version"}); err != nil {
			t.Fatalf("Run(version): %v", err)
		}
	})
	if got, want := stdout, "hrns "+Version+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunVersionRejectsExtraArgs(t *testing.T) {
	err := Run([]string{"version", "extra"})
	if err == nil || !strings.Contains(err.Error(), "usage: hrns version") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestRunHelpHasNoSideEffects(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)

	stdout := captureStdout(t, func() {
		if err := Run([]string{"init", "--help"}); err != nil {
			t.Fatalf("Run(init --help): %v", err)
		}
	})
	if !strings.Contains(stdout, "usage: hrns init") {
		t.Fatalf("help output missing init usage: %s", stdout)
	}
	if _, err := os.Stat("hrns.config.json"); !os.IsNotExist(err) {
		t.Fatalf("init --help should not create config, stat err=%v", err)
	}
}

func TestRunTopLevelHelp(t *testing.T) {
	stdout := captureStdout(t, func() {
		if err := Run([]string{"--help"}); err != nil {
			t.Fatalf("Run(--help): %v", err)
		}
	})
	if !strings.Contains(stdout, "hrns audit [--all]") {
		t.Fatalf("top-level help missing audit usage: %s", stdout)
	}
	if !strings.Contains(stdout, "hrns explain <audit-name>") {
		t.Fatalf("top-level help missing explain usage: %s", stdout)
	}
}

func TestRunExplainAudit(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)

	stdout := captureStdout(t, func() {
		if err := Run([]string{"explain", "verify-line-count"}); err != nil {
			t.Fatalf("Run(explain): %v", err)
		}
	})
	for _, want := range []string{"verify-line-count", "status: active", "lineAudit.maxLines"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("explain output missing %q:\n%s", want, stdout)
		}
	}
}

func TestRunListShowsAuditStatus(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)

	stdout := captureStdout(t, func() {
		if err := Run([]string{"list"}); err != nil {
			t.Fatalf("Run(list): %v", err)
		}
	})
	if !strings.Contains(stdout, "verify-line-count [active]") {
		t.Fatalf("list missing active status:\n%s", stdout)
	}
	if !strings.Contains(stdout, "verify-forbidden-references [needs config]") {
		t.Fatalf("list missing needs config status:\n%s", stdout)
	}
}

func TestRunListJSON(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)

	stdout := captureStdout(t, func() {
		if err := Run([]string{"list", "--json"}); err != nil {
			t.Fatalf("Run(list --json): %v", err)
		}
	})
	var payload struct {
		Stable []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"stable"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout)
	}
	if len(payload.Stable) == 0 || payload.Stable[0].Name != "verify-line-count" {
		t.Fatalf("unexpected payload: %+v", payload)
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

func TestLineAuditFailsWhenNoFilesScanned(t *testing.T) {
	dir := t.TempDir()
	withCwd(t, dir)

	err := RunLineAudit(Config{LineAudit: LineAuditConfig{
		MaxLines:   300,
		Roots:      []string{"missing"},
		Extensions: []string{".go"},
	}})
	if err == nil || !strings.Contains(err.Error(), "0 files scanned") {
		t.Fatalf("expected zero-scan failure, got %v", err)
	}
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func withCwd(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var out bytes.Buffer
	if _, err := io.Copy(&out, reader); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return out.String()
}

package hrns

import (
	"bytes"
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

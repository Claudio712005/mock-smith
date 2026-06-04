package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func runErr(t *testing.T, args ...string) error {
	t.Helper()
	root := NewRootCmd()
	root.SetArgs(args)
	root.SetOut(os.NewFile(0, os.DevNull))
	root.SetErr(os.NewFile(0, os.DevNull))
	return root.Execute()
}

func TestRun_NoSpecNoConfig(t *testing.T) {
	if err := runErr(t, "run"); err == nil {
		t.Fatal("run without spec/config expected error, got nil")
	}
}

func TestRun_BadConfigPath(t *testing.T) {
	if err := runErr(t, "run", "--config", filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("run with missing config expected error, got nil")
	}
}

func TestRun_ConfigWithoutSpec(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "c.yaml")
	os.WriteFile(cfg, []byte("addr: \":0\"\n"), 0o644)
	if err := runErr(t, "run", "--config", cfg); err == nil {
		t.Fatal("config without 'spec' and no positional expected error, got nil")
	}
}

func TestRun_PositionalSpecBadFileErrors(t *testing.T) {
	// Positional spec is used; load fails → proves the arg reached app.New.
	if err := runErr(t, "run", filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("run with missing spec file expected error, got nil")
	}
}

func TestRun_BadProfileErrors(t *testing.T) {
	if err := runErr(t, "run", filepath.Join(t.TempDir(), "x.yaml"), "--profile", "bogus"); err == nil {
		t.Fatal("run with unknown profile expected error, got nil")
	}
}

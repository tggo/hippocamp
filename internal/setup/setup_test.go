package setup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSetup_CreatesFiles(t *testing.T) {
	dir := t.TempDir()

	if err := Setup(dir, ""); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// Check that skill file was created.
	skillPath := filepath.Join(dir, ".claude", "skills", "project-analyze.md")
	info, err := os.Stat(skillPath)
	if err != nil {
		t.Fatalf("skill file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("skill file is empty")
	}

	// Check that hook files were created.
	for _, name := range []string{"hippocamp-pre-query.sh", "hippocamp-post-edit.sh"} {
		hookPath := filepath.Join(dir, ".claude", "hooks", name)
		info, err := os.Stat(hookPath)
		if err != nil {
			t.Fatalf("hook %s not created: %v", name, err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Errorf("hook %s is not executable (mode: %v)", name, info.Mode())
		}
	}
}

func TestSetup_SkipsNewerFiles(t *testing.T) {
	dir := t.TempDir()

	// First setup — creates files.
	if err := Setup(dir, ""); err != nil {
		t.Fatalf("first Setup: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "project-analyze.md")

	// Write custom content to simulate user modification.
	custom := []byte("# my custom skill")
	if err := os.WriteFile(skillPath, custom, 0o644); err != nil {
		t.Fatal(err)
	}

	// Set file mtime to the future.
	future := time.Now().Add(24 * time.Hour)
	os.Chtimes(skillPath, future, future)

	// Second setup with a build time in the past — should NOT overwrite.
	pastBuild := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	if err := Setup(dir, pastBuild); err != nil {
		t.Fatalf("second Setup: %v", err)
	}

	got, _ := os.ReadFile(skillPath)
	if string(got) != string(custom) {
		t.Error("file was overwritten despite being newer than build")
	}
}

func TestSetup_OverwritesOlderFiles(t *testing.T) {
	dir := t.TempDir()

	// First setup.
	if err := Setup(dir, ""); err != nil {
		t.Fatalf("first Setup: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "project-analyze.md")

	// Set file mtime to the past.
	past := time.Now().Add(-48 * time.Hour)
	os.Chtimes(skillPath, past, past)

	// Write stale content.
	os.WriteFile(skillPath, []byte("stale"), 0o644)
	os.Chtimes(skillPath, past, past)

	// Setup with a recent build time — should overwrite.
	recentBuild := time.Now().Format(time.RFC3339)
	if err := Setup(dir, recentBuild); err != nil {
		t.Fatalf("second Setup: %v", err)
	}

	got, _ := os.ReadFile(skillPath)
	if string(got) == "stale" {
		t.Error("stale file was not overwritten by newer build")
	}
}

func TestSetup_DevBuildAlwaysOverwrites(t *testing.T) {
	dir := t.TempDir()

	// First setup.
	if err := Setup(dir, ""); err != nil {
		t.Fatalf("first Setup: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "project-analyze.md")
	os.WriteFile(skillPath, []byte("old content"), 0o644)

	// Setup with empty buildTime (dev build) — should always overwrite.
	if err := Setup(dir, ""); err != nil {
		t.Fatalf("second Setup: %v", err)
	}

	got, _ := os.ReadFile(skillPath)
	if string(got) == "old content" {
		t.Error("dev build did not overwrite existing file")
	}
}

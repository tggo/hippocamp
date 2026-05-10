package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readEmbeddedSkill runs Setup into a temp dir and returns the project-analyze
// skill body. We read through Setup (rather than the embed FS directly) to also
// cover the write path.
func readEmbeddedSkill(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := Setup(dir, ""); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, ".claude", "skills", "project-analyze.md"))
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	return string(body)
}

// TestSkill_NoBrokenProjPrefix guards the fix in PR #4: the `proj:` prefix
// mapping `https://hippocamp.dev/project/` produces invalid TriG because slashes
// are not allowed in a prefixed-name local part. The skill must not advise it.
func TestSkill_NoBrokenProjPrefix(t *testing.T) {
	body := readEmbeddedSkill(t)

	if strings.Contains(body, "prefix=proj uri=https://hippocamp.dev/project/") {
		t.Error("skill still registers the broken proj: prefix; see PR #4")
	}
	if strings.Contains(body, "prefix_add prefix=proj") {
		t.Error("skill registers a proj: prefix — slashes in local part will fail TriG parse")
	}
	// The replacement guidance must be present.
	if !strings.Contains(body, "Do NOT register a `proj:` prefix") {
		t.Error("skill missing the explicit warning against proj: prefix")
	}
}

// TestSkill_DeepScanProcedure guards the Step 3 rewrite: shallow root-only
// scans miss subfolder content (e.g. contacts/). The skill must prescribe a
// recursive scan with named subfolder hints.
func TestSkill_DeepScanProcedure(t *testing.T) {
	body := readEmbeddedSkill(t)

	mustContain := []string{
		"DEEP, not shallow",
		"find . -maxdepth 4",
		"Read every leaf file",
		"Sanity check",
	}
	for _, s := range mustContain {
		if !strings.Contains(body, s) {
			t.Errorf("skill missing deep-scan marker: %q", s)
		}
	}

	// Subfolder hints — at least these bilingual ones.
	hints := []string{"contacts/", "контакти/", "suppliers/", "постачальники/", "decisions/"}
	for _, h := range hints {
		if !strings.Contains(body, h) {
			t.Errorf("skill missing subfolder hint: %q", h)
		}
	}
}

// TestSkill_NoDuplicateQuotesHint catches the post-review nit: `quotes/` was
// listed twice in the subfolder hint line.
func TestSkill_NoDuplicateQuotesHint(t *testing.T) {
	body := readEmbeddedSkill(t)

	for _, line := range strings.Split(body, "\n") {
		if !strings.Contains(line, "Identify content-bearing subfolders") {
			continue
		}
		if n := strings.Count(line, "`quotes/`"); n > 1 {
			t.Errorf("subfolder hint line lists `quotes/` %d times, want 1", n)
		}
		return
	}
	t.Error("subfolder hint line not found")
}

// TestSkill_ExtractionPatternsTable guards the Step 6 patterns table that
// codifies signal→RDF mapping (phone → Entity, ✅ → Decision, etc.).
func TestSkill_ExtractionPatternsTable(t *testing.T) {
	body := readEmbeddedSkill(t)

	if !strings.Contains(body, "Extraction patterns") {
		t.Fatal("skill missing 'Extraction patterns' section")
	}

	// Each row encodes a high-value pattern. Drop one and the agent loses it.
	patterns := []string{
		"`Телефон:`",
		"`hippo:url`",
		"`#supplier`",
		"hippo:Decision",
		"hippo:rationale",
		"`hippo:references`",
		"hippo:Question",
		"`[[wiki-link]]`",
	}
	for _, p := range patterns {
		if !strings.Contains(body, p) {
			t.Errorf("extraction patterns table missing: %q", p)
		}
	}
}

// TestSkill_ImportTargetsDefaultGraphNote keeps the warning about
// `graph action=import` writing to the default graph (the trap that motivated
// the proj: fix in the first place).
func TestSkill_ImportTargetsDefaultGraphNote(t *testing.T) {
	body := readEmbeddedSkill(t)

	if !strings.Contains(body, "graph action=import") || !strings.Contains(body, "default graph") {
		t.Error("skill missing the note that graph action=import writes to default graph")
	}
}

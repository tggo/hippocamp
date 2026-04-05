// Package setup embeds Claude Code integration files (hooks, skills) and
// writes them to the project directory on first launch or when the binary
// is newer than the files on disk.
package setup

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed embedded
var content embed.FS

// Setup writes Claude Code hooks and skills to projectRoot/.claude/.
// If buildTimeStr is non-empty (RFC 3339), files are only overwritten
// when the build time is newer than the file's mtime on disk.
// A zero buildTimeStr (dev builds) means "always overwrite".
func Setup(projectRoot, buildTimeStr string) error {
	buildTime, _ := time.Parse(time.RFC3339, buildTimeStr)

	return fs.WalkDir(content, "embedded/claude", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		// "embedded/claude/skills/foo.md" → ".claude/skills/foo.md"
		inner := strings.TrimPrefix(path, "embedded/claude")     // "/skills/foo.md"
		destPath := filepath.Join(projectRoot, ".claude"+inner)

		// Check if existing file is newer than this build.
		if !buildTime.IsZero() {
			if info, statErr := os.Stat(destPath); statErr == nil {
				if info.ModTime().After(buildTime) {
					return nil // file on disk is newer, skip
				}
			}
		}

		data, readErr := content.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read embedded %s: %w", path, readErr)
		}

		if mkErr := os.MkdirAll(filepath.Dir(destPath), 0o755); mkErr != nil {
			return fmt.Errorf("mkdir %s: %w", filepath.Dir(destPath), mkErr)
		}

		// Shell scripts need execute permission.
		perm := os.FileMode(0o644)
		if strings.HasSuffix(destPath, ".sh") {
			perm = 0o755
		}

		if writeErr := os.WriteFile(destPath, data, perm); writeErr != nil {
			return fmt.Errorf("write %s: %w", destPath, writeErr)
		}

		return nil
	})
}

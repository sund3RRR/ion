// Package profile analyzes an adapted package's store path and materializes
// its anchor paths (bin/, lib/, share/applications, share/icons, etc.) as
// symlinks into a profile directory.
package profile

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultAnchors lists the store-relative directories that get materialized
// into a profile. Only files reachable under one of these prefixes are
// considered part of a package's system integration surface.
var DefaultAnchors = []string{
	"bin",
	"sbin",
	"lib",
	"libexec",
	"share/applications",
	"share/icons",
	"share/man",
	"share/fonts",
	"share/dbus-1",
	"share/systemd",
}

// Entry describes one anchor file to materialize into a profile.
type Entry struct {
	// RelativePath is the file's path relative to the store output root.
	RelativePath string
	// StorePath is the file's absolute path inside the store output.
	StorePath string
	// MaterializedPath is the absolute symlink target path inside the
	// profile directory.
	MaterializedPath string
	// Executable reports whether the store file is executable.
	Executable bool
}

// Plan is the set of anchor entries to materialize for one package output.
type Plan struct {
	Entries []Entry
}

// Writer analyzes store outputs and materializes their anchors into profile
// directories.
type Writer struct {
	anchors []string
}

// New creates a Writer using DefaultAnchors.
func New() *Writer {
	return &Writer{anchors: DefaultAnchors}
}

// Plan walks storePath under the writer's anchor prefixes and computes the
// materialized symlink path for each anchor file under profileDir. Plan only
// reads the filesystem; it does not modify the profile.
func (w *Writer) Plan(profileDir, storePath string) (*Plan, error) {
	var entries []Entry

	for _, anchor := range w.anchors {
		anchorRoot := filepath.Join(storePath, anchor)

		info, err := os.Lstat(anchorRoot)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("profile: stat anchor %q: %w", anchorRoot, err)
		}
		if !info.IsDir() {
			continue
		}

		err = filepath.WalkDir(anchorRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(storePath, path)
			if err != nil {
				return fmt.Errorf("profile: relative path for %q: %w", path, err)
			}

			fileInfo, err := d.Info()
			if err != nil {
				return fmt.Errorf("profile: stat %q: %w", path, err)
			}

			entries = append(entries, Entry{
				RelativePath:     relPath,
				StorePath:        path,
				MaterializedPath: filepath.Join(profileDir, relPath),
				Executable:       fileInfo.Mode()&0o111 != 0,
			})

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("profile: walk anchor %q: %w", anchorRoot, err)
		}
	}

	return &Plan{Entries: entries}, nil
}

// Link creates a symlink at e.MaterializedPath pointing at e.StorePath,
// replacing any existing file or symlink at that path.
func (w *Writer) Link(e Entry) error {
	if err := os.MkdirAll(filepath.Dir(e.MaterializedPath), 0o755); err != nil {
		return fmt.Errorf("profile: create parent directory for %q: %w", e.MaterializedPath, err)
	}

	if err := w.Unlink(e.MaterializedPath); err != nil {
		return err
	}

	if err := os.Symlink(e.StorePath, e.MaterializedPath); err != nil {
		return fmt.Errorf("profile: link %q to %q: %w", e.MaterializedPath, e.StorePath, err)
	}

	return nil
}

// Unlink removes the file or symlink at path, if present.
func (w *Writer) Unlink(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("profile: remove %q: %w", path, err)
	}

	return nil
}

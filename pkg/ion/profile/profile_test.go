package profile_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/sund3RRR/ion/pkg/ion/profile"
)

func TestPlanSelectsOnlyAnchors(t *testing.T) {
	storePath := t.TempDir()
	profileDir := t.TempDir()

	writeFile(t, filepath.Join(storePath, "bin", "foo"), 0o755)
	writeFile(t, filepath.Join(storePath, "share", "applications", "foo.desktop"), 0o644)
	writeFile(t, filepath.Join(storePath, "nix-support", "propagated-build-inputs"), 0o644)

	w := profile.New()
	plan, err := w.Plan(profileDir, storePath)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	relPaths := make([]string, 0, len(plan.Entries))
	for _, e := range plan.Entries {
		relPaths = append(relPaths, e.RelativePath)
	}
	sort.Strings(relPaths)

	want := []string{"bin/foo", "share/applications/foo.desktop"}
	if len(relPaths) != len(want) {
		t.Fatalf("Plan() entries = %v, want %v", relPaths, want)
	}
	for i := range want {
		if relPaths[i] != want[i] {
			t.Fatalf("Plan() entries = %v, want %v", relPaths, want)
		}
	}

	var binEntry, desktopEntry profile.Entry
	for _, e := range plan.Entries {
		switch e.RelativePath {
		case "bin/foo":
			binEntry = e
		case "share/applications/foo.desktop":
			desktopEntry = e
		}
	}

	if !binEntry.Executable {
		t.Fatalf("bin/foo Executable = false, want true")
	}
	if desktopEntry.Executable {
		t.Fatalf("share/applications/foo.desktop Executable = true, want false")
	}

	wantMaterialized := filepath.Join(profileDir, "bin", "foo")
	if binEntry.MaterializedPath != wantMaterialized {
		t.Fatalf("bin/foo MaterializedPath = %q, want %q", binEntry.MaterializedPath, wantMaterialized)
	}

	wantStorePath := filepath.Join(storePath, "bin", "foo")
	if binEntry.StorePath != wantStorePath {
		t.Fatalf("bin/foo StorePath = %q, want %q", binEntry.StorePath, wantStorePath)
	}
}

func TestPlanSelectsSystemdUnits(t *testing.T) {
	storePath := t.TempDir()
	profileDir := t.TempDir()

	writeFile(t, filepath.Join(storePath, "lib", "systemd", "system", "foo.service"), 0o644)
	writeFile(t, filepath.Join(storePath, "share", "systemd", "user", "foo.service"), 0o644)

	w := profile.New()
	plan, err := w.Plan(profileDir, storePath)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	relPaths := make([]string, 0, len(plan.Entries))
	for _, e := range plan.Entries {
		relPaths = append(relPaths, e.RelativePath)
	}
	sort.Strings(relPaths)

	want := []string{
		"lib/systemd/system/foo.service",
		"share/systemd/user/foo.service",
	}
	if len(relPaths) != len(want) {
		t.Fatalf("Plan() entries = %v, want %v", relPaths, want)
	}
	for i := range want {
		if relPaths[i] != want[i] {
			t.Fatalf("Plan() entries = %v, want %v", relPaths, want)
		}
	}
}

func TestPlanSkipsMissingAnchors(t *testing.T) {
	storePath := t.TempDir()
	profileDir := t.TempDir()

	w := profile.New()
	plan, err := w.Plan(profileDir, storePath)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(plan.Entries) != 0 {
		t.Fatalf("Plan() entries = %v, want empty", plan.Entries)
	}
}

func TestLinkCreatesSymlinkAndUnlinkRemovesIt(t *testing.T) {
	storePath := t.TempDir()
	profileDir := t.TempDir()

	storeFile := filepath.Join(storePath, "bin", "foo")
	writeFile(t, storeFile, 0o755)

	entry := profile.Entry{
		RelativePath:     "bin/foo",
		StorePath:        storeFile,
		MaterializedPath: filepath.Join(profileDir, "bin", "foo"),
		Executable:       true,
	}

	w := profile.New()
	if err := w.Link(entry); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	target, err := os.Readlink(entry.MaterializedPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if target != storeFile {
		t.Fatalf("Readlink() = %q, want %q", target, storeFile)
	}

	if err := w.Link(entry); err != nil {
		t.Fatalf("Link() second call error = %v", err)
	}

	if err := w.Unlink(entry.MaterializedPath); err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}
	if _, err := os.Lstat(entry.MaterializedPath); !os.IsNotExist(err) {
		t.Fatalf("Lstat() after Unlink() error = %v, want not exist", err)
	}

	if err := w.Unlink(entry.MaterializedPath); err != nil {
		t.Fatalf("Unlink() on absent path error = %v, want nil", err)
	}
}

func writeFile(t *testing.T, path string, mode os.FileMode) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("content"), mode); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

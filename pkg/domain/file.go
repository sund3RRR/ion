package domain

// FileEntry describes one materialized anchor file owned by an installed
// package.
type FileEntry struct {
	// Executable reports whether the source file has any executable bit.
	Executable bool
	// RelativePath is the path relative to the package output root.
	RelativePath string
	// MaterializedPath is the absolute path inside the profile.
	MaterializedPath string
	// CreatedAt is the Unix timestamp when the file record was created.
	CreatedAt int64
}

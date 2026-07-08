package domain

type FileEntry struct {
	Executable       bool
	RelativePath     string
	MaterializedPath string
	CreatedAt        int64
}

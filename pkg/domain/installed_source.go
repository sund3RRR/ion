package domain

// InstalledSource groups one installed profile package with its source flake
// revision and materialized files.
type InstalledSource struct {
	// Profile is the profile that owns the installation.
	Profile Profile
	// FlakeRev is the source flake revision used for the package metadata.
	FlakeRev FlakeRev
	// Packages contains the installed package output records.
	Packages []ProfilePackage
	// Files contains the materialized profile files owned by the source.
	Files []FileEntry
}

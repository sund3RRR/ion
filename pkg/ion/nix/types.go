package nix

// Package describes one resolved package derivation.
type Package struct {
	// Attr is the user-facing package attribute.
	Attr string `json:"attr"`
	// AttrPath is the exact flake output path that resolved successfully.
	AttrPath []string `json:"attr_path"`
	// System is the Nix system used for resolution.
	System string `json:"system"`
	// Name is the package name exposed by Nix.
	Name string `json:"name"`
	// Version is the package version exposed by Nix, when available.
	Version string `json:"version"`
	// DrvPath is the derivation store path to realize.
	DrvPath string `json:"drv_path"`
	// OutputNames lists declared derivation output names, when available.
	OutputNames []string `json:"output_names"`
}

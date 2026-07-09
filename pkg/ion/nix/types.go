// Package nix adapts gonix operations for ION package resolution and
// realization.
package nix

// RealizedPackage describes a resolved package and one realized output.
type RealizedOutput struct {
	// OutputName is the realized derivation output name.
	OutputName string
	// StorePath is the canonical Nix store path for the output.
	StorePath string
	// RealPath is the resolved filesystem path for StorePath.
	RealPath string
	// DrvPath is the derivation store path to realize.
	DrvPath string
	// Name is the realized output name reported by Nix.
	Name string
	// Hash is the store hash of the realized output.
	Hash [20]byte
}

// Package describes one resolved package derivation.
type ResolvedPackage struct {
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

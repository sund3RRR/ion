package domain

import "time"

// ProfilePackage describes one package output installed into a profile.
type ProfilePackage struct {
	// System is the Nix platform system for the realized output.
	System string
	// DrvPath is the adapted derivation path.
	DrvPath string
	// StorePath is the realized output path.
	StorePath string
	// OutputName is the realized derivation output name.
	OutputName string
	// CreatedAt is the time the profile package was recorded.
	CreatedAt time.Time
	// UpdatedAt is the time the profile package record last changed.
	UpdatedAt time.Time
	// Package is the indexed package metadata for the installed output.
	Package Package
}

// Package describes indexed package metadata.
type Package struct {
	// Attr is the package attribute within its flake.
	Attr string
	// Name is the package name.
	Name string
	// Description summarizes the package.
	Description string
	// Version is the package version.
	Version string
	// License describes the package license when indexed.
	License License
	// Outputs lists available derivation output names.
	Outputs []string
	// Platforms lists systems where the package is available.
	Platforms []string
	// CreatedAt is the time the package metadata was indexed.
	CreatedAt time.Time
	// UpdatedAt is the time the package metadata last changed.
	UpdatedAt time.Time
}

// License describes package license metadata.
type License struct {
	// Open reports whether the license is considered open.
	Open bool
	// Name is the license identifier or display name.
	Name string
}

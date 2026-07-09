package domain

import (
	"time"

	"github.com/sund3RRR/gonix/flake"
)

// Flake describes a configured package source flake.
type Flake struct {
	// Owner is the user or authority that owns the alias.
	Owner string
	// Alias is the user-facing name for the flake.
	Alias string
	// Ref is the Nix flake reference.
	Ref string
	// CreatedAt is the time the flake was indexed.
	CreatedAt time.Time
	// UpdatedAt is the time the flake metadata last changed.
	UpdatedAt time.Time
}

// FlakeRev describes one locked revision of a flake.
type FlakeRev struct {
	// Fingerprint uniquely identifies the locked revision for a flake.
	Fingerprint string
	// LockInfo is the Nix lock graph for the revision.
	LockInfo flake.LockInfo
	// CreatedAt is the time the revision was indexed.
	CreatedAt time.Time
	// Flake is the source flake that owns the revision.
	Flake Flake
}

package domain

// ProfileKind identifies the scope of a profile.
type ProfileKind string

const (
	// ProfileKindSystem identifies the machine-wide system profile.
	ProfileKindSystem ProfileKind = "system"
	// ProfileKindUser identifies a user-owned profile.
	ProfileKindUser ProfileKind = "user"
)

// SystemProfile is the reserved name of the system profile.
const SystemProfile = "system"

// Decision chooses how ApplyInstall handles a plan whose anchors collide with
// an already-installed package.
type Decision int

const (
	// DecisionNone records that no conflict resolution should be applied.
	DecisionNone Decision = iota
	// DecisionOverwrite removes conflicting installed packages.
	DecisionOverwrite
)

// String returns the stable text form of d.
func (d Decision) String() string {
	switch d {
	case DecisionNone:
		return "none"
	case DecisionOverwrite:
		return "overwrite"
	default:
		return "unknown"
	}
}

// IsValid reports whether d is a recognized decision value.
func (d Decision) IsValid() bool {
	switch d {
	case DecisionNone, DecisionOverwrite:
		return true
	default:
		return false
	}
}

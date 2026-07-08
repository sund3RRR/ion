package domain

type ProfileKind string

const (
	ProfileKindSystem ProfileKind = "system"
	ProfileKindUser   ProfileKind = "user"
)

const SystemProfile = "system"

// ConflictDecision chooses how ApplyInstall handles a plan whose anchors
// collide with an already-installed package.
type Decision int

const (
	DecisionNone Decision = iota
	DecisionOverwrite
)

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

func (d Decision) IsValid() bool {
	switch d {
	case DecisionNone, DecisionOverwrite:
		return true
	default:
		return false
	}
}

package domain

// Profile describes a profile that can receive materialized package anchors.
type Profile struct {
	// Kind identifies whether the profile is system or user scoped.
	Kind ProfileKind
	// Name is the profile name.
	Name string
	// Owner is the user or authority that owns the profile.
	Owner string
	// Path is the profile directory where anchors are materialized.
	Path string
	// CreatedAt is the Unix timestamp when the profile was created.
	CreatedAt int64 `json:"created_at"`
}

package domain

type Profile struct {
	Kind      ProfileKind
	Name      string
	Owner     string
	Path      string
	CreatedAt int64 `json:"created_at"`
}

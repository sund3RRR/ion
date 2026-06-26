package version

import "strings"

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func String() string {
	parts := make([]string, 0, 3)
	if Commit != "" && Commit != "unknown" {
		parts = append(parts, Commit)
	}
	if Date != "" && Date != "unknown" {
		parts = append(parts, Date)
	}

	if len(parts) == 0 {
		return Version
	}

	return Version + " (" + strings.Join(parts, ", ") + ")"
}

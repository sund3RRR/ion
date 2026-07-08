package domain

import (
	"time"

	"github.com/sund3RRR/gonix/flake"
)

type Flake struct {
	Owner       string
	Alias       string
	Ref         string
	Fingerprint string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FlakeRev struct {
	Fingerprint string
	LockInfo    flake.LockInfo
	CreatedAt   time.Time
	Flake       Flake
}

package types

import "time"

type ProfileKind string

const (
	ProfileKindSystem ProfileKind = "system"
	ProfileKindUser   ProfileKind = "user"
	ProfileKindCustom ProfileKind = "custom"
)

type ProfileRef struct {
	Kind ProfileKind
	Name string
	Dir  string
}

type PackageRef struct {
	Flake string
	Attr  string
	Name  string
}

type OperationStatus string

const (
	OperationStatusPending   OperationStatus = "pending"
	OperationStatusRunning   OperationStatus = "running"
	OperationStatusSucceeded OperationStatus = "succeeded"
	OperationStatusFailed    OperationStatus = "failed"
)

type Operation struct {
	ID         string
	Status     OperationStatus
	StartedAt  time.Time
	FinishedAt *time.Time
	Error      string
}

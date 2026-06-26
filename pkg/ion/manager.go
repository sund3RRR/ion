package ion

import (
	"context"
	"time"

	"github.com/sund3RRR/ion/pkg/types"
)

type InstallRequest struct {
	Package types.PackageRef
	Profile types.ProfileRef
}

type RemoveRequest struct {
	Package types.PackageRef
	Profile types.ProfileRef
}

type UpdateRequest struct {
	Profile types.ProfileRef
}

type SearchRequest struct {
	Query string
}

type ListRequest struct {
	Profile types.ProfileRef
}

type Manager interface {
	Install(context.Context, InstallRequest) (*types.Operation, error)
	Remove(context.Context, RemoveRequest) (*types.Operation, error)
	Update(context.Context, UpdateRequest) (*types.Operation, error)
	Search(context.Context, SearchRequest) ([]types.PackageRef, error)
	List(context.Context, ListRequest) ([]types.PackageRef, error)
}

type NoopManager struct{}

func NewManager() Manager {
	return NoopManager{}
}

func (NoopManager) Install(context.Context, InstallRequest) (*types.Operation, error) {
	return noopOperation(), nil
}

func (NoopManager) Remove(context.Context, RemoveRequest) (*types.Operation, error) {
	return noopOperation(), nil
}

func (NoopManager) Update(context.Context, UpdateRequest) (*types.Operation, error) {
	return noopOperation(), nil
}

func (NoopManager) Search(context.Context, SearchRequest) ([]types.PackageRef, error) {
	return nil, nil
}

func (NoopManager) List(context.Context, ListRequest) ([]types.PackageRef, error) {
	return nil, nil
}

func noopOperation() *types.Operation {
	now := time.Now()
	return &types.Operation{
		Status:     types.OperationStatusSucceeded,
		StartedAt:  now,
		FinishedAt: &now,
	}
}

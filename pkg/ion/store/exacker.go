package store

import (
	"context"

	"github.com/sund3RRR/ion/pkg/domain"
	"github.com/sund3RRR/ion/pkg/ion/store/sqlc"
)

type Exacker struct {
	q *sqlc.Queries
}

func NewExacker(q *sqlc.Queries) *Exacker {
	return &Exacker{
		q: q,
	}
}

func (e *Exacker) GetLatestFlakeRev(ctx context.Context, owner string, alias string) (*domain.FlakeRev, error)
func (e *Exacker) GetPackage(ctx context.Context, flakeRev *domain.FlakeRev, attr string) (*domain.Package, error)
func (e *Exacker) GetProfilePackage(ctx context.Context, path string) (*domain.ProfilePackage, error)
func (e *Exacker) GetConflictedPackages(ctx context.Context, files []string) ([]*domain.InstalledSource, error)
func (e *Exacker) GetProfile(ctx context.Context, name string, kind domain.ProfileKind) (*domain.Profile, error)
func (e *Exacker) DeleteProfilePackage(ctx context.Context, source *domain.InstalledSource) error
func (e *Exacker) ListProfilePackageFiles(ctx context.Context, source *domain.InstalledSource) ([]string, error)
func (e *Exacker) CreateProfilePackage(ctx context.Context, source *domain.InstalledSource) error

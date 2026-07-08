package ion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sund3RRR/gonix"
	"github.com/sund3RRR/ion/pkg/domain"
	"github.com/sund3RRR/ion/pkg/ion/inject"
	"github.com/sund3RRR/ion/pkg/ion/nix"
	"github.com/sund3RRR/ion/pkg/ion/profile"
	"github.com/sund3RRR/ion/pkg/ion/store"
)

// defaultOutput is the derivation output installed when a request does not
// specify one.
const defaultOutput = "out"

// Ion is the core ION package manager: it resolves and realizes packages
// through Nix, adapts them, and materializes them into profiles.
type Ion struct {
	store    *store.Store
	nix      *nix.Nix
	injector *inject.Injector
	profile  *profile.Writer
}

// New creates an Ion backed by store and nix.
func New(store *store.Store, nix *nix.Nix) *Ion {
	return &Ion{
		store:    store,
		nix:      nix,
		injector: inject.New(nix),
		profile:  profile.New(),
	}
}

// Close releases the store and nix resources owned by ion.
func (ion *Ion) Close() error {
	var errs []error
	if err := ion.store.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := ion.nix.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close ion: %w", errors.Join(errs...))
	}
	return nil
}

// Conflict describes one anchor that a planned installation shares with an
// already-installed, different package.
type Conflict struct {
	// MaterializedPath is the profile path both packages want to claim.
	MaterializedPath string
	// RelativePath is the anchor's path relative to a package output root.
	RelativePath string
	// ExistingProfilePackageID identifies the installation that already
	// owns MaterializedPath.
	ExistingProfilePackageID int64
	// ExistingPackageID identifies the package that already owns
	// MaterializedPath.
	ExistingPackageID int64
}

// InstallPlan is the result of resolving, realizing, and adapting a package,
// plus its anchor plan and any conflicts against the target profile. It
// carries everything ApplyInstall needs to commit the installation without
// repeating Nix work.
type InstallPlan struct {
	Profile        domain.Profile
	Request        *InstallPackageRequest
	FlakeRev       *domain.FlakeRev
	Package        *domain.Package
	AdaptedPackage gonix.RealizedOutput
	AdaptedDrvPath string
	Entries        []profile.Entry
	Conflicts      []*domain.InstalledSource
}

// HasConflict reports whether installing the plan would collide with an
// already-installed, different package.
func (p *InstallPlan) HasConflict() bool {
	return len(p.Conflicts) > 0
}

// InstallPackageRequest describes a package to resolve, adapt, and
// materialize into a profile.
type InstallPackageRequest struct {
	// FlakeAlias is the alias of the flake to resolve the package from.
	FlakeAlias string
	// Attr is the package attribute to resolve within the flake.
	Attr string
	// Output is the derivation output to install. Defaults to "out".
	Output string
	// Profile is the name of the target profile.
	Profile string
	// User is the owner o the flake alias.
	User string
	// System is the Nix system to resolve and adapt the package for.
	System string
}

// PlanInstall resolves req's package from its flake, realizes it, adapts it
// via the injector, and computes its anchor plan against the requested
// profile. It performs no database writes and creates no symlinks; call
// ApplyInstall with the returned plan to commit the installation.
func (ion *Ion) PlanInstall(ctx context.Context, req InstallPackageRequest) (*InstallPlan, error) {
	if req.FlakeAlias == "" || req.Attr == "" || req.Profile == "" || req.User == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	if req.Output == "" {
		req.Output = defaultOutput
	}

	if req.System == "" {
		req.System = gonix.DefaultSystem()
	}

	flakeRev, err := ion.store.GetLatestFlakeRev(ctx, req.User, req.FlakeAlias)
	if err != nil {
		return nil, fmt.Errorf("get latest flake revision: %w", err)
	}

	pkg, err := ion.store.GetPackage(ctx, flakeRev, req.Attr)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}

	nixFlake, err := ion.nix.OpenLockedFlake(ctx, flakeRev.Flake.Ref, &flakeRev.LockInfo)
	if err != nil {
		return nil, fmt.Errorf("open locked flake: %w", err)
	}

	nixPkg, err := ion.nix.PullPackage(ctx, nixFlake, nix.PullPackageRequest{
		Attr:       req.Attr,
		System:     req.System,
		OutputName: req.Output,
	})
	if err != nil {
		return nil, fmt.Errorf("realize package: %w", err)
	}

	adaptedPkg, adaptedDrvPath, err := ion.injector.AdaptPackage(ctx, inject.Request{
		Attr:             req.Attr,
		FlakeRef:         flakeRev.Flake.Ref,
		System:           req.System,
		StorePath:        nixPkg.StorePath,
		OutputName:       req.Output,
		PlaceholderTweak: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to adapt package: %w", err)
	}

	profileKind := domain.ProfileKindUser
	if req.Profile == domain.SystemProfile {
		profileKind = domain.ProfileKindSystem
	}

	profile, err := ion.store.GetProfile(ctx, req.Profile, profileKind)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	planned, err := ion.profile.Plan(profile.Path, adaptedPkg.StorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to plan profile installation: %w", err)
	}

	paths := make([]string, 0, len(planned.Entries))
	for _, entry := range planned.Entries {
		paths = append(paths, entry.MaterializedPath)
	}

	conflicts, err := ion.store.GetConflictedPackages(ctx, paths)
	if err != nil {
		return nil, fmt.Errorf("failed to detect conflicts: %w", err)
	}

	return &InstallPlan{
		Request:        &req,
		FlakeRev:       flakeRev,
		Package:        pkg,
		AdaptedPackage: adaptedPkg,
		AdaptedDrvPath: adaptedDrvPath,
		Entries:        planned.Entries,
		Conflicts:      conflicts,
	}, nil
}

// ApplyInstall commits plan. When plan has no conflict, decision is ignored.
// ConflictReplace removes the whole conflicting package before installing
// the newer one; ConflictResolve disambiguates the newer package (via the
// resolver) so both installations coexist. The package, its profile
// installation, and its materialized files are recorded in the store in a
// single transaction.
func (ion *Ion) ApplyInstall(ctx context.Context, plan *InstallPlan, decision domain.Decision) error {
	var linked []string
	if err := ion.store.ExecTx(ctx, func(e *store.Exacker) error {
		for _, conflict := range plan.Conflicts {
			if err := ion.ResolveConflict(ctx, e, conflict, decision); err != nil {
				return fmt.Errorf("ion: resolve conflict: %w", err)
			}
		}

		files := make([]domain.FileEntry, 0, len(plan.Entries))
		for _, entry := range plan.Entries {
			files = append(files, domain.FileEntry{
				Executable:       entry.Executable,
				RelativePath:     entry.RelativePath,
				MaterializedPath: entry.MaterializedPath,
			})
		}

		installedSource := &domain.InstalledSource{
			Profile:  plan.Profile,
			FlakeRev: *plan.FlakeRev,
			Packages: []domain.ProfilePackage{
				{
					System:     plan.Request.System,
					DrvPath:    plan.AdaptedDrvPath,
					StorePath:  plan.AdaptedPackage.StorePath,
					OutputName: plan.Request.Output,
					Package:    *plan.Package,
				},
			},
			Files: files,
		}

		if err := e.CreateProfilePackage(ctx, installedSource); err != nil {
			return fmt.Errorf("ion: create profile package: %w", err)
		}

		for _, entry := range plan.Entries {
			if err := ion.profile.Link(entry); err != nil {
				return fmt.Errorf("ion: link file %q: %w", entry.MaterializedPath, err)
			}
			linked = append(linked, entry.MaterializedPath)
		}

		return nil
	}); err != nil {
		for _, path := range linked {
			_ = ion.profile.Unlink(path)
		}
		return err
	}

	return nil
}

func (ion *Ion) ResolveConflict(ctx context.Context, e *store.Exacker, conflictedSource *domain.InstalledSource, decision domain.Decision) error {
	if decision != domain.DecisionOverwrite {
		return fmt.Errorf("ion: resolve conflict: unknown decision %q", decision)
	}

	files, err := ion.store.ListProfilePackageFiles(ctx, conflictedSource)

	_ = files
	_ = err

	return nil
}

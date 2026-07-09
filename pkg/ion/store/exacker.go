package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sund3RRR/gonix/flake"

	"github.com/sund3RRR/ion/pkg/domain"
	"github.com/sund3RRR/ion/pkg/ion/store/sqlc"
)

// Exacker executes store operations against either the main database handle
// or an active transaction.
type Exacker struct {
	q *sqlc.Queries
}

// NewExacker creates an Exacker backed by q.
func NewExacker(q *sqlc.Queries) *Exacker {
	return &Exacker{
		q: q,
	}
}

// GetLatestFlakeRev returns the newest indexed revision for owner and alias.
func (e *Exacker) GetLatestFlakeRev(ctx context.Context, owner string, alias string) (*domain.FlakeRev, error) {
	flakeRow, err := e.q.GetFlakeByOwnerAlias(ctx, sqlc.GetFlakeByOwnerAliasParams{
		Owner: owner,
		Alias: alias,
	})
	if err != nil {
		return nil, fmt.Errorf("store: get flake %q/%q: %w", owner, alias, err)
	}

	revRow, err := e.q.GetLatestFlakeRevision(ctx, flakeRow.ID)
	if err != nil {
		return nil, fmt.Errorf("store: get latest flake revision for %q/%q: %w", owner, alias, err)
	}

	rev, err := e.flakeRevFromRows(ctx, revRow, flakeRow)
	if err != nil {
		return nil, err
	}

	return &rev, nil
}

// GetPackage returns package metadata for attr in flakeRev.
func (e *Exacker) GetPackage(ctx context.Context, flakeRev *domain.FlakeRev, attr string) (*domain.Package, error) {
	if flakeRev == nil {
		return nil, errors.New("store: get package: nil flake revision")
	}

	revRow, err := e.getFlakeRevRow(ctx, *flakeRev)
	if err != nil {
		return nil, err
	}

	pkgRow, err := e.q.GetPackageByRevisionAttr(ctx, sqlc.GetPackageByRevisionAttrParams{
		FlakeRevisionID: revRow.ID,
		Attr:            attr,
	})
	if err != nil {
		return nil, fmt.Errorf("store: get package %q: %w", attr, err)
	}

	pkg, err := e.packageFromRow(ctx, pkgRow)
	if err != nil {
		return nil, err
	}

	return &pkg, nil
}

// GetProfilePackage returns the installed package that owns materialized path.
func (e *Exacker) GetProfilePackage(ctx context.Context, path string) (*domain.ProfilePackage, error) {
	fileRow, err := e.q.GetFileByMaterializedPath(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("store: get file by materialized path %q: %w", path, err)
	}

	profilePackageRow, err := e.q.GetProfilePackage(ctx, fileRow.ProfilePackageID)
	if err != nil {
		return nil, fmt.Errorf("store: get profile package for %q: %w", path, err)
	}

	profilePackage, err := e.profilePackageFromRow(ctx, profilePackageRow)
	if err != nil {
		return nil, err
	}

	return &profilePackage, nil
}

// GetConflictedPackages returns installed packages that already own any path
// in files.
func (e *Exacker) GetConflictedPackages(ctx context.Context, files []string) ([]*domain.InstalledSource, error) {
	var sources []*domain.InstalledSource
	seen := make(map[int64]struct{})

	for _, path := range files {
		fileRow, err := e.q.GetFileByMaterializedPath(ctx, path)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, fmt.Errorf("store: get conflict for %q: %w", path, err)
		}
		if _, ok := seen[fileRow.ProfilePackageID]; ok {
			continue
		}
		seen[fileRow.ProfilePackageID] = struct{}{}

		profilePackageRow, err := e.q.GetProfilePackage(ctx, fileRow.ProfilePackageID)
		if err != nil {
			return nil, fmt.Errorf("store: get conflicting profile package for %q: %w", path, err)
		}

		source, err := e.installedSourceFromProfilePackage(ctx, profilePackageRow)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	return sources, nil
}

// GetProfile returns the profile identified by name and kind.
func (e *Exacker) GetProfile(ctx context.Context, name string, kind domain.ProfileKind) (*domain.Profile, error) {
	row, err := e.q.GetProfileByKindName(ctx, sqlc.GetProfileByKindNameParams{
		Kind: string(kind),
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("store: get profile %s/%s: %w", kind, name, err)
	}

	profile := profileFromRow(row)
	return &profile, nil
}

// DeleteProfilePackage deletes source's profile package record and cascades
// its file records.
func (e *Exacker) DeleteProfilePackage(ctx context.Context, source *domain.InstalledSource) error {
	row, err := e.getProfilePackageRow(ctx, source)
	if err != nil {
		return err
	}

	if err := e.q.DeleteProfilePackage(ctx, row.ID); err != nil {
		return fmt.Errorf("store: delete profile package: %w", err)
	}

	return nil
}

// ListProfilePackageFiles returns the materialized paths owned by source.
func (e *Exacker) ListProfilePackageFiles(ctx context.Context, source *domain.InstalledSource) ([]string, error) {
	row, err := e.getProfilePackageRow(ctx, source)
	if err != nil {
		return nil, err
	}

	fileRows, err := e.q.ListFilesByProfilePackage(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("store: list profile package files: %w", err)
	}

	paths := make([]string, 0, len(fileRows))
	for _, fileRow := range fileRows {
		paths = append(paths, fileRow.MaterializedPath)
	}

	return paths, nil
}

// CreateProfilePackage records source's installed profile package and files.
func (e *Exacker) CreateProfilePackage(ctx context.Context, source *domain.InstalledSource) error {
	if source == nil {
		return errors.New("store: create profile package: nil source")
	}
	if len(source.Packages) != 1 {
		return fmt.Errorf("store: create profile package: got %d packages, want 1", len(source.Packages))
	}

	profileRow, err := e.getProfileRow(ctx, source.Profile)
	if err != nil {
		return err
	}

	revRow, err := e.getFlakeRevRow(ctx, source.FlakeRev)
	if err != nil {
		return err
	}

	profilePackage := source.Packages[0]
	packageRow, err := e.q.GetPackageByRevisionAttr(ctx, sqlc.GetPackageByRevisionAttrParams{
		FlakeRevisionID: revRow.ID,
		Attr:            profilePackage.Package.Attr,
	})
	if err != nil {
		return fmt.Errorf("store: get package %q: %w", profilePackage.Package.Attr, err)
	}

	platformRow, err := e.q.GetPlatformByName(ctx, profilePackage.System)
	if err != nil {
		return fmt.Errorf("store: get platform %q: %w", profilePackage.System, err)
	}

	created, err := e.q.CreateProfilePackage(ctx, sqlc.CreateProfilePackageParams{
		ProfileID:  profileRow.ID,
		PackageID:  packageRow.ID,
		PlatformID: platformRow.ID,
		OutputName: profilePackage.OutputName,
		DrvPath:    profilePackage.DrvPath,
		StorePath:  profilePackage.StorePath,
	})
	if err != nil {
		return fmt.Errorf("store: create profile package: %w", err)
	}

	if err := e.q.DeleteFilesByProfilePackage(ctx, created.ID); err != nil {
		return fmt.Errorf("store: reset profile package files: %w", err)
	}

	for _, file := range source.Files {
		if _, err := e.q.CreateFile(ctx, sqlc.CreateFileParams{
			ProfilePackageID: created.ID,
			Executable:       file.Executable,
			RelativePath:     file.RelativePath,
			MaterializedPath: file.MaterializedPath,
		}); err != nil {
			return fmt.Errorf("store: create file %q: %w", file.MaterializedPath, err)
		}
	}

	return nil
}

func (e *Exacker) installedSourceFromProfilePackage(ctx context.Context, row sqlc.ProfilePackage) (*domain.InstalledSource, error) {
	profileRow, err := e.q.GetProfile(ctx, row.ProfileID)
	if err != nil {
		return nil, fmt.Errorf("store: get profile for installed source: %w", err)
	}

	packageRow, err := e.q.GetPackage(ctx, row.PackageID)
	if err != nil {
		return nil, fmt.Errorf("store: get package for installed source: %w", err)
	}

	revRow, err := e.q.GetFlakeRevision(ctx, packageRow.FlakeRevisionID)
	if err != nil {
		return nil, fmt.Errorf("store: get flake revision for installed source: %w", err)
	}

	flakeRow, err := e.q.GetFlake(ctx, revRow.FlakeID)
	if err != nil {
		return nil, fmt.Errorf("store: get flake for installed source: %w", err)
	}

	rev, err := e.flakeRevFromRows(ctx, revRow, flakeRow)
	if err != nil {
		return nil, err
	}

	profilePackage, err := e.profilePackageFromRow(ctx, row)
	if err != nil {
		return nil, err
	}

	fileRows, err := e.q.ListFilesByProfilePackage(ctx, row.ID)
	if err != nil {
		return nil, fmt.Errorf("store: list installed source files: %w", err)
	}

	files := make([]domain.FileEntry, 0, len(fileRows))
	for _, fileRow := range fileRows {
		files = append(files, fileFromRow(fileRow))
	}

	return &domain.InstalledSource{
		Profile:  profileFromRow(profileRow),
		FlakeRev: rev,
		Packages: []domain.ProfilePackage{profilePackage},
		Files:    files,
	}, nil
}

func (e *Exacker) profilePackageFromRow(ctx context.Context, row sqlc.ProfilePackage) (domain.ProfilePackage, error) {
	platformRow, err := e.q.GetPlatform(ctx, row.PlatformID)
	if err != nil {
		return domain.ProfilePackage{}, fmt.Errorf("store: get profile package platform: %w", err)
	}

	packageRow, err := e.q.GetPackage(ctx, row.PackageID)
	if err != nil {
		return domain.ProfilePackage{}, fmt.Errorf("store: get profile package package: %w", err)
	}

	pkg, err := e.packageFromRow(ctx, packageRow)
	if err != nil {
		return domain.ProfilePackage{}, err
	}

	return domain.ProfilePackage{
		System:     platformRow.Name,
		DrvPath:    row.DrvPath,
		StorePath:  row.StorePath,
		OutputName: row.OutputName,
		CreatedAt:  unixTime(row.CreatedAt),
		UpdatedAt:  unixTime(row.UpdatedAt),
		Package:    pkg,
	}, nil
}

func (e *Exacker) packageFromRow(ctx context.Context, row sqlc.Package) (domain.Package, error) {
	var license domain.License
	if row.LicenseID.Valid {
		licenseRow, err := e.q.GetLicense(ctx, row.LicenseID.Int64)
		if err != nil {
			return domain.Package{}, fmt.Errorf("store: get package license: %w", err)
		}
		license = domain.License{
			Open: licenseRow.Open,
			Name: licenseRow.Name,
		}
	}

	platformRows, err := e.q.ListPackagePlatforms(ctx, row.ID)
	if err != nil {
		return domain.Package{}, fmt.Errorf("store: list package platforms: %w", err)
	}

	platforms := make([]string, 0, len(platformRows))
	for _, platformRow := range platformRows {
		platforms = append(platforms, platformRow.Name)
	}

	return domain.Package{
		Attr:        row.Attr,
		Name:        row.Name,
		Description: row.Description,
		Version:     row.Version,
		License:     license,
		Outputs:     append([]string(nil), row.Outputs...),
		Platforms:   platforms,
		CreatedAt:   unixTime(row.CreatedAt),
		UpdatedAt:   unixTime(row.UpdatedAt),
	}, nil
}

func (e *Exacker) flakeRevFromRows(_ context.Context, revRow sqlc.FlakeRevision, flakeRow sqlc.Flake) (domain.FlakeRev, error) {
	var lockInfo flake.LockInfo
	if err := json.Unmarshal([]byte(revRow.LockJson), &lockInfo); err != nil {
		return domain.FlakeRev{}, fmt.Errorf("store: decode flake revision lock JSON: %w", err)
	}

	return domain.FlakeRev{
		Fingerprint: revRow.Fingerprint,
		LockInfo:    lockInfo,
		CreatedAt:   unixTime(revRow.CreatedAt),
		Flake:       flakeFromRow(flakeRow),
	}, nil
}

func (e *Exacker) getProfilePackageRow(ctx context.Context, source *domain.InstalledSource) (sqlc.ProfilePackage, error) {
	if source == nil {
		return sqlc.ProfilePackage{}, errors.New("store: get profile package row: nil source")
	}
	if len(source.Packages) != 1 {
		return sqlc.ProfilePackage{}, fmt.Errorf("store: get profile package row: got %d packages, want 1", len(source.Packages))
	}

	profileRow, err := e.getProfileRow(ctx, source.Profile)
	if err != nil {
		return sqlc.ProfilePackage{}, err
	}

	revRow, err := e.getFlakeRevRow(ctx, source.FlakeRev)
	if err != nil {
		return sqlc.ProfilePackage{}, err
	}

	profilePackage := source.Packages[0]
	packageRow, err := e.q.GetPackageByRevisionAttr(ctx, sqlc.GetPackageByRevisionAttrParams{
		FlakeRevisionID: revRow.ID,
		Attr:            profilePackage.Package.Attr,
	})
	if err != nil {
		return sqlc.ProfilePackage{}, fmt.Errorf("store: get profile package package %q: %w", profilePackage.Package.Attr, err)
	}

	platformRow, err := e.q.GetPlatformByName(ctx, profilePackage.System)
	if err != nil {
		return sqlc.ProfilePackage{}, fmt.Errorf("store: get profile package platform %q: %w", profilePackage.System, err)
	}

	row, err := e.q.GetProfilePackageByOutput(ctx, sqlc.GetProfilePackageByOutputParams{
		ProfileID:  profileRow.ID,
		PackageID:  packageRow.ID,
		PlatformID: platformRow.ID,
		OutputName: profilePackage.OutputName,
	})
	if err != nil {
		return sqlc.ProfilePackage{}, fmt.Errorf("store: get profile package row: %w", err)
	}

	return row, nil
}

func (e *Exacker) getProfileRow(ctx context.Context, profile domain.Profile) (sqlc.Profile, error) {
	if profile.Owner != "" {
		row, err := e.q.GetProfileByKindOwnerName(ctx, sqlc.GetProfileByKindOwnerNameParams{
			Kind:  string(profile.Kind),
			Owner: profile.Owner,
			Name:  profile.Name,
		})
		if err != nil {
			return sqlc.Profile{}, fmt.Errorf("store: get profile %s/%s/%s: %w", profile.Kind, profile.Owner, profile.Name, err)
		}
		return row, nil
	}

	row, err := e.q.GetProfileByKindName(ctx, sqlc.GetProfileByKindNameParams{
		Kind: string(profile.Kind),
		Name: profile.Name,
	})
	if err != nil {
		return sqlc.Profile{}, fmt.Errorf("store: get profile %s/%s: %w", profile.Kind, profile.Name, err)
	}
	return row, nil
}

func (e *Exacker) getFlakeRevRow(ctx context.Context, rev domain.FlakeRev) (sqlc.FlakeRevision, error) {
	flakeRow, err := e.q.GetFlakeByOwnerAlias(ctx, sqlc.GetFlakeByOwnerAliasParams{
		Owner: rev.Flake.Owner,
		Alias: rev.Flake.Alias,
	})
	if err != nil {
		return sqlc.FlakeRevision{}, fmt.Errorf("store: get flake %q/%q: %w", rev.Flake.Owner, rev.Flake.Alias, err)
	}

	revRow, err := e.q.GetFlakeRevisionByFingerprint(ctx, sqlc.GetFlakeRevisionByFingerprintParams{
		FlakeID:     flakeRow.ID,
		Fingerprint: rev.Fingerprint,
	})
	if err != nil {
		return sqlc.FlakeRevision{}, fmt.Errorf("store: get flake revision %q: %w", rev.Fingerprint, err)
	}

	return revRow, nil
}

func flakeFromRow(row sqlc.Flake) domain.Flake {
	return domain.Flake{
		Owner:     row.Owner,
		Alias:     row.Alias,
		Ref:       row.FlakeRef,
		CreatedAt: unixTime(row.CreatedAt),
		UpdatedAt: unixTime(row.UpdatedAt),
	}
}

func profileFromRow(row sqlc.Profile) domain.Profile {
	return domain.Profile{
		Kind:      domain.ProfileKind(row.Kind),
		Name:      row.Name,
		Owner:     row.Owner,
		Path:      row.Path,
		CreatedAt: row.CreatedAt,
	}
}

func fileFromRow(row sqlc.File) domain.FileEntry {
	return domain.FileEntry{
		Executable:       row.Executable,
		RelativePath:     row.RelativePath,
		MaterializedPath: row.MaterializedPath,
		CreatedAt:        row.CreatedAt,
	}
}

func unixTime(sec int64) time.Time {
	return time.Unix(sec, 0).UTC()
}

package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sund3RRR/gonix/flake"

	"github.com/sund3RRR/ion/pkg/domain"
	"github.com/sund3RRR/ion/pkg/ion/store/sqlc"
)

func TestStorePackageAndFlakeLookups(t *testing.T) {
	ctx := context.Background()
	store, fixture := newTestStore(t, ctx)
	defer store.Close() //nolint:errcheck

	rev, err := store.GetLatestFlakeRev(ctx, fixture.profile.Owner, fixture.flake.Alias)
	if err != nil {
		t.Fatalf("GetLatestFlakeRev() error = %v", err)
	}
	if rev.Fingerprint != fixture.rev.Fingerprint {
		t.Fatalf("GetLatestFlakeRev().Fingerprint = %q, want %q", rev.Fingerprint, fixture.rev.Fingerprint)
	}
	if rev.LockInfo.Version != 7 || rev.LockInfo.Root != "root" {
		t.Fatalf("GetLatestFlakeRev().LockInfo = %#v, want version 7 root", rev.LockInfo)
	}

	pkg, err := store.GetPackage(ctx, rev, fixture.pkg.Attr)
	if err != nil {
		t.Fatalf("GetPackage() error = %v", err)
	}
	if pkg.Attr != fixture.pkg.Attr || pkg.License.Name != "MIT" || !pkg.License.Open {
		t.Fatalf("GetPackage() = %#v, want attr %q and MIT open license", pkg, fixture.pkg.Attr)
	}
	if !reflect.DeepEqual(pkg.Outputs, []string{"out", "dev"}) {
		t.Fatalf("GetPackage().Outputs = %#v, want [out dev]", pkg.Outputs)
	}
	if !reflect.DeepEqual(pkg.Platforms, []string{fixture.platform.Name}) {
		t.Fatalf("GetPackage().Platforms = %#v, want %q", pkg.Platforms, fixture.platform.Name)
	}

	_, err = store.GetPackage(ctx, rev, "missing")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetPackage(missing) error = %v, want sql.ErrNoRows", err)
	}
}

func TestStoreProfilePackageLifecycleAndConflicts(t *testing.T) {
	ctx := context.Background()
	store, fixture := newTestStore(t, ctx)
	defer store.Close() //nolint:errcheck

	source := fixture.installedSource()
	if err := store.CreateProfilePackage(ctx, source); err != nil {
		t.Fatalf("CreateProfilePackage() error = %v", err)
	}

	files, err := store.ListProfilePackageFiles(ctx, source)
	if err != nil {
		t.Fatalf("ListProfilePackageFiles() error = %v", err)
	}
	wantFiles := []string{
		filepath.Join(fixture.profile.Path, "bin", "hello"),
		filepath.Join(fixture.profile.Path, "share", "applications", "hello.desktop"),
	}
	if !reflect.DeepEqual(files, wantFiles) {
		t.Fatalf("ListProfilePackageFiles() = %#v, want %#v", files, wantFiles)
	}

	profilePackage, err := store.GetProfilePackage(ctx, wantFiles[0])
	if err != nil {
		t.Fatalf("GetProfilePackage() error = %v", err)
	}
	if profilePackage.Package.Attr != fixture.pkg.Attr || profilePackage.System != fixture.platform.Name {
		t.Fatalf("GetProfilePackage() = %#v, want package %q on %q", profilePackage, fixture.pkg.Attr, fixture.platform.Name)
	}

	conflicts, err := store.GetConflictedPackages(ctx, []string{wantFiles[0], wantFiles[0], "/missing"})
	if err != nil {
		t.Fatalf("GetConflictedPackages() error = %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("GetConflictedPackages() len = %d, want 1", len(conflicts))
	}
	if conflicts[0].Profile.Name != fixture.profile.Name || conflicts[0].Packages[0].Package.Attr != fixture.pkg.Attr {
		t.Fatalf("GetConflictedPackages()[0] = %#v", conflicts[0])
	}
	if len(conflicts[0].Files) != len(wantFiles) {
		t.Fatalf("GetConflictedPackages()[0].Files len = %d, want %d", len(conflicts[0].Files), len(wantFiles))
	}

	source.Packages[0].StorePath = "/nix/store/updated-hello"
	if err := store.CreateProfilePackage(ctx, source); err != nil {
		t.Fatalf("CreateProfilePackage() update error = %v", err)
	}
	conflicts, err = store.GetConflictedPackages(ctx, []string{wantFiles[0]})
	if err != nil {
		t.Fatalf("GetConflictedPackages() after update error = %v", err)
	}
	if got := conflicts[0].Packages[0].StorePath; got != "/nix/store/updated-hello" {
		t.Fatalf("updated StorePath = %q, want updated path", got)
	}

	if err := store.DeleteProfilePackage(ctx, source); err != nil {
		t.Fatalf("DeleteProfilePackage() error = %v", err)
	}
	conflicts, err = store.GetConflictedPackages(ctx, []string{wantFiles[0]})
	if err != nil {
		t.Fatalf("GetConflictedPackages() after delete error = %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("GetConflictedPackages() after delete len = %d, want 0", len(conflicts))
	}
}

type storeFixture struct {
	flake    sqlc.Flake
	rev      sqlc.FlakeRevision
	profile  sqlc.Profile
	pkg      sqlc.Package
	platform sqlc.Platform
}

func newTestStore(t *testing.T, ctx context.Context) (*Store, storeFixture) {
	t.Helper()

	store, err := New(ctx, filepath.Join(t.TempDir(), "ion.db"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	q := sqlc.New(store.db)
	flakeRow, err := q.CreateFlake(ctx, sqlc.CreateFlakeParams{
		Owner:    "alice",
		Alias:    "nixpkgs",
		FlakeRef: "github:NixOS/nixpkgs",
	})
	if err != nil {
		t.Fatalf("CreateFlake() error = %v", err)
	}

	revRow, err := q.CreateFlakeRevision(ctx, sqlc.CreateFlakeRevisionParams{
		FlakeID:     flakeRow.ID,
		LockJson:    `{"version":7,"root":"root","nodes":{"root":{}}}`,
		Fingerprint: "rev-1",
	})
	if err != nil {
		t.Fatalf("CreateFlakeRevision() error = %v", err)
	}

	licenseRow, err := q.UpsertLicense(ctx, sqlc.UpsertLicenseParams{
		Open:        true,
		Name:        "MIT",
		Description: "MIT license",
	})
	if err != nil {
		t.Fatalf("UpsertLicense() error = %v", err)
	}

	platformRow, err := q.UpsertPlatform(ctx, "x86_64-linux")
	if err != nil {
		t.Fatalf("UpsertPlatform() error = %v", err)
	}

	pkgRow, err := q.CreatePackage(ctx, sqlc.CreatePackageParams{
		FlakeRevisionID: revRow.ID,
		LicenseID: sql.NullInt64{
			Int64: licenseRow.ID,
			Valid: true,
		},
		Attr:        "hello",
		Name:        "hello",
		Description: "hello package",
		Version:     "1.0.0",
		Outputs:     sqlc.StringList{"out", "dev"},
	})
	if err != nil {
		t.Fatalf("CreatePackage() error = %v", err)
	}

	if err := q.LinkPackagePlatform(ctx, sqlc.LinkPackagePlatformParams{
		PackageID:  pkgRow.ID,
		PlatformID: platformRow.ID,
	}); err != nil {
		t.Fatalf("LinkPackagePlatform() error = %v", err)
	}

	profileRow, err := q.CreateProfile(ctx, sqlc.CreateProfileParams{
		Kind:  string(domain.ProfileKindUser),
		Name:  "default",
		Owner: "alice",
		Path:  filepath.Join(t.TempDir(), "profile"),
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	return store, storeFixture{
		flake:    flakeRow,
		rev:      revRow,
		profile:  profileRow,
		pkg:      pkgRow,
		platform: platformRow,
	}
}

func (f storeFixture) installedSource() *domain.InstalledSource {
	profile := domain.Profile{
		Kind:  domain.ProfileKind(f.profile.Kind),
		Name:  f.profile.Name,
		Owner: f.profile.Owner,
		Path:  f.profile.Path,
	}

	flakeRev := domain.FlakeRev{
		Fingerprint: f.rev.Fingerprint,
		LockInfo: flake.LockInfo{
			Version: 7,
			Root:    "root",
			Nodes:   map[string]flake.LockNode{"root": {}},
		},
		Flake: domain.Flake{
			Owner: f.flake.Owner,
			Alias: f.flake.Alias,
			Ref:   f.flake.FlakeRef,
		},
	}

	return &domain.InstalledSource{
		Profile:  profile,
		FlakeRev: flakeRev,
		Packages: []domain.ProfilePackage{
			{
				System:     f.platform.Name,
				DrvPath:    "/nix/store/hello.drv",
				StorePath:  "/nix/store/hello",
				OutputName: "out",
				Package: domain.Package{
					Attr: f.pkg.Attr,
				},
			},
		},
		Files: []domain.FileEntry{
			{
				Executable:       true,
				RelativePath:     "bin/hello",
				MaterializedPath: filepath.Join(f.profile.Path, "bin", "hello"),
			},
			{
				RelativePath:     "share/applications/hello.desktop",
				MaterializedPath: filepath.Join(f.profile.Path, "share", "applications", "hello.desktop"),
			},
		},
	}
}

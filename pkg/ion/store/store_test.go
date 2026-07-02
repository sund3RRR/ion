package store_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/sund3RRR/ion/pkg/ion/store"
	"github.com/sund3RRR/ion/pkg/ion/store/sqlc"
)

func TestOpenCreatesDirectoryAndMigrates(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "nested", "ion.db")

	st, err := store.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer closeStore(t, st)

	var foreignKeys int
	if err := st.DB().QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("query foreign_keys pragma: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	for _, table := range []string{
		"flakes",
		"platforms",
		"licenses",
		"packages",
		"package_platforms",
		"profiles",
		"profile_packages",
		"files",
	} {
		var name string
		err := st.DB().QueryRowContext(
			ctx,
			"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
			table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("expected migrated table %q: %v", table, err)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	defer closeStore(t, st)

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() first error = %v", err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() second error = %v", err)
	}
}

func TestQueriesCoverCoreLifecycle(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	defer closeStore(t, st)

	queries := st.Queries()

	profile, err := queries.CreateProfile(ctx, sqlc.CreateProfileParams{
		Kind:  "user",
		Name:  "default",
		Owner: "sunder",
		Path:  filepath.Join(t.TempDir(), ".ion"),
	})
	if err != nil {
		t.Fatalf("CreateProfile() error = %v", err)
	}

	profiles, err := queries.ListProfiles(ctx)
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].ID != profile.ID {
		t.Fatalf("ListProfiles() = %#v, want created profile", profiles)
	}

	ownerProfiles, err := queries.ListProfilesByOwner(ctx, "sunder")
	if err != nil {
		t.Fatalf("ListProfilesByOwner() error = %v", err)
	}
	if len(ownerProfiles) != 1 || ownerProfiles[0].ID != profile.ID {
		t.Fatalf("ListProfilesByOwner() = %#v, want created profile", ownerProfiles)
	}

	flake, err := queries.CreateFlake(ctx, sqlc.CreateFlakeParams{
		Alias:       "unstable",
		FlakeRef:    "github:NixOS/nixpkgs/nixos-unstable",
		LockJson:    `{"version":7}`,
		Fingerprint: "fingerprint",
	})
	if err != nil {
		t.Fatalf("CreateFlake() error = %v", err)
	}

	gotFlake, err := queries.GetFlakeByAliasFingerprint(ctx, sqlc.GetFlakeByAliasFingerprintParams{
		Alias:       flake.Alias,
		Fingerprint: flake.Fingerprint,
	})
	if err != nil {
		t.Fatalf("GetFlakeByAliasFingerprint() error = %v", err)
	}
	if gotFlake.ID != flake.ID {
		t.Fatalf("flake id = %d, want %d", gotFlake.ID, flake.ID)
	}

	platform, err := queries.UpsertPlatform(ctx, "aarch64-darwin")
	if err != nil {
		t.Fatalf("UpsertPlatform() error = %v", err)
	}

	license, err := queries.UpsertLicense(ctx, sqlc.UpsertLicenseParams{
		Open:        1,
		Name:        "MIT",
		Description: "MIT License",
	})
	if err != nil {
		t.Fatalf("UpsertLicense() error = %v", err)
	}

	pkg, err := queries.CreatePackage(ctx, sqlc.CreatePackageParams{
		FlakeID:     flake.ID,
		LicenseID:   sql.NullInt64{Int64: license.ID, Valid: true},
		Attr:        "hello",
		Name:        "hello",
		Description: "GNU Hello",
		Version:     "2.12.2",
		Outputs:     sqlc.StringList{"out"},
	})
	if err != nil {
		t.Fatalf("CreatePackage() error = %v", err)
	}
	if len(pkg.Outputs) != 1 || pkg.Outputs[0] != "out" {
		t.Fatalf("package outputs = %#v, want [out]", pkg.Outputs)
	}

	if err := queries.LinkPackagePlatform(ctx, sqlc.LinkPackagePlatformParams{
		PackageID:  pkg.ID,
		PlatformID: platform.ID,
	}); err != nil {
		t.Fatalf("LinkPackagePlatform() error = %v", err)
	}

	packagePlatforms, err := queries.ListPackagePlatforms(ctx, pkg.ID)
	if err != nil {
		t.Fatalf("ListPackagePlatforms() error = %v", err)
	}
	if len(packagePlatforms) != 1 || packagePlatforms[0].ID != platform.ID {
		t.Fatalf("ListPackagePlatforms() = %#v, want linked platform", packagePlatforms)
	}

	platformPackages, err := queries.ListPackagesByPlatform(ctx, platform.ID)
	if err != nil {
		t.Fatalf("ListPackagesByPlatform() error = %v", err)
	}
	if len(platformPackages) != 1 || platformPackages[0].ID != pkg.ID {
		t.Fatalf("ListPackagesByPlatform() = %#v, want linked package", platformPackages)
	}

	profilePackage, err := queries.CreateProfilePackage(ctx, sqlc.CreateProfilePackageParams{
		ProfileID:  profile.ID,
		PackageID:  pkg.ID,
		PlatformID: platform.ID,
		OutputName: "out",
		DrvPath:    "/nix/store/hello.drv",
		StorePath:  "/nix/store/hello",
	})
	if err != nil {
		t.Fatalf("CreateProfilePackage() error = %v", err)
	}

	profilePackages, err := queries.ListProfilePackages(ctx, profile.ID)
	if err != nil {
		t.Fatalf("ListProfilePackages() error = %v", err)
	}
	if len(profilePackages) != 1 || profilePackages[0].ID != profilePackage.ID {
		t.Fatalf("ListProfilePackages() = %#v, want installed profile package", profilePackages)
	}

	materializedPath := filepath.Join(profile.Path, "bin", "hello")
	file, err := queries.CreateFile(ctx, sqlc.CreateFileParams{
		ProfilePackageID: profilePackage.ID,
		Executable:       1,
		RelativePath:     "bin/hello",
		MaterializedPath: materializedPath,
		StorePath:        "/nix/store/hello/bin/hello",
	})
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}

	fileByPath, err := queries.GetFileByMaterializedPath(ctx, materializedPath)
	if err != nil {
		t.Fatalf("GetFileByMaterializedPath() error = %v", err)
	}
	if fileByPath.ID != file.ID {
		t.Fatalf("file id = %d, want %d", fileByPath.ID, file.ID)
	}

	filesByPackage, err := queries.ListFilesByProfilePackage(ctx, profilePackage.ID)
	if err != nil {
		t.Fatalf("ListFilesByProfilePackage() error = %v", err)
	}
	if len(filesByPackage) != 1 || filesByPackage[0].ID != file.ID {
		t.Fatalf("ListFilesByProfilePackage() = %#v, want materialized file", filesByPackage)
	}

	filesByProfile, err := queries.ListFilesByProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("ListFilesByProfile() error = %v", err)
	}
	if len(filesByProfile) != 1 || filesByProfile[0].ID != file.ID {
		t.Fatalf("ListFilesByProfile() = %#v, want materialized file", filesByProfile)
	}

	if err := queries.DeleteFilesByProfilePackage(ctx, profilePackage.ID); err != nil {
		t.Fatalf("DeleteFilesByProfilePackage() error = %v", err)
	}

	filesByPackage, err = queries.ListFilesByProfilePackage(ctx, profilePackage.ID)
	if err != nil {
		t.Fatalf("ListFilesByProfilePackage() after delete error = %v", err)
	}
	if len(filesByPackage) != 0 {
		t.Fatalf("ListFilesByProfilePackage() after delete = %#v, want empty", filesByPackage)
	}
}

func TestWithTxCommitsAndRollsBack(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	defer closeStore(t, st)

	if err := st.WithTx(ctx, func(queries *sqlc.Queries) error {
		_, err := queries.CreateProfile(ctx, sqlc.CreateProfileParams{
			Kind:  "user",
			Name:  "committed",
			Owner: "sunder",
			Path:  "/tmp/committed",
		})
		return err
	}); err != nil {
		t.Fatalf("WithTx() commit error = %v", err)
	}

	if _, err := st.Queries().GetProfileByKindName(ctx, sqlc.GetProfileByKindNameParams{
		Kind: "user",
		Name: "committed",
	}); err != nil {
		t.Fatalf("GetProfileByKindName() after commit error = %v", err)
	}

	errRollback := errors.New("rollback")
	if err := st.WithTx(ctx, func(queries *sqlc.Queries) error {
		_, err := queries.CreateProfile(ctx, sqlc.CreateProfileParams{
			Kind:  "user",
			Name:  "rolled-back",
			Owner: "sunder",
			Path:  "/tmp/rolled-back",
		})
		if err != nil {
			return err
		}
		return errRollback
	}); !errors.Is(err, errRollback) {
		t.Fatalf("WithTx() rollback error = %v, want %v", err, errRollback)
	}

	_, err := st.Queries().GetProfileByKindName(ctx, sqlc.GetProfileByKindNameParams{
		Kind: "user",
		Name: "rolled-back",
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("GetProfileByKindName() after rollback error = %v, want sql.ErrNoRows", err)
	}
}

func openTestStore(t *testing.T) *store.Store {
	t.Helper()

	st, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "ion.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	return st
}

func closeStore(t *testing.T, st *store.Store) {
	t.Helper()

	if err := st.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

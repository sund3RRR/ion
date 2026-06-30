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
		"profiles",
		"sources",
		"source_revisions",
		"installed_packages",
		"transactions",
		"transaction_items",
		"gc_roots",
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
		Kind:           "user",
		Name:           "default",
		Path:           filepath.Join(t.TempDir(), ".ion"),
		ActiveRevision: "",
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

	source, err := queries.CreateSource(ctx, sqlc.CreateSourceParams{
		Alias:    "unstable",
		FlakeRef: "github:NixOS/nixpkgs/nixos-unstable",
		Enabled:  1,
		Priority: 10,
	})
	if err != nil {
		t.Fatalf("CreateSource() error = %v", err)
	}

	revision, err := queries.CreateSourceRevision(ctx, sqlc.CreateSourceRevisionParams{
		SourceID:     source.ID,
		LockJson:     `{"version":7}`,
		Fingerprint:  "fingerprint",
		MetadataJson: `{}`,
	})
	if err != nil {
		t.Fatalf("CreateSourceRevision() error = %v", err)
	}

	source, err = queries.SetSourceCurrentRevision(ctx, sqlc.SetSourceCurrentRevisionParams{
		CurrentRevisionID: sql.NullInt64{Int64: revision.ID, Valid: true},
		ID:                source.ID,
	})
	if err != nil {
		t.Fatalf("SetSourceCurrentRevision() error = %v", err)
	}
	if !source.CurrentRevisionID.Valid || source.CurrentRevisionID.Int64 != revision.ID {
		t.Fatalf("CurrentRevisionID = %#v, want %d", source.CurrentRevisionID, revision.ID)
	}

	currentRevision, err := queries.GetCurrentSourceRevisionByAlias(ctx, source.Alias)
	if err != nil {
		t.Fatalf("GetCurrentSourceRevisionByAlias() error = %v", err)
	}
	if currentRevision.ID != revision.ID {
		t.Fatalf("current revision id = %d, want %d", currentRevision.ID, revision.ID)
	}

	installed, err := queries.CreateInstalledPackage(ctx, sqlc.CreateInstalledPackageParams{
		ProfileID:        profile.ID,
		SourceID:         source.ID,
		SourceRevisionID: revision.ID,
		Attr:             "hello",
		Name:             "hello",
		Version:          "2.12.2",
		OutputsJson:      `{"out":"/nix/store/hello"}`,
		DrvPath:          "/nix/store/hello.drv",
		StorePathsJson:   `{"out":"/nix/store/hello"}`,
		Reason:           "user",
		Priority:         0,
		UpgradePolicy:    "follow-source",
		State:            "installed",
	})
	if err != nil {
		t.Fatalf("CreateInstalledPackage() error = %v", err)
	}

	packages, err := queries.ListInstalledPackagesByProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("ListInstalledPackagesByProfile() error = %v", err)
	}
	if len(packages) != 1 || packages[0].ID != installed.ID {
		t.Fatalf("ListInstalledPackagesByProfile() = %#v, want installed package", packages)
	}

	updatedPackage, err := queries.UpdateInstalledPackageState(ctx, sqlc.UpdateInstalledPackageStateParams{
		State: "removed",
		ID:    installed.ID,
	})
	if err != nil {
		t.Fatalf("UpdateInstalledPackageState() error = %v", err)
	}
	if updatedPackage.State != "removed" {
		t.Fatalf("updated package state = %q, want removed", updatedPackage.State)
	}

	transaction, err := queries.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		Kind:         "install",
		ProfileID:    sql.NullInt64{Int64: profile.ID, Valid: true},
		State:        "planned",
		MetadataJson: `{}`,
	})
	if err != nil {
		t.Fatalf("CreateTransaction() error = %v", err)
	}

	unfinished, err := queries.ListUnfinishedTransactions(ctx)
	if err != nil {
		t.Fatalf("ListUnfinishedTransactions() error = %v", err)
	}
	if len(unfinished) != 1 || unfinished[0].ID != transaction.ID {
		t.Fatalf("ListUnfinishedTransactions() = %#v, want planned transaction", unfinished)
	}

	item, err := queries.AddTransactionItem(ctx, sqlc.AddTransactionItemParams{
		TransactionID: transaction.ID,
		Action:        "install",
		PackageID:     sql.NullInt64{Int64: installed.ID, Valid: true},
		OldJson:       `{}`,
		NewJson:       `{"attr":"hello"}`,
		State:         "planned",
		Error:         "",
	})
	if err != nil {
		t.Fatalf("AddTransactionItem() error = %v", err)
	}

	item, err = queries.UpdateTransactionItemState(ctx, sqlc.UpdateTransactionItemStateParams{
		State: "succeeded",
		Error: "",
		ID:    item.ID,
	})
	if err != nil {
		t.Fatalf("UpdateTransactionItemState() error = %v", err)
	}
	if item.State != "succeeded" {
		t.Fatalf("transaction item state = %q, want succeeded", item.State)
	}

	transaction, err = queries.UpdateTransactionState(ctx, sqlc.UpdateTransactionStateParams{
		State:      "succeeded",
		FinishedAt: sql.NullInt64{Int64: transaction.StartedAt + 1, Valid: true},
		Error:      "",
		ID:         transaction.ID,
	})
	if err != nil {
		t.Fatalf("UpdateTransactionState() error = %v", err)
	}
	if transaction.State != "succeeded" {
		t.Fatalf("transaction state = %q, want succeeded", transaction.State)
	}

	unfinished, err = queries.ListUnfinishedTransactions(ctx)
	if err != nil {
		t.Fatalf("ListUnfinishedTransactions() after finish error = %v", err)
	}
	if len(unfinished) != 0 {
		t.Fatalf("ListUnfinishedTransactions() after finish = %#v, want empty", unfinished)
	}

	root, err := queries.UpsertGCRoot(ctx, sqlc.UpsertGCRootParams{
		ProfileID:          profile.ID,
		InstalledPackageID: installed.ID,
		OutputName:         "out",
		RootPath:           filepath.Join(t.TempDir(), "hello-root"),
		StorePath:          "/nix/store/hello",
		State:              "active",
	})
	if err != nil {
		t.Fatalf("UpsertGCRoot() error = %v", err)
	}

	root, err = queries.UpdateGCRootState(ctx, sqlc.UpdateGCRootStateParams{
		State: "stale",
		ID:    root.ID,
	})
	if err != nil {
		t.Fatalf("UpdateGCRootState() error = %v", err)
	}
	if root.State != "stale" {
		t.Fatalf("gc root state = %q, want stale", root.State)
	}

	roots, err := queries.ListGCRootsByProfile(ctx, profile.ID)
	if err != nil {
		t.Fatalf("ListGCRootsByProfile() error = %v", err)
	}
	if len(roots) != 1 || roots[0].ID != root.ID {
		t.Fatalf("ListGCRootsByProfile() = %#v, want gc root", roots)
	}
}

func TestWithTxCommitsAndRollsBack(t *testing.T) {
	ctx := context.Background()
	st := openTestStore(t)
	defer closeStore(t, st)

	if err := st.WithTx(ctx, func(queries *sqlc.Queries) error {
		_, err := queries.CreateProfile(ctx, sqlc.CreateProfileParams{
			Kind:           "user",
			Name:           "committed",
			Path:           "/tmp/committed",
			ActiveRevision: "",
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
			Kind:           "user",
			Name:           "rolled-back",
			Path:           "/tmp/rolled-back",
			ActiveRevision: "",
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

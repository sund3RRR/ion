-- +goose Up
CREATE TABLE profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT NOT NULL,
    active_revision TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (kind, name)
);

CREATE TABLE sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alias TEXT NOT NULL UNIQUE,
    flake_ref TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 0,
    current_revision_id INTEGER,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (current_revision_id) REFERENCES source_revisions(id)
);

CREATE TABLE source_revisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id INTEGER NOT NULL,
    lock_json TEXT NOT NULL,
    fingerprint TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

CREATE INDEX source_revisions_source_id ON source_revisions(source_id);
CREATE INDEX source_revisions_fingerprint ON source_revisions(fingerprint);

CREATE TABLE installed_packages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL,
    source_id INTEGER NOT NULL,
    source_revision_id INTEGER NOT NULL,
    attr TEXT NOT NULL,
    name TEXT NOT NULL,
    version TEXT NOT NULL DEFAULT '',
    outputs_json TEXT NOT NULL DEFAULT '{}',
    drv_path TEXT NOT NULL DEFAULT '',
    store_paths_json TEXT NOT NULL DEFAULT '{}',
    reason TEXT NOT NULL DEFAULT 'user',
    priority INTEGER NOT NULL DEFAULT 0,
    upgrade_policy TEXT NOT NULL DEFAULT 'follow-source',
    state TEXT NOT NULL DEFAULT 'installed',
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES sources(id),
    FOREIGN KEY (source_revision_id) REFERENCES source_revisions(id),
    UNIQUE (profile_id, source_id, attr)
);

CREATE INDEX installed_packages_profile_id ON installed_packages(profile_id);
CREATE INDEX installed_packages_source_revision_id ON installed_packages(source_revision_id);
CREATE INDEX installed_packages_state ON installed_packages(state);

CREATE TABLE transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    profile_id INTEGER,
    state TEXT NOT NULL DEFAULT 'planned',
    started_at INTEGER NOT NULL DEFAULT (unixepoch()),
    finished_at INTEGER,
    error TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE SET NULL
);

CREATE INDEX transactions_profile_id ON transactions(profile_id);
CREATE INDEX transactions_state ON transactions(state);

CREATE TABLE transaction_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    transaction_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    package_id INTEGER,
    old_json TEXT NOT NULL DEFAULT '{}',
    new_json TEXT NOT NULL DEFAULT '{}',
    state TEXT NOT NULL DEFAULT 'planned',
    error TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    FOREIGN KEY (package_id) REFERENCES installed_packages(id) ON DELETE SET NULL
);

CREATE INDEX transaction_items_transaction_id ON transaction_items(transaction_id);
CREATE INDEX transaction_items_package_id ON transaction_items(package_id);

CREATE TABLE gc_roots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL,
    installed_package_id INTEGER NOT NULL,
    output_name TEXT NOT NULL,
    root_path TEXT NOT NULL UNIQUE,
    store_path TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'active',
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (installed_package_id) REFERENCES installed_packages(id) ON DELETE CASCADE,
    UNIQUE (profile_id, installed_package_id, output_name)
);

CREATE INDEX gc_roots_profile_id ON gc_roots(profile_id);
CREATE INDEX gc_roots_installed_package_id ON gc_roots(installed_package_id);
CREATE INDEX gc_roots_state ON gc_roots(state);

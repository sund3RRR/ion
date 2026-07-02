-- +goose Up
CREATE TABLE flakes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alias TEXT NOT NULL,
    flake_ref TEXT NOT NULL,
    lock_json TEXT NOT NULL,
    fingerprint TEXT NOT NULL CHECK (fingerprint <> ''),
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (alias, fingerprint)
);

CREATE TABLE platforms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE licenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    open INTEGER NOT NULL DEFAULT 0,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE packages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    flake_id INTEGER NOT NULL,
    license_id INTEGER,
    attr TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    version TEXT NOT NULL,
    outputs TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (flake_id) REFERENCES flakes(id) ON DELETE CASCADE,
    FOREIGN KEY (license_id) REFERENCES licenses(id) ON DELETE SET NULL,
    UNIQUE (attr, flake_id)
);

CREATE TABLE package_platforms (
    package_id INTEGER NOT NULL,
    platform_id INTEGER NOT NULL,
    PRIMARY KEY (package_id, platform_id),
    FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE CASCADE,
    FOREIGN KEY (platform_id) REFERENCES platforms(id) ON DELETE CASCADE
);

CREATE TABLE profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL CHECK (kind IN ('system', 'user')),
    name TEXT NOT NULL,
    owner TEXT NOT NULL,
    path TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (name, kind)
);

CREATE TABLE profile_packages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL,
    package_id INTEGER NOT NULL,
    platform_id INTEGER NOT NULL,
    output_name TEXT NOT NULL,
    drv_path TEXT NOT NULL,
    store_path TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (package_id, profile_id, output_name, platform_id),
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE,
    FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE CASCADE,
    FOREIGN KEY (platform_id) REFERENCES platforms(id) ON DELETE CASCADE
);

CREATE TABLE files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_package_id INTEGER NOT NULL,
    executable INTEGER NOT NULL DEFAULT 0,
    relative_path TEXT NOT NULL,
    materialized_path TEXT NOT NULL UNIQUE,
    store_path TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (profile_package_id) REFERENCES profile_packages(id) ON DELETE CASCADE
);

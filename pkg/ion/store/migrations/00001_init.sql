-- +goose Up
CREATE TABLE flakes (
    id INTEGER PRIMARY KEY,
    alias TEXT NOT NULL,
    flake_ref TEXT NOT NULL,
    lock_json TEXT NOT NULL CHECK (json_valid(lock_json)),
    fingerprint TEXT NOT NULL CHECK (fingerprint <> ''),
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (alias, fingerprint)
);

CREATE TABLE platforms (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE licenses (
    id INTEGER PRIMARY KEY,
    open INTEGER NOT NULL DEFAULT 0 CHECK (open IN (0, 1)),
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE packages (
    id INTEGER PRIMARY KEY,
    flake_id INTEGER NOT NULL,
    license_id INTEGER,
    attr TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version TEXT NOT NULL,
    outputs TEXT NOT NULL CHECK (json_valid(outputs) AND json_type(outputs) = 'array'),
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (flake_id) REFERENCES flakes(id) ON DELETE CASCADE,
    FOREIGN KEY (license_id) REFERENCES licenses(id) ON DELETE SET NULL,
    UNIQUE (attr, flake_id)
);

CREATE INDEX idx_packages_flake_id ON packages(flake_id);
CREATE INDEX idx_packages_license_id ON packages(license_id);

CREATE TABLE package_platforms (
    package_id INTEGER NOT NULL,
    platform_id INTEGER NOT NULL,
    PRIMARY KEY (package_id, platform_id),
    FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE CASCADE,
    FOREIGN KEY (platform_id) REFERENCES platforms(id) ON DELETE CASCADE
);

CREATE INDEX idx_package_platforms_platform_id ON package_platforms(platform_id);

CREATE TABLE profiles (
    id INTEGER PRIMARY KEY,
    kind TEXT NOT NULL CHECK (kind IN ('system', 'user')),
    name TEXT NOT NULL,
    owner TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    UNIQUE (owner, name, kind)
);

CREATE TABLE profile_packages (
    id INTEGER PRIMARY KEY,
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
    FOREIGN KEY (platform_id) REFERENCES platforms(id) ON DELETE CASCADE,
    FOREIGN KEY (package_id, platform_id) REFERENCES package_platforms(package_id, platform_id)
);

CREATE INDEX idx_profile_packages_profile_id ON profile_packages(profile_id);
CREATE INDEX idx_profile_packages_package_id ON profile_packages(package_id);
CREATE INDEX idx_profile_packages_platform_id ON profile_packages(platform_id);

CREATE TABLE files (
    id INTEGER PRIMARY KEY,
    profile_package_id INTEGER NOT NULL,
    executable INTEGER NOT NULL DEFAULT 0 CHECK (executable IN (0, 1)),
    relative_path TEXT NOT NULL,
    materialized_path TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    FOREIGN KEY (profile_package_id) REFERENCES profile_packages(id) ON DELETE CASCADE
);

CREATE INDEX idx_files_profile_package_id ON files(profile_package_id);

-- +goose StatementBegin
CREATE TRIGGER packages_updated_at
AFTER UPDATE OF license_id, name, description, version, outputs ON packages
FOR EACH ROW
BEGIN
    UPDATE packages
    SET updated_at = unixepoch()
    WHERE id = OLD.id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER profile_packages_updated_at
AFTER UPDATE OF profile_id, package_id, platform_id, output_name, drv_path, store_path ON profile_packages
FOR EACH ROW
BEGIN
    UPDATE profile_packages
    SET updated_at = unixepoch()
    WHERE id = OLD.id;
END;
-- +goose StatementEnd

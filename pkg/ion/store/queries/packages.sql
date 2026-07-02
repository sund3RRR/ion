-- name: CreatePackage :one
INSERT INTO packages (
    flake_id,
    license_id,
    attr,
    name,
    description,
    version,
    outputs
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetPackage :one
SELECT * FROM packages
WHERE id = ?;

-- name: GetPackageByFlakeAttr :one
SELECT * FROM packages
WHERE flake_id = ? AND attr = ?;

-- name: ListPackagesByFlake :many
SELECT * FROM packages
WHERE flake_id = ?
ORDER BY attr;

-- name: SearchPackagesByName :many
SELECT * FROM packages
WHERE name LIKE ? OR attr LIKE ?
ORDER BY name, attr
LIMIT ?;

-- name: UpdatePackageMetadata :one
UPDATE packages
SET license_id = ?,
    name = ?,
    description = ?,
    version = ?,
    outputs = ?,
    updated_at = unixepoch()
WHERE id = ?
RETURNING *;

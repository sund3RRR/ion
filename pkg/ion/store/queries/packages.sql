-- name: CreatePackage :one
INSERT INTO packages (
    flake_revision_id,
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

-- name: GetPackageByRevisionAttr :one
SELECT * FROM packages
WHERE flake_revision_id = ? AND attr = ?;

-- name: ListPackagesByRevision :many
SELECT * FROM packages
WHERE flake_revision_id = ?
ORDER BY attr;

-- name: GetLatestPackageByFlakeAlias :one
SELECT packages.* FROM packages
JOIN flake_revisions ON flake_revisions.id = packages.flake_revision_id
JOIN flakes ON flakes.id = flake_revisions.flake_id
WHERE flakes.owner = ?
  AND flakes.alias = ?
  AND packages.attr = ?
ORDER BY flake_revisions.created_at DESC, flake_revisions.id DESC
LIMIT 1;

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

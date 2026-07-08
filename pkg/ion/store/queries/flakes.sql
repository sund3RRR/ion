-- name: CreateFlake :one
INSERT INTO flakes (
    owner,
    alias,
    flake_ref
) VALUES (
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetFlake :one
SELECT * FROM flakes
WHERE id = ?;

-- name: GetFlakeByOwnerAlias :one
SELECT * FROM flakes
WHERE owner = ? AND alias = ?;

-- name: ListFlakes :many
SELECT * FROM flakes
ORDER BY owner, alias;

-- name: ListFlakesByOwner :many
SELECT * FROM flakes
WHERE owner = ?
ORDER BY alias;

-- name: CreateFlakeRevision :one
INSERT INTO flake_revisions (
    flake_id,
    lock_json,
    fingerprint
) VALUES (
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetFlakeRevision :one
SELECT * FROM flake_revisions
WHERE id = ?;

-- name: GetFlakeRevisionByFingerprint :one
SELECT * FROM flake_revisions
WHERE flake_id = ? AND fingerprint = ?;

-- name: GetLatestFlakeRevision :one
SELECT * FROM flake_revisions
WHERE flake_id = ?
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListFlakeRevisions :many
SELECT * FROM flake_revisions
WHERE flake_id = ?
ORDER BY created_at DESC, id DESC;

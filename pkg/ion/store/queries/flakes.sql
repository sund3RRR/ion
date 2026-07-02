-- name: CreateFlake :one
INSERT INTO flakes (
    alias,
    flake_ref,
    lock_json,
    fingerprint
) VALUES (
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetFlake :one
SELECT * FROM flakes
WHERE id = ?;

-- name: GetFlakeByAliasFingerprint :one
SELECT * FROM flakes
WHERE alias = ? AND fingerprint = ?;

-- name: ListFlakes :many
SELECT * FROM flakes
ORDER BY alias, id DESC;

-- name: ListFlakesByAlias :many
SELECT * FROM flakes
WHERE alias = ?
ORDER BY id DESC;

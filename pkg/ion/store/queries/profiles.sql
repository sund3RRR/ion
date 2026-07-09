-- name: CreateProfile :one
INSERT INTO profiles (
    kind,
    name,
    owner,
    path
) VALUES (
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetProfile :one
SELECT * FROM profiles
WHERE id = ?;

-- name: GetProfileByKindName :one
SELECT * FROM profiles
WHERE kind = ? AND name = ?;

-- name: GetProfileByKindOwnerName :one
SELECT * FROM profiles
WHERE kind = ? AND owner = ? AND name = ?;

-- name: ListProfiles :many
SELECT * FROM profiles
ORDER BY kind, owner, name;

-- name: ListProfilesByOwner :many
SELECT * FROM profiles
WHERE owner = ?
ORDER BY kind, name;

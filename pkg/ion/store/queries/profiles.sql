-- name: CreateProfile :one
INSERT INTO profiles (
    kind,
    name,
    path,
    active_revision
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

-- name: ListProfiles :many
SELECT * FROM profiles
ORDER BY kind, name;

-- name: UpdateProfileActiveRevision :one
UPDATE profiles
SET active_revision = ?
WHERE id = ?
RETURNING *;

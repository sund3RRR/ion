-- name: UpsertPlatform :one
INSERT INTO platforms (
    name
) VALUES (
    ?
)
ON CONFLICT(name) DO UPDATE SET
    name = excluded.name
RETURNING *;

-- name: GetPlatform :one
SELECT * FROM platforms
WHERE id = ?;

-- name: GetPlatformByName :one
SELECT * FROM platforms
WHERE name = ?;

-- name: ListPlatforms :many
SELECT * FROM platforms
ORDER BY name;

-- name: UpsertLicense :one
INSERT INTO licenses (
    open,
    name,
    description
) VALUES (
    ?,
    ?,
    ?
)
ON CONFLICT(name) DO UPDATE SET
    open = excluded.open,
    description = excluded.description
RETURNING *;

-- name: GetLicense :one
SELECT * FROM licenses
WHERE id = ?;

-- name: GetLicenseByName :one
SELECT * FROM licenses
WHERE name = ?;

-- name: ListLicenses :many
SELECT * FROM licenses
ORDER BY name;

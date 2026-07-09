-- name: CreateProfilePackage :one
INSERT INTO profile_packages (
    profile_id,
    package_id,
    platform_id,
    output_name,
    drv_path,
    store_path
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
ON CONFLICT(package_id, profile_id, output_name, platform_id) DO UPDATE SET
    drv_path = excluded.drv_path,
    store_path = excluded.store_path,
    updated_at = unixepoch()
RETURNING *;

-- name: GetProfilePackage :one
SELECT * FROM profile_packages
WHERE id = ?;

-- name: GetProfilePackageByOutput :one
SELECT * FROM profile_packages
WHERE profile_id = ?
  AND package_id = ?
  AND platform_id = ?
  AND output_name = ?;

-- name: ListProfilePackages :many
SELECT * FROM profile_packages
WHERE profile_id = ?
ORDER BY updated_at DESC, id DESC;

-- name: TouchProfilePackage :one
UPDATE profile_packages
SET updated_at = unixepoch()
WHERE id = ?
RETURNING *;

-- name: DeleteProfilePackage :exec
DELETE FROM profile_packages
WHERE id = ?;

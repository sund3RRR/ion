-- name: CreateFile :one
INSERT INTO files (
    profile_package_id,
    executable,
    relative_path,
    materialized_path
) VALUES (
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetFile :one
SELECT * FROM files
WHERE id = ?;

-- name: GetFileByMaterializedPath :one
SELECT * FROM files
WHERE materialized_path = ?;

-- name: ListFilesByProfilePackage :many
SELECT * FROM files
WHERE profile_package_id = ?
ORDER BY relative_path;

-- name: ListFilesByProfile :many
SELECT files.* FROM files
JOIN profile_packages ON profile_packages.id = files.profile_package_id
WHERE profile_packages.profile_id = ?
ORDER BY files.materialized_path;

-- name: DeleteFilesByProfilePackage :exec
DELETE FROM files
WHERE profile_package_id = ?;

-- name: DeleteFile :exec
DELETE FROM files
WHERE id = ?;

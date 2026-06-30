-- name: UpsertGCRoot :one
INSERT INTO gc_roots (
    profile_id,
    installed_package_id,
    output_name,
    root_path,
    store_path,
    state
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
)
ON CONFLICT(profile_id, installed_package_id, output_name) DO UPDATE SET
    root_path = excluded.root_path,
    store_path = excluded.store_path,
    state = excluded.state,
    updated_at = unixepoch()
RETURNING *;

-- name: ListGCRootsByProfile :many
SELECT * FROM gc_roots
WHERE profile_id = ?
ORDER BY root_path;

-- name: UpdateGCRootState :one
UPDATE gc_roots
SET state = ?, updated_at = unixepoch()
WHERE id = ?
RETURNING *;

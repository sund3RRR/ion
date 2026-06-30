-- name: CreateInstalledPackage :one
INSERT INTO installed_packages (
    profile_id,
    source_id,
    source_revision_id,
    attr,
    name,
    version,
    outputs_json,
    drv_path,
    store_paths_json,
    reason,
    priority,
    upgrade_policy,
    state
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetInstalledPackage :one
SELECT * FROM installed_packages
WHERE id = ?;

-- name: ListInstalledPackagesByProfile :many
SELECT * FROM installed_packages
WHERE profile_id = ?
ORDER BY priority DESC, name, attr;

-- name: UpdateInstalledPackageState :one
UPDATE installed_packages
SET state = ?, updated_at = unixepoch()
WHERE id = ?
RETURNING *;

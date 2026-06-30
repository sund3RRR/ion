-- name: CreateSource :one
INSERT INTO sources (
    alias,
    flake_ref,
    enabled,
    priority
) VALUES (
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetSource :one
SELECT * FROM sources
WHERE id = ?;

-- name: GetSourceByAlias :one
SELECT * FROM sources
WHERE alias = ?;

-- name: ListSources :many
SELECT * FROM sources
ORDER BY priority DESC, alias;

-- name: CreateSourceRevision :one
INSERT INTO source_revisions (
    source_id,
    lock_json,
    fingerprint,
    metadata_json
) VALUES (
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetSourceRevision :one
SELECT * FROM source_revisions
WHERE id = ?;

-- name: ListSourceRevisions :many
SELECT * FROM source_revisions
WHERE source_id = ?
ORDER BY id DESC;

-- name: SetSourceCurrentRevision :one
UPDATE sources
SET current_revision_id = ?
WHERE id = ?
RETURNING *;

-- name: GetCurrentSourceRevisionByAlias :one
SELECT source_revisions.* FROM source_revisions
JOIN sources ON sources.current_revision_id = source_revisions.id
WHERE sources.alias = ?;

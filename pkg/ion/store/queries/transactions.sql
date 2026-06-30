-- name: CreateTransaction :one
INSERT INTO transactions (
    kind,
    profile_id,
    state,
    metadata_json
) VALUES (
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions
WHERE id = ?;

-- name: UpdateTransactionState :one
UPDATE transactions
SET state = ?,
    finished_at = ?,
    error = ?
WHERE id = ?
RETURNING *;

-- name: ListUnfinishedTransactions :many
SELECT * FROM transactions
WHERE state NOT IN ('succeeded', 'failed', 'interrupted')
ORDER BY id;

-- name: AddTransactionItem :one
INSERT INTO transaction_items (
    transaction_id,
    action,
    package_id,
    old_json,
    new_json,
    state,
    error
) VALUES (
    ?,
    ?,
    ?,
    ?,
    ?,
    ?,
    ?
) RETURNING *;

-- name: UpdateTransactionItemState :one
UPDATE transaction_items
SET state = ?,
    error = ?
WHERE id = ?
RETURNING *;

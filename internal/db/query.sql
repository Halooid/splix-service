-- name: CreateUser :one
INSERT INTO users (id, email, name, country_code, default_currency_code)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: AddConnection :exec
INSERT INTO connections (user_id, connected_user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListConnections :many
SELECT u.* FROM users u
JOIN connections c ON u.id = c.connected_user_id
WHERE c.user_id = $1;

-- name: CreateGroup :one
INSERT INTO groups (name, created_by)
VALUES ($1, $2)
RETURNING *;

-- name: AddGroupMember :exec
INSERT INTO group_members (group_id, user_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ListGroups :many
SELECT g.* FROM groups g
JOIN group_members gm ON g.id = gm.group_id
WHERE gm.user_id = $1;

-- name: GetGroupMembers :many
SELECT user_id FROM group_members WHERE group_id = $1;

-- name: CreateExpense :one
INSERT INTO expenses (group_id, description, currency_code, total_amount, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateExpenseSplit :exec
INSERT INTO expense_splits (expense_id, user_id, paid_amount, owed_amount)
VALUES ($1, $2, $3, $4);

-- name: GetUserExpenses :many
SELECT e.* FROM expenses e
JOIN expense_splits es ON e.id = es.expense_id
WHERE es.user_id = $1 AND (e.group_id = $2 OR $2 IS NULL);

-- name: GetExpenseSplits :many
SELECT * FROM expense_splits WHERE expense_id = $1;

-- name: GetUserBalances :many
SELECT currency_code, SUM(paid_amount - owed_amount) as balance
FROM expenses e
JOIN expense_splits es ON e.id = es.expense_id
WHERE es.user_id = $1
GROUP BY currency_code;

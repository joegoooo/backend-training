-- name: Create :one
INSERT INTO users (email)
VALUES ($1)
RETURNING *;

-- name: ExistsByEmail :one
SELECT  EXISTS(SELECT 1 FROM users WHERE email = $1) AS exists;

-- name: GetByEmail :one
SELECT * FROM users
WHERE email = $1
LIMIT 1;

-- name: GetByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;
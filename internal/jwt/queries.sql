-- name: Create :one
INSERT INTO jwt (user_id, expiration_time)
VALUES ($1, $2)
RETURNING *;

-- name: Update :one
UPDATE jwt SET is_available = $2
where id = $1
RETURNING *;

-- name: IsAvailable :one
SELECT * FROM jwt
where id = $1
limit 1;


-- name: Create :one
INSERT INTO forms (title, description, author_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: List :many
SELECT * FROM forms
ORDER BY id;

-- name: Update :one
UPDATE forms SET title = $2, description = $3
where id = $1
RETURNING *;

-- name: Delete :exec
DELETE FROM forms WHERE id = $1;

-- name: IsBookmarked :one
SELECT EXISTS(
    SELECT 1 FROM bookmarks
    where form_id = $1 and user_id = $2
);
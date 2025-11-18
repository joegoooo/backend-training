-- name: GetFormsByUserID :many
SELECT
    b.form_id,
    f.title,
    f.description,
    f.author_id,
    f.created_at
FROM bookmarks b
JOIN forms f ON b.form_id = f.id
WHERE b.user_id = $1;

-- name: Create :one
INSERT INTO bookmarks (form_id, user_id)
VALUES ($1, $2)
RETURNING *;

-- name: Delete :exec
DELETE FROM bookmarks
WHERE form_id = $1 AND user_id = $2;

-- name: Exist :one
SELECT EXISTS (SELECT 1 FROM bookmarks WHERE form_id = $1 AND user_id = $2) AS exists;

-- name: CountBookmark :one
SELECT
    COUNT(*) AS form_count
FROM bookmarks
WHERE user_id = $1;

-- name: FormCount :one
SELECT
    COUNT(*) AS user_count
FROM bookmarks
WHERE form_id = $1;

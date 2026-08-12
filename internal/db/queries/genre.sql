-- name: UpsertGenre :exec
INSERT INTO genres (id, name)
VALUES ($1, $2)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    updated_at = NOW();

-- name: GetGenreByID :one
SELECT * FROM genres WHERE id = $1;

-- name: ListGenres :many
SELECT * FROM genres
ORDER BY name;

-- name: DeleteGenre :exec
DELETE FROM genres WHERE id = $1;
-- name: GetTVShowByID :one
SELECT * FROM tv_shows WHERE id = $1;

-- name: ListTVShows :many
SELECT * FROM tv_shows
ORDER BY popularity DESC
LIMIT $1 OFFSET $2;

-- name: CountTVShows :one
SELECT COUNT(*) FROM tv_shows;

-- name: SearchTVShows :many
SELECT * FROM tv_shows
WHERE name ILIKE '%' || $1 || '%'
ORDER BY popularity DESC
LIMIT $2 OFFSET $3;

-- name: ListTVShowsByGenre :many
SELECT * FROM tv_shows
WHERE genre_ids @> $1::integer[]
ORDER BY popularity DESC
LIMIT $2 OFFSET $3;

-- name: CountTVShowsByGenre :one
SELECT COUNT(*) FROM tv_shows
WHERE genre_ids @> $1::integer[];

-- name: GetTVShowsByIDs :many
SELECT * FROM tv_shows
WHERE id = ANY($1::bigint[])
ORDER BY popularity DESC;

-- name: DeleteTVShow :exec
DELETE FROM tv_shows WHERE id = $1;
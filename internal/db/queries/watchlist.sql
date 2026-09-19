-- name: CountWatchlistByUserID :one
SELECT COUNT(*) FROM user_watchlist WHERE user_id = $1;

-- name: ListWatchlistByUserID :many
SELECT
    wl.id,
    wl.user_id,
    COALESCE(wl.movie_id, wl.tv_id) AS media_id,
    CASE WHEN wl.movie_id IS NOT NULL THEN 'movie' ELSE 'tv' END AS media_type,
    COALESCE(m.title, t.name) AS media_title,
    COALESCE(m.poster_path, t.poster_path) AS media_poster_path,
    wl.added_at
FROM user_watchlist wl
LEFT JOIN movies m ON wl.movie_id = m.id
LEFT JOIN tv_shows t ON wl.tv_id = t.id
WHERE wl.user_id = $1
ORDER BY wl.added_at DESC
LIMIT $2 OFFSET $3;
-- name: AddMovieToWatchlist :one
INSERT INTO user_watchlist (
    user_id,
    movie_id
)
VALUES (
    sqlc.arg(user_id),
    sqlc.arg(movie_id)
)
RETURNING *;
-- name: AddTVToWatchlist :one
INSERT INTO user_watchlist (
    user_id,
    tv_id
)
VALUES (
    sqlc.arg(user_id),
    sqlc.arg(tv_id)
)
RETURNING *;
-- name: IsMovieInWatchlist :one
SELECT EXISTS (
    SELECT 1
    FROM user_watchlist
    WHERE user_id = sqlc.arg(user_id)
      AND movie_id = sqlc.arg(movie_id)
);
-- name: IsTVInWatchlist :one
SELECT EXISTS (
    SELECT 1
    FROM user_watchlist
    WHERE user_id = sqlc.arg(user_id)
      AND tv_id = sqlc.arg(tv_id)
);
-- name: RemoveMovieFromWatchlist :execrows
DELETE FROM user_watchlist
WHERE user_id = $1
  AND movie_id = $2;
-- name: RemoveTVFromWatchlist :execrows
DELETE FROM user_watchlist
WHERE user_id = $1
  AND tv_id = $2;
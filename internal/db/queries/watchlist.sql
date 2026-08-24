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
-- name: GetTVVideosByShowID :many
SELECT * FROM tv_videos
WHERE tv_id = $1 AND type = $2
ORDER BY official DESC, published_at DESC
LIMIT $3 OFFSET $4;

-- name: CountTVVideosByType :one
SELECT COUNT(*) FROM tv_videos
WHERE tv_id = $1 AND type = $2;
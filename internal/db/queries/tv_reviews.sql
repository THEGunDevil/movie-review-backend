-- name: GetTVReviewsByShow :many
SELECT
    r.id,
    r.user_id,
    r.movie_id,
    r.tv_id,
    r.rating,
    r.content,
    r.contains_spoilers,
    r.created_at,
    r.updated_at,
    u.user_name,
    u.email,
    u.profile_picture
FROM reviews r
JOIN users u ON u.id = r.user_id
WHERE r.tv_id = $1
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountTVReviews :one
SELECT COUNT(*) FROM reviews WHERE tv_id = $1;

-- name: CreateTVReview :one
INSERT INTO reviews (user_id, tv_id, rating, content, contains_spoilers)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;


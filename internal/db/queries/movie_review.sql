-- name: CreateReview :one
INSERT INTO reviews (user_id, movie_id, rating, content, contains_spoilers)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetReviewsByMovieID :many
SELECT
    r.id,
    r.user_id,
    r.movie_id,
    r.rating,
    r.content,
    r.contains_spoilers,
    r.created_at,
    r.updated_at,
    u.user_name AS user_name,
    u.email AS user_email,
    u.profile_picture AS profile_picture
FROM reviews r
JOIN users u ON u.id = r.user_id
WHERE r.movie_id = $1
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountReviewsByMovieID :one
SELECT COUNT(*) FROM reviews WHERE movie_id = $1;

-- name: UpdateReview :exec
UPDATE reviews
SET rating = $2, content = $3, contains_spoilers = $4, updated_at = NOW()
WHERE id = $1 AND user_id = $5;

-- name: DeleteReview :exec
DELETE FROM reviews WHERE id = $1 AND user_id = $2;

-- name: GetReviewsByUserID :many
SELECT
    r.id, r.movie_id, r.rating, r.content,
    r.contains_spoilers, r.created_at,
    m.title AS movie_title, m.poster_path AS movie_poster
FROM reviews r
JOIN movies m ON m.id = r.movie_id
WHERE r.user_id = $1
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetReviewByID :one
SELECT
    r.id, r.user_id, r.movie_id, r.rating, r.content,
    r.contains_spoilers, r.created_at, r.updated_at,
    u.user_name AS user_name, u.email AS user_email
FROM reviews r
JOIN users u ON u.id = r.user_id
WHERE r.id = $1;
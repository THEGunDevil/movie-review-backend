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

-- name: CountReviewsByUserID :one
SELECT COUNT(*) FROM reviews WHERE user_id = $1;

-- name: UpdateReview :exec
UPDATE reviews
SET rating = $2, content = $3, contains_spoilers = $4, updated_at = NOW()
WHERE id = $1 AND user_id = $5;

-- name: DeleteReview :exec
DELETE FROM reviews WHERE id = $1 AND user_id = $2;

-- name: ListReviewsByUserID :many
SELECT
    r.id,
    r.user_id,
    u.user_name,
    u.profile_picture AS user_profile_picture,
    r.rating,
    r.content,
    r.contains_spoilers,
    r.created_at,
    r.updated_at,
    CASE WHEN r.movie_id IS NOT NULL THEN 'movie' ELSE 'tv' END AS media_type,
    COALESCE(m.title, t.name) AS media_title,
    COALESCE(m.poster_path, t.poster_path) AS media_poster_path,
    COALESCE(m.id, t.id) AS media_id,
    (SELECT COUNT(*) FROM review_likes rl WHERE rl.review_id = r.id) AS like_count,
    (SELECT COUNT(*) FROM review_comments rc WHERE rc.review_id = r.id) AS comment_count,
    (SELECT COUNT(*) FROM review_votes rv WHERE rv.review_id = r.id AND rv.vote = 'up') AS upvotes,
    (SELECT COUNT(*) FROM review_votes rv WHERE rv.review_id = r.id AND rv.vote = 'down') AS downvotes,
    0 AS view_count,  -- replace with actual view count if you have a column
    CASE
        WHEN EXISTS (
            SELECT 1 FROM review_likes rl
            WHERE rl.review_id = r.id AND rl.user_id = $2
        ) THEN TRUE ELSE FALSE
    END AS user_liked,
    CASE
        WHEN EXISTS (
            SELECT 1 FROM review_votes rv
            WHERE rv.review_id = r.id AND rv.user_id = $2 AND rv.vote = 'up'
        ) THEN 'up'
        WHEN EXISTS (
            SELECT 1 FROM review_votes rv
            WHERE rv.review_id = r.id AND rv.user_id = $2 AND rv.vote = 'down'
        ) THEN 'down'
        ELSE NULL
    END AS user_vote,
    FALSE AS user_saved  -- or implement saved logic if needed
FROM reviews r
JOIN users u ON r.user_id = u.id
LEFT JOIN movies m ON r.movie_id = m.id
LEFT JOIN tv_shows t ON r.tv_id = t.id
WHERE r.user_id = $1
ORDER BY r.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetReviewByID :one
SELECT
    r.id, r.user_id, r.movie_id, r.rating, r.content,
    r.contains_spoilers, r.created_at, r.updated_at,
    u.user_name AS user_name, u.email AS user_email
FROM reviews r
JOIN users u ON u.id = r.user_id
WHERE r.id = $1;
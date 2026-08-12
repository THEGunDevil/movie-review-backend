-- name: VoteOnReview :exec
INSERT INTO review_votes (review_id, user_id, vote)
VALUES ($1, $2, $3)
ON CONFLICT (review_id, user_id) DO UPDATE SET vote = EXCLUDED.vote;

-- name: RemoveVote :exec
DELETE FROM review_votes WHERE review_id = $1 AND user_id = $2;

-- name: GetVoteCounts :one
SELECT
    COUNT(*) FILTER (WHERE vote = 'up') AS upvotes,
    COUNT(*) FILTER (WHERE vote = 'down') AS downvotes
FROM review_votes WHERE review_id = $1;

-- name: GetUserVote :one
SELECT vote FROM review_votes WHERE review_id = $1 AND user_id = $2;

-- name: LikeReview :exec
INSERT INTO review_likes (review_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnlikeReview :exec
DELETE FROM review_likes WHERE review_id = $1 AND user_id = $2;

-- name: GetLikeCount :one
SELECT COUNT(*) FROM review_likes WHERE review_id = $1;

-- name: HasUserLiked :one
SELECT EXISTS(SELECT 1 FROM review_likes WHERE review_id = $1 AND user_id = $2);

-- name: AddComment :one
INSERT INTO review_comments (review_id, user_id, content) VALUES ($1, $2, $3) RETURNING *;

-- name: GetCommentsByReview :many
SELECT
    c.id, c.review_id, c.user_id, c.content,
    c.created_at, c.updated_at,
    u.user_name, u.email, u.profile_picture AS user_profile_picture
FROM review_comments c
JOIN users u ON u.id = c.user_id
WHERE c.review_id = $1
ORDER BY c.created_at ASC;

-- name: ReportReview :exec
INSERT INTO review_reports (review_id, user_id, reason) VALUES ($1, $2, $3);
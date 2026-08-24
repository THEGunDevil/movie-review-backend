-- name: CreateUser :one
INSERT INTO users (user_name, email, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUserNameByID :one
SELECT user_name FROM users WHERE id = $1;
-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUserProfile :one
UPDATE users
SET user_name = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
-- name: UpdatePassword :exec
UPDATE users SET password_hash = $2, token_version = token_version + 1, updated_at = NOW()
WHERE id = $1;
-- name: UpdateUserRole :exec
UPDATE users 
SET role = $2, updated_at = NOW()
WHERE id = $1;
-- name: IncrementTokenVersion :exec
-- Invalidates all existing refresh tokens for a user (e.g. on logout-everywhere or password change)
UPDATE users SET token_version = token_version + 1 WHERE id = $1;

-- name: BanUser :exec
UPDATE users
SET is_banned = true, ban_reason = $2, ban_until = $3, is_permanent_ban = $4
WHERE id = $1;

-- name: UnbanUser :exec
UPDATE users
SET is_banned = false, ban_reason = NULL, ban_until = NULL, is_permanent_ban = false
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;
-- name: CountUsersByEmail :one
SELECT COUNT(*)
FROM users
WHERE
    (CASE 
        WHEN $1 = '' THEN TRUE
        ELSE email ILIKE '%' || $1 || '%'
    END);


-- name: CountUsers :one
SELECT COUNT(*) FROM users;
-- name: ListUsersPaginated :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListBannedUsersPaginated :many
SELECT id, user_name, email, role, is_banned, is_permanent_ban, ban_reason, ban_until, created_at, token_version
FROM users
WHERE is_banned = TRUE
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountBannedUsers :one
SELECT COUNT(*) FROM users WHERE is_banned = TRUE;
-- name: GetUserProfile :one
SELECT
    u.id,
    u.user_name,
    u.profile_picture,
    u.bio,
    u.created_at AS join_date,
    (SELECT COUNT(*) FROM reviews r WHERE r.user_id = u.id) AS review_count,
    (SELECT COUNT(*) FROM review_likes rl
        JOIN reviews r ON r.id = rl.review_id
        WHERE r.user_id = u.id) AS like_count,
    (SELECT COUNT(*) FROM review_comments rc WHERE rc.user_id = u.id) AS comment_count,
    (SELECT COUNT(*) FROM follows f WHERE f.following_id = u.id) AS follower_count,
    (SELECT COUNT(*) FROM follows f WHERE f.follower_id = u.id) AS following_count,
    EXISTS (
        SELECT 1 FROM follows f
        WHERE f.follower_id = $2 AND f.following_id = $1
    ) AS is_following
FROM users u
WHERE u.id = $1;
-- name: GetAllReviews :many
SELECT
    r.id,
    r.rating,
    r.content,
    r.contains_spoilers,
    r.created_at,
    r.updated_at,
    r.user_id,
    u.user_name,
    u.profile_picture AS user_profile_picture,
    COALESCE(m.id, t.id) AS media_id,
    COALESCE(m.title, t.name) AS media_title,
    CASE
        WHEN r.movie_id IS NOT NULL THEN 'movie'
        ELSE 'tv'
    END AS media_type,
    COALESCE(m.poster_path, t.poster_path) AS media_poster_path,
    COALESCE(v.upvotes, 0) AS upvotes,
    COALESCE(v.downvotes, 0) AS downvotes,
    COALESCE(c.cnt, 0) AS comment_count,
    COALESCE(l.cnt, 0) AS like_count,
    0 AS view_count,
    rv.vote AS user_vote,
    CASE WHEN rl.review_id IS NOT NULL THEN TRUE ELSE FALSE END AS user_liked,
    FALSE AS user_saved
FROM reviews r
JOIN users u ON u.id = r.user_id
LEFT JOIN movies m ON m.id = r.movie_id
LEFT JOIN tv_shows t ON t.id = r.tv_id

LEFT JOIN LATERAL (
    SELECT
        COUNT(*) FILTER (WHERE vote = 'up') AS upvotes,
        COUNT(*) FILTER (WHERE vote = 'down') AS downvotes
    FROM review_votes
    WHERE review_id = r.id
) v ON TRUE

LEFT JOIN LATERAL (
    SELECT COUNT(*) AS cnt
    FROM review_comments
    WHERE review_id = r.id
) c ON TRUE

LEFT JOIN LATERAL (
    SELECT COUNT(*) AS cnt
    FROM review_likes
    WHERE review_id = r.id
) l ON TRUE

LEFT JOIN review_votes rv
    ON rv.review_id = r.id
    AND rv.user_id = sqlc.arg(user_id)

LEFT JOIN review_likes rl
    ON rl.review_id = r.id
    AND rl.user_id = sqlc.arg(user_id)

WHERE (
    sqlc.arg(media_type)::text = 'all'
    OR (
        r.movie_id IS NOT NULL
        AND sqlc.arg(media_type)::text = 'movie'
    )
    OR (
        r.tv_id IS NOT NULL
        AND sqlc.arg(media_type)::text = 'tv'
    )
)

ORDER BY r.created_at DESC
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountAllReviews :one
SELECT COUNT(*) FROM reviews;

-- name: GetTopRatedMediaLastWeek :one
WITH last_week_reviews AS (
    SELECT
        r.movie_id,
        r.tv_id,
        CASE WHEN r.movie_id IS NOT NULL THEN 'movie' ELSE 'tv' END AS media_type,
        COALESCE(m.title, t.name) AS media_title,
        COALESCE(m.poster_path, t.poster_path) AS media_poster_path,
        AVG(r.rating) AS avg_rating,
        COUNT(*) AS review_count
    FROM reviews r
    LEFT JOIN movies m ON r.movie_id = m.id
    LEFT JOIN tv_shows t ON r.tv_id = t.id
    WHERE r.created_at >= NOW() - INTERVAL '7 days'
    GROUP BY r.movie_id, r.tv_id, media_type, media_title, media_poster_path
),
top_review_text AS (
    SELECT DISTINCT ON (lr.movie_id, lr.tv_id)
        lr.movie_id,
        lr.tv_id,
        r2.content AS top_review,
        r2.user_id,
        u.user_name,
        r2.created_at
    FROM last_week_reviews lr
    JOIN reviews r2 ON (
        (lr.movie_id IS NOT NULL AND r2.movie_id = lr.movie_id) OR
        (lr.tv_id IS NOT NULL AND r2.tv_id = lr.tv_id)
    )
    JOIN users u ON u.id = r2.user_id
    LEFT JOIN review_likes rl ON rl.review_id = r2.id
    WHERE r2.created_at >= NOW() - INTERVAL '7 days'
    GROUP BY lr.movie_id, lr.tv_id, r2.id, u.user_name
    ORDER BY lr.movie_id, lr.tv_id, COUNT(rl.user_id) DESC
)
SELECT
    COALESCE(lr.movie_id::text, lr.tv_id::text) AS media_id,
    lr.media_type,
    lr.media_title,
    lr.media_poster_path AS poster_path,
    lr.avg_rating,
    lr.review_count,
    tr.top_review,
    tr.user_name,
    tr.user_id,
    tr.created_at,
    -- জঁরা আনার জন্য
    CASE
        WHEN lr.movie_id IS NOT NULL THEN (
            SELECT ARRAY_AGG(g.name)
            FROM genres g
            JOIN movies m2 ON m2.id = lr.movie_id
            WHERE g.id = ANY(m2.genre_ids)
        )
        ELSE (
            SELECT ARRAY_AGG(g.name)
            FROM genres g
            JOIN tv_shows t2 ON t2.id = lr.tv_id
            WHERE g.id = ANY(t2.genre_ids)
        )
    END AS genres
FROM last_week_reviews lr
JOIN top_review_text tr ON (
    (lr.movie_id IS NOT NULL AND tr.movie_id = lr.movie_id) OR
    (lr.tv_id IS NOT NULL AND tr.tv_id = lr.tv_id)
)
ORDER BY lr.avg_rating DESC, lr.review_count DESC
LIMIT 1;

-- name: GetTopRatedMediaByPeriod :one
WITH period_reviews AS (
    SELECT
        r.movie_id,
        r.tv_id,
        CASE WHEN r.movie_id IS NOT NULL THEN 'movie' ELSE 'tv' END AS media_type,
        COALESCE(m.title, t.name) AS media_title,
        COALESCE(m.poster_path, t.poster_path) AS media_poster_path,
        AVG(r.rating) AS avg_rating,
        COUNT(*) AS review_count
    FROM reviews r
    LEFT JOIN movies m ON r.movie_id = m.id
    LEFT JOIN tv_shows t ON r.tv_id = t.id
    WHERE r.created_at >= $1 AND r.created_at <= $2
    GROUP BY r.movie_id, r.tv_id, media_type, media_title, media_poster_path
),
top_review_text AS (
    SELECT DISTINCT ON (pr.movie_id, pr.tv_id)
        pr.movie_id,
        pr.tv_id,
        r2.content AS top_review,
        r2.user_id,
        u.user_name,
        r2.created_at
    FROM period_reviews pr
    JOIN reviews r2 ON (
        (pr.movie_id IS NOT NULL AND r2.movie_id = pr.movie_id) OR
        (pr.tv_id IS NOT NULL AND r2.tv_id = pr.tv_id)
    )
    JOIN users u ON u.id = r2.user_id
    LEFT JOIN review_likes rl ON rl.review_id = r2.id
    WHERE r2.created_at >= $1 AND r2.created_at <= $2
    GROUP BY pr.movie_id, pr.tv_id, r2.id, u.user_name
    ORDER BY pr.movie_id, pr.tv_id, COUNT(rl.user_id) DESC
)
SELECT
    COALESCE(pr.movie_id::text, pr.tv_id::text) AS media_id,
    pr.media_type,
    pr.media_title,
    pr.media_poster_path AS poster_path,
    pr.avg_rating,
    pr.review_count,
    tr.top_review,
    tr.user_name,
    tr.user_id::text AS user_id,
    tr.created_at,
    CASE
        WHEN pr.movie_id IS NOT NULL THEN (
            SELECT ARRAY_AGG(g.name)
            FROM genres g
            JOIN movies m2 ON m2.id = pr.movie_id
            WHERE g.id = ANY(m2.genre_ids)
        )
        ELSE (
            SELECT ARRAY_AGG(g.name)
            FROM genres g
            JOIN tv_shows t2 ON t2.id = pr.tv_id
            WHERE g.id = ANY(t2.genre_ids)
        )
    END AS genres
FROM period_reviews pr
JOIN top_review_text tr ON (
    (pr.movie_id IS NOT NULL AND tr.movie_id = pr.movie_id) OR
    (pr.tv_id IS NOT NULL AND tr.tv_id = pr.tv_id)
)
ORDER BY pr.avg_rating DESC, pr.review_count DESC
LIMIT 1;
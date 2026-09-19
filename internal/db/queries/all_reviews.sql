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

    -- Media ID
    CASE
        WHEN r.movie_id IS NOT NULL THEN r.movie_id
        ELSE r.tv_id
    END AS media_id,

    -- Media title
    CASE
        WHEN r.movie_id IS NOT NULL THEN m.title
        ELSE tv.name
    END AS media_title,

    -- Media type
    CASE
        WHEN r.movie_id IS NOT NULL THEN 'movie'
        ELSE 'tv'
    END AS media_type,

    -- Poster
    CASE
        WHEN r.movie_id IS NOT NULL THEN m.poster_path
        ELSE tv.poster_path
    END AS media_poster_path,

    -- Upvotes
    (
        SELECT COUNT(*)
        FROM review_votes rv
        WHERE rv.review_id = r.id
          AND rv.vote = 'up'
    )::bigint AS upvotes,

    -- Downvotes
    (
        SELECT COUNT(*)
        FROM review_votes rv
        WHERE rv.review_id = r.id
          AND rv.vote = 'down'
    )::bigint AS downvotes,

    -- Comments
    (
        SELECT COUNT(*)
        FROM review_comments rc
        WHERE rc.review_id = r.id
    )::bigint AS comment_count,

    -- Likes
    (
        SELECT COUNT(*)
        FROM review_likes rl
        WHERE rl.review_id = r.id
    )::bigint AS like_count,

    -- Current user's vote
COALESCE(
    (
        SELECT rv.vote::text
        FROM review_votes rv
        WHERE rv.review_id = r.id
          AND rv.user_id = sqlc.arg(user_id)
        LIMIT 1
    ),
    ''
) AS user_vote,

    EXISTS (
        SELECT 1
        FROM review_likes rl
        WHERE rl.review_id = r.id
          AND rl.user_id = sqlc.arg(user_id)
    ) AS user_liked

FROM reviews r

INNER JOIN users u
    ON u.id = r.user_id

LEFT JOIN movies m
    ON m.id = r.movie_id

LEFT JOIN tv_shows tv
    ON tv.id = r.tv_id

WHERE
    -- Media filter
    (
        sqlc.arg(media_type) = 'all'
        OR (
            sqlc.arg(media_type) = 'movie'
            AND r.movie_id IS NOT NULL
        )
        OR (
            sqlc.arg(media_type) = 'tv'
            AND r.tv_id IS NOT NULL
        )
    )

    -- Search
    AND (
        sqlc.arg(search) = ''
        OR r.content ILIKE '%' || sqlc.arg(search) || '%'
        OR u.user_name ILIKE '%' || sqlc.arg(search) || '%'
        OR (
            r.movie_id IS NOT NULL
            AND m.title ILIKE '%' || sqlc.arg(search) || '%'
        )
        OR (
            r.tv_id IS NOT NULL
            AND tv.name ILIKE '%' || sqlc.arg(search) || '%'
        )
    )

    -- Rating filter
    AND (
        sqlc.arg(min_rating) IS NULL
        OR r.rating >= sqlc.arg(min_rating)
    )

ORDER BY
    CASE
        WHEN sqlc.arg(sort) = 'newest'
        THEN r.created_at
    END DESC,

    CASE
        WHEN sqlc.arg(sort) = 'highest_rated'
        THEN r.rating
    END DESC,

    CASE
        WHEN sqlc.arg(sort) = 'popular'
        THEN (
            (
                SELECT COUNT(*)
                FROM review_likes rl
                WHERE rl.review_id = r.id
            )
            +
            (
                SELECT COUNT(*)
                FROM review_votes rv
                WHERE rv.review_id = r.id
                  AND rv.vote = 'up'
            )
        )
    END DESC,

    CASE
        WHEN sqlc.arg(sort) = 'discussed'
        THEN (
            SELECT COUNT(*)
            FROM review_comments rc
            WHERE rc.review_id = r.id
        )
    END DESC,

    r.created_at DESC

LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);
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


-- name: GetReviewsNewest :many
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

    r.movie_id,
    r.tv_id,

    CASE
        WHEN r.movie_id IS NOT NULL THEN m.title
        WHEN r.tv_id IS NOT NULL THEN tv.name
    END AS media_title,

    CASE
        WHEN r.movie_id IS NOT NULL THEN 'movie'
        WHEN r.tv_id IS NOT NULL THEN 'tv'
    END AS media_type,

    CASE
        WHEN r.movie_id IS NOT NULL THEN m.poster_path
        WHEN r.tv_id IS NOT NULL THEN tv.poster_path
    END AS media_poster_path,

    COALESCE(v.upvotes, 0)::bigint AS upvotes,
    COALESCE(v.downvotes, 0)::bigint AS downvotes,
    COALESCE(c.comment_count, 0)::bigint AS comment_count,
    COALESCE(l.like_count, 0)::bigint AS like_count,

COALESCE(uv.vote, '') AS user_vote,
EXISTS (
    SELECT 1
    FROM review_likes rl
    WHERE rl.review_id = r.id
      AND rl.user_id = sqlc.arg(user_id)
) AS user_liked
FROM reviews r

JOIN users u
    ON u.id = r.user_id

LEFT JOIN movies m
    ON m.id = r.movie_id

LEFT JOIN tv_shows tv
    ON tv.id = r.tv_id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) FILTER (WHERE vote = 'up') AS upvotes,
        COUNT(*) FILTER (WHERE vote = 'down') AS downvotes
    FROM review_votes
    GROUP BY review_id
) v
    ON v.review_id = r.id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) AS comment_count
    FROM review_comments
    GROUP BY review_id
) c
    ON c.review_id = r.id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) AS like_count
    FROM review_likes
    GROUP BY review_id
) l
    ON l.review_id = r.id

LEFT JOIN review_votes uv
    ON uv.review_id = r.id
    AND uv.user_id = sqlc.arg(user_id)

LEFT JOIN review_likes ul
    ON ul.review_id = r.id
    AND ul.user_id = sqlc.arg(user_id)

WHERE
    (
        sqlc.arg(media_type) = 'all'
        OR (
            sqlc.arg(media_type) = 'movie'
            AND r.movie_id IS NOT NULL
        )
        OR (
            sqlc.arg(media_type) = 'tv'
            AND r.tv_id IS NOT NULL
        )
    )

    AND (
        sqlc.arg(search) = ''
        OR COALESCE(m.title, tv.name) ILIKE '%' || sqlc.arg(search) || '%'
        OR r.content ILIKE '%' || sqlc.arg(search) || '%'
        OR u.user_name ILIKE '%' || sqlc.arg(search) || '%'
    )

    AND (
        sqlc.arg(min_rating) IS NULL
        OR r.rating >= sqlc.arg(min_rating)
    )

ORDER BY r.created_at DESC

LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: GetReviewsHighestRated :many
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

    r.movie_id,
    r.tv_id,

    CASE
        WHEN r.movie_id IS NOT NULL THEN m.title
        WHEN r.tv_id IS NOT NULL THEN tv.name
    END AS media_title,

    CASE
        WHEN r.movie_id IS NOT NULL THEN 'movie'
        ELSE 'tv'
    END AS media_type,

    CASE
        WHEN r.movie_id IS NOT NULL THEN m.poster_path
        ELSE tv.poster_path
    END AS media_poster_path,

    COALESCE(v.upvotes, 0)::bigint AS upvotes,
    COALESCE(v.downvotes, 0)::bigint AS downvotes,
    COALESCE(c.comment_count, 0)::bigint AS comment_count,
    COALESCE(l.like_count, 0)::bigint AS like_count,

COALESCE(uv.vote, '') AS user_vote,
EXISTS (
    SELECT 1
    FROM review_likes rl
    WHERE rl.review_id = r.id
      AND rl.user_id = sqlc.arg(user_id)
) AS user_liked
FROM reviews r

JOIN users u
    ON u.id = r.user_id

LEFT JOIN movies m
    ON m.id = r.movie_id

LEFT JOIN tv_shows tv
    ON tv.id = r.tv_id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) FILTER (WHERE vote = 'up') AS upvotes,
        COUNT(*) FILTER (WHERE vote = 'down') AS downvotes
    FROM review_votes
    GROUP BY review_id
) v ON v.review_id = r.id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) AS comment_count
    FROM review_comments
    GROUP BY review_id
) c ON c.review_id = r.id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) AS like_count
    FROM review_likes
    GROUP BY review_id
) l ON l.review_id = r.id

LEFT JOIN review_votes uv
    ON uv.review_id = r.id
    AND uv.user_id = sqlc.arg(user_id)

LEFT JOIN review_likes ul
    ON ul.review_id = r.id
    AND ul.user_id = sqlc.arg(user_id)

WHERE
    (
        sqlc.arg(media_type) = 'all'
        OR (
            sqlc.arg(media_type) = 'movie'
            AND r.movie_id IS NOT NULL
        )
        OR (
            sqlc.arg(media_type) = 'tv'
            AND r.tv_id IS NOT NULL
        )
    )

    AND (
        sqlc.arg(search) = ''
        OR COALESCE(m.title, tv.name) ILIKE '%' || sqlc.arg(search) || '%'
        OR r.content ILIKE '%' || sqlc.arg(search) || '%'
        OR u.user_name ILIKE '%' || sqlc.arg(search) || '%'
    )

    AND (
        sqlc.arg(min_rating) IS NULL
        OR r.rating >= sqlc.arg(min_rating)
    )

ORDER BY
    r.rating DESC,
    r.created_at DESC

LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);


-- name: GetReviewsPopular :many
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

    r.movie_id,
    r.tv_id,

    CASE
        WHEN r.movie_id IS NOT NULL THEN m.title
        ELSE tv.name
    END AS media_title,

    CASE
        WHEN r.movie_id IS NOT NULL THEN 'movie'
        ELSE 'tv'
    END AS media_type,

    CASE
        WHEN r.movie_id IS NOT NULL THEN m.poster_path
        ELSE tv.poster_path
    END AS media_poster_path,

    COALESCE(v.upvotes, 0)::bigint AS upvotes,
    COALESCE(v.downvotes, 0)::bigint AS downvotes,
    COALESCE(c.comment_count, 0)::bigint AS comment_count,
    COALESCE(l.like_count, 0)::bigint AS like_count,

COALESCE(uv.vote, '') AS user_vote,
EXISTS (
    SELECT 1
    FROM review_likes rl
    WHERE rl.review_id = r.id
      AND rl.user_id = sqlc.arg(user_id)
) AS user_liked
FROM reviews r

JOIN users u
    ON u.id = r.user_id

LEFT JOIN movies m
    ON m.id = r.movie_id

LEFT JOIN tv_shows tv
    ON tv.id = r.tv_id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) FILTER (WHERE vote = 'up') AS upvotes,
        COUNT(*) FILTER (WHERE vote = 'down') AS downvotes
    FROM review_votes
    GROUP BY review_id
) v ON v.review_id = r.id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) AS comment_count
    FROM review_comments
    GROUP BY review_id
) c ON c.review_id = r.id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) AS like_count
    FROM review_likes
    GROUP BY review_id
) l ON l.review_id = r.id

LEFT JOIN review_votes uv
    ON uv.review_id = r.id
    AND uv.user_id = sqlc.arg(user_id)

LEFT JOIN review_likes ul
    ON ul.review_id = r.id
    AND ul.user_id = sqlc.arg(user_id)

WHERE
    (
        sqlc.arg(media_type) = 'all'
        OR (
            sqlc.arg(media_type) = 'movie'
            AND r.movie_id IS NOT NULL
        )
        OR (
            sqlc.arg(media_type) = 'tv'
            AND r.tv_id IS NOT NULL
        )
    )

    AND (
        sqlc.arg(search) = ''
        OR COALESCE(m.title, tv.name) ILIKE '%' || sqlc.arg(search) || '%'
        OR r.content ILIKE '%' || sqlc.arg(search) || '%'
        OR u.user_name ILIKE '%' || sqlc.arg(search) || '%'
    )

    AND (
        sqlc.arg(min_rating) IS NULL
        OR r.rating >= sqlc.arg(min_rating)
    )

ORDER BY
    (
        COALESCE(l.like_count, 0)
        + COALESCE(v.upvotes, 0)
    ) DESC,
    r.created_at DESC

LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);


-- name: GetReviewsDiscussed :many
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

    r.movie_id,
    r.tv_id,

    CASE
        WHEN r.movie_id IS NOT NULL THEN m.title
        ELSE tv.name
    END AS media_title,

    CASE
        WHEN r.movie_id IS NOT NULL THEN 'movie'
        ELSE 'tv'
    END AS media_type,

    CASE
        WHEN r.movie_id IS NOT NULL THEN m.poster_path
        ELSE tv.poster_path
    END AS media_poster_path,

    COALESCE(v.upvotes, 0)::bigint AS upvotes,
    COALESCE(v.downvotes, 0)::bigint AS downvotes,
    COALESCE(c.comment_count, 0)::bigint AS comment_count,
    COALESCE(l.like_count, 0)::bigint AS like_count,

COALESCE(uv.vote, '') AS user_vote,
EXISTS (
    SELECT 1
    FROM review_likes rl
    WHERE rl.review_id = r.id
      AND rl.user_id = sqlc.arg(user_id)
) AS user_liked
FROM reviews r

JOIN users u
    ON u.id = r.user_id

LEFT JOIN movies m
    ON m.id = r.movie_id

LEFT JOIN tv_shows tv
    ON tv.id = r.tv_id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) FILTER (WHERE vote = 'up') AS upvotes,
        COUNT(*) FILTER (WHERE vote = 'down') AS downvotes
    FROM review_votes
    GROUP BY review_id
) v ON v.review_id = r.id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) AS comment_count
    FROM review_comments
    GROUP BY review_id
) c ON c.review_id = r.id

LEFT JOIN (
    SELECT
        review_id,
        COUNT(*) AS like_count
    FROM review_likes
    GROUP BY review_id
) l ON l.review_id = r.id

LEFT JOIN review_votes uv
    ON uv.review_id = r.id
    AND uv.user_id = sqlc.arg(user_id)

LEFT JOIN review_likes ul
    ON ul.review_id = r.id
    AND ul.user_id = sqlc.arg(user_id)

WHERE
    (
        sqlc.arg(media_type) = 'all'
        OR (
            sqlc.arg(media_type) = 'movie'
            AND r.movie_id IS NOT NULL
        )
        OR (
            sqlc.arg(media_type) = 'tv'
            AND r.tv_id IS NOT NULL
        )
    )

    AND (
        sqlc.arg(search) = ''
        OR COALESCE(m.title, tv.name) ILIKE '%' || sqlc.arg(search) || '%'
        OR r.content ILIKE '%' || sqlc.arg(search) || '%'
        OR u.user_name ILIKE '%' || sqlc.arg(search) || '%'
    )

    AND (
        sqlc.arg(min_rating) IS NULL
        OR r.rating >= sqlc.arg(min_rating)
    )

ORDER BY
    COALESCE(c.comment_count, 0) DESC,
    r.created_at DESC

LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);
-- name: CountAllReviews :one

SELECT COUNT(*)
FROM reviews r

INNER JOIN users u
    ON u.id = r.user_id

LEFT JOIN movies m
    ON m.id = r.movie_id

LEFT JOIN tv_shows tv
    ON tv.id = r.tv_id

WHERE
    (
        sqlc.arg(media_type) = 'all'
        OR (
            sqlc.arg(media_type) = 'movie'
            AND r.movie_id IS NOT NULL
        )
        OR (
            sqlc.arg(media_type) = 'tv'
            AND r.tv_id IS NOT NULL
        )
    )

    AND (
        sqlc.arg(search) = ''
        OR r.content ILIKE '%' || sqlc.arg(search) || '%'
        OR u.user_name ILIKE '%' || sqlc.arg(search) || '%'
        OR (
            r.movie_id IS NOT NULL
            AND m.title ILIKE '%' || sqlc.arg(search) || '%'
        )
        OR (
            r.tv_id IS NOT NULL
            AND tv.name ILIKE '%' || sqlc.arg(search) || '%'
        )
    )

    AND (
        sqlc.arg(min_rating) IS NULL
        OR r.rating >= sqlc.arg(min_rating)
    );
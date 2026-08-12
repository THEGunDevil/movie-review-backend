-- name: CreateMovieCredit :one
INSERT INTO movie_credits (
    movie_id, person_id, role, type, "order", department
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetCreditsByMovieID :many
SELECT 
    mc.id,
    mc.movie_id,
    mc.person_id,
    mc.role,
    mc.type,
    mc."order",
    mc.department,
    p.name AS person_name,
    p.profile_path AS person_profile_path
FROM movie_credits mc
JOIN persons p ON mc.person_id = p.id
WHERE mc.movie_id = $1 AND mc.type = $2
ORDER BY mc.type ASC, mc."order" ASC
LIMIT $3 OFFSET $4;

-- name: GetCastByMovieID :many
SELECT 
    mc.id,
    mc.movie_id,
    mc.person_id,
    mc.role,
    mc."order",
    p.name AS person_name,
    p.profile_path AS person_profile_path
FROM movie_credits mc
JOIN persons p ON mc.person_id = p.id
WHERE mc.movie_id = $1 AND mc.type = 'cast'
ORDER BY mc."order" ASC;

-- name: GetCrewByMovieID :many
SELECT 
    mc.id,
    mc.movie_id,
    mc.person_id,
    mc.role,
    mc.department,
    p.name AS person_name,
    p.profile_path AS person_profile_path
FROM movie_credits mc
JOIN persons p ON mc.person_id = p.id
WHERE mc.movie_id = $1 AND mc.type = 'crew'
ORDER BY mc.department ASC;

-- name: DeleteCreditsByMovieID :exec
DELETE FROM movie_credits
WHERE movie_id = $1;

-- name: CountCreditsByType :one
SELECT COUNT(*) FROM movie_credits
WHERE movie_id = $1 AND type = $2;
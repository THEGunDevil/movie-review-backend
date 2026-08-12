-- name: GetTVCreditsByShowID :many
SELECT
    c.id,
    c.tv_id,
    c.person_id,
    c.role,
    c.type,
    c."order",
    c.department,
    p.name AS person_name,
    p.profile_path AS person_profile_path
FROM tv_credits c
JOIN persons p ON p.id = c.person_id
WHERE c.tv_id = $1 AND c.type = $2
ORDER BY c.type, c."order"
LIMIT $3 OFFSET $4;

-- name: CountTVCreditsByType :one
SELECT COUNT(*) FROM tv_credits
WHERE tv_id = $1 AND type = $2;
-- name: GetPersonByTVCreditID :one
SELECT p.*
FROM persons p
JOIN tv_credits c ON c.person_id = p.id
WHERE c.id = $1;
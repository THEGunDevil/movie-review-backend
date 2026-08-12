-- name: UpsertPerson :one
INSERT INTO persons (
    id, name, profile_path, popularity, known_for_department, updated_at
) VALUES (
    $1, $2, $3, $4, $5, NOW()
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    profile_path = EXCLUDED.profile_path,
    popularity = EXCLUDED.popularity,
    known_for_department = EXCLUDED.known_for_department,
    updated_at = NOW()
RETURNING *;

-- name: GetPersonByID :one
SELECT * FROM persons
WHERE id = $1 LIMIT 1;
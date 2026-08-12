-- sqlc queries for movies

-- name: UpsertMovie :exec
INSERT INTO movies (
    id, title, original_language, original_title, overview,
    release_date, popularity, vote_average, vote_count,
    poster_path, backdrop_path, adult, genre_ids, softcore, video,
    runtime, budget, revenue, homepage, imdb_id, status, tagline
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12, $13, $14, $15,
    $16, $17, $18, $19, $20, $21, $22
)
ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    original_language = EXCLUDED.original_language,
    original_title = EXCLUDED.original_title,
    overview = EXCLUDED.overview,
    release_date = EXCLUDED.release_date,
    popularity = EXCLUDED.popularity,
    vote_average = EXCLUDED.vote_average,
    vote_count = EXCLUDED.vote_count,
    poster_path = EXCLUDED.poster_path,
    backdrop_path = EXCLUDED.backdrop_path,
    adult = EXCLUDED.adult,
    genre_ids = EXCLUDED.genre_ids,
    softcore = EXCLUDED.softcore,
    video = EXCLUDED.video,
    runtime = EXCLUDED.runtime,
    budget = EXCLUDED.budget,
    revenue = EXCLUDED.revenue,
    homepage = EXCLUDED.homepage,
    imdb_id = EXCLUDED.imdb_id,
    status = EXCLUDED.status,
    tagline = EXCLUDED.tagline,
    updated_at = NOW();

-- name: GetMovieByID :one
SELECT * FROM movies WHERE id = $1;

-- name: ListMovies :many
SELECT * FROM movies
ORDER BY popularity DESC
LIMIT $1 OFFSET $2;

-- name: ListMoviesByGenre :many
SELECT * FROM movies
WHERE genre_ids @> $1::integer[]
ORDER BY popularity DESC
LIMIT $2 OFFSET $3;

-- name: SearchMovies :many
SELECT * FROM movies
WHERE title ILIKE '%' || $1 || '%'
ORDER BY popularity DESC
LIMIT $2 OFFSET $3;

-- name: GetMoviesByIDs :many
SELECT * FROM movies
WHERE id = ANY($1::bigint[])
ORDER BY popularity DESC;

-- name: DeleteMovie :exec
DELETE FROM movies WHERE id = $1;

-- name: CountMoviesByGenre :one
SELECT COUNT(*) FROM movies WHERE $1::int[] <@ genre_ids;
-- name: CountMovies :one
SELECT COUNT(*) FROM movies;

-- name: GetVideosByMovieID :many
SELECT
    *
FROM movie_videos
WHERE movie_id = $1
  AND type = $2
ORDER BY published_at DESC
LIMIT $3
OFFSET $4;


-- name: CountVideosByType :one
SELECT COUNT(*) FROM movie_videos
WHERE movie_id = $1 AND type = $2;
-- sqlc queries for tv_episodes

-- name: UpsertEpisode :exec
INSERT INTO tv_episodes (
    id, tv_id, season_id, name, overview,
    episode_number, season_number, air_date,
    still_path, vote_average, vote_count, runtime
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8,
    $9, $10, $11, $12
)
ON CONFLICT (id) DO UPDATE SET
    tv_id = EXCLUDED.tv_id,
    season_id = EXCLUDED.season_id,
    name = EXCLUDED.name,
    overview = EXCLUDED.overview,
    episode_number = EXCLUDED.episode_number,
    season_number = EXCLUDED.season_number,
    air_date = EXCLUDED.air_date,
    still_path = EXCLUDED.still_path,
    vote_average = EXCLUDED.vote_average,
    vote_count = EXCLUDED.vote_count,
    runtime = EXCLUDED.runtime,
    updated_at = NOW();

-- name: GetEpisodeByID :one
SELECT * FROM tv_episodes WHERE id = $1;

-- name: ListEpisodesByTVShow :many
SELECT * FROM tv_episodes
WHERE tv_id = $1
ORDER BY season_number, episode_number;

-- name: ListEpisodesBySeason :many
SELECT * FROM tv_episodes
WHERE season_id = $1
ORDER BY episode_number;

-- name: DeleteEpisode :exec
DELETE FROM tv_episodes WHERE id = $1;
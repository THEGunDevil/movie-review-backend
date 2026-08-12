-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- 1. USERS
-- ============================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    profile_picture TEXT,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    is_banned BOOLEAN NOT NULL DEFAULT FALSE,
    ban_reason TEXT,
    ban_until TIMESTAMPTZ,
    is_permanent_ban BOOLEAN NOT NULL DEFAULT FALSE,
    token_version INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 2. MOVIES (TMDB Movie Schema)
-- ============================================================
CREATE TABLE movies (
    id BIGINT PRIMARY KEY,                 -- TMDB movie ID
    title TEXT NOT NULL,
    original_language VARCHAR(10) NOT NULL,
    original_title TEXT NOT NULL,
    overview TEXT,
    release_date DATE,
    popularity DOUBLE PRECISION DEFAULT 0,
    vote_average DOUBLE PRECISION DEFAULT 0,
    vote_count INTEGER DEFAULT 0,
    poster_path TEXT,
    backdrop_path TEXT,
    adult BOOLEAN DEFAULT FALSE,
    genre_ids INTEGER[] DEFAULT '{}',      -- PostgreSQL GIN array index support
    softcore BOOLEAN DEFAULT FALSE,
    video BOOLEAN DEFAULT FALSE,
    runtime INTEGER,                       -- Duration in minutes
    budget BIGINT DEFAULT 0,
    revenue BIGINT DEFAULT 0,
    homepage TEXT,
    imdb_id VARCHAR(20),
    status VARCHAR(50),                    -- Released, Rumored, Post Production, etc.
    tagline TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 3. GENRES
-- ============================================================
CREATE TABLE genres (
    id INTEGER PRIMARY KEY,                -- TMDB Genre ID
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 4. PERSONS (TMDB Cast / Crew Details)
-- ============================================================
CREATE TABLE persons (
    id BIGINT PRIMARY KEY,                 -- TMDB person ID
    name TEXT NOT NULL,
    original_name TEXT,
    profile_path TEXT,
    popularity DOUBLE PRECISION DEFAULT 0,
    known_for_department TEXT,
    gender INTEGER DEFAULT 0,              -- 0: Unspecified, 1: Female, 2: Male, 3: Non-binary
    biography TEXT,
    birthday DATE,
    deathday DATE,
    place_of_birth TEXT,
    adult BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 5. MOVIE_CREDITS
-- ============================================================
CREATE TABLE movie_credits (
    id BIGSERIAL PRIMARY KEY,
    credit_id VARCHAR(50) UNIQUE,          -- TMDB unique credit string identifier
    movie_id BIGINT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    person_id BIGINT NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    role TEXT NOT NULL,                    -- character name (cast) or job title (crew)
    type VARCHAR(10) NOT NULL CHECK (type IN ('cast', 'crew')),
    "order" INTEGER DEFAULT 0,            -- cast placement order
    department TEXT,                       -- crew department (e.g. Directing, Camera, Writing)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 6. MOVIE VIDEOS
-- ============================================================
CREATE TABLE movie_videos (
    id VARCHAR(50) PRIMARY KEY,            -- TMDB video string ID (e.g. "57a1518f9251412d2100010f")
    movie_id BIGINT NOT NULL REFERENCES movies(id) ON DELETE CASCADE,
    iso_639_1 VARCHAR(10),
    iso_3166_1 VARCHAR(10),
    name TEXT NOT NULL,
    key TEXT NOT NULL,                     -- YouTube video key
    site VARCHAR(50) NOT NULL,
    size INTEGER,                          -- 720, 1080, etc.
    type VARCHAR(50) NOT NULL,             -- Trailer, Teaser, Featurette, Clip, etc.
    official BOOLEAN DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 7. TV SHOWS
-- ============================================================
CREATE TABLE tv_shows (
    id BIGINT PRIMARY KEY,                 -- TMDB TV show ID
    name TEXT NOT NULL,
    original_name TEXT NOT NULL,
    overview TEXT,
    original_language VARCHAR(10) NOT NULL,
    origin_country VARCHAR(10)[],
    poster_path TEXT,
    backdrop_path TEXT,
    first_air_date DATE,
    last_air_date DATE,
    popularity DOUBLE PRECISION DEFAULT 0,
    vote_average DOUBLE PRECISION DEFAULT 0,
    vote_count INTEGER DEFAULT 0,
    adult BOOLEAN DEFAULT FALSE,
    in_production BOOLEAN DEFAULT FALSE,
    number_of_seasons INTEGER DEFAULT 0,
    number_of_episodes INTEGER DEFAULT 0,
    genre_ids INTEGER[] DEFAULT '{}',
    status VARCHAR(50),                    -- Returning Series, Ended, Canceled, etc.
    type VARCHAR(50),                      -- Scripted, Reality, Documentary, etc.
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- ============================================================
-- 8. TV SEASONS
-- ============================================================
CREATE TABLE tv_seasons (
    id BIGINT PRIMARY KEY,                 -- TMDB season ID
    tv_id BIGINT NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    season_number INTEGER NOT NULL,
    name TEXT NOT NULL,
    overview TEXT,
    air_date DATE,
    poster_path TEXT,
    episode_count INTEGER NOT NULL DEFAULT 0,
    vote_average NUMERIC(3,1) DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tv_id, season_number)
);

-- ============================================================
-- 9. TV EPISODES
-- ============================================================
CREATE TABLE tv_episodes (
    id BIGINT PRIMARY KEY,                 -- TMDB episode ID
    tv_id BIGINT NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    season_id BIGINT NOT NULL REFERENCES tv_seasons(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    overview TEXT,
    episode_number INTEGER NOT NULL,
    season_number INTEGER NOT NULL,
    air_date DATE,
    still_path TEXT,
    vote_average DOUBLE PRECISION DEFAULT 0,
    vote_count INTEGER DEFAULT 0,
    runtime INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS tv_credits (
    id BIGSERIAL PRIMARY KEY,
    tv_id BIGINT NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    person_id BIGINT NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    type VARCHAR(10) NOT NULL CHECK (type IN ('cast', 'crew')),
    "order" INTEGER DEFAULT 0,
    department TEXT,
    profile_path TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tv_credits_tv_id ON tv_credits(tv_id);
CREATE INDEX IF NOT EXISTS idx_tv_credits_person_id ON tv_credits(person_id);

-- 3. tv_videos table
CREATE TABLE IF NOT EXISTS tv_videos (
    id VARCHAR(50) PRIMARY KEY,
    tv_id BIGINT NOT NULL REFERENCES tv_shows(id) ON DELETE CASCADE,
    iso_639_1 VARCHAR(10),
    iso_3166_1 VARCHAR(10),
    name TEXT NOT NULL,
    key TEXT NOT NULL,
    site VARCHAR(50) NOT NULL,
    size INTEGER,
    type VARCHAR(50) NOT NULL,
    official BOOLEAN DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tv_videos_tv_id ON tv_videos(tv_id);

-- ============================================================
-- 10. REVIEWS (Movies & TV Shows support)
-- ============================================================
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    movie_id BIGINT REFERENCES movies(id) ON DELETE CASCADE,
    tv_id BIGINT REFERENCES tv_shows(id) ON DELETE CASCADE,
    rating DECIMAL(3,1) CHECK (rating >= 0 AND rating <= 10),
    content TEXT,
    contains_spoilers BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT review_target_check CHECK (
        (movie_id IS NOT NULL AND tv_id IS NULL) OR 
        (movie_id IS NULL AND tv_id IS NOT NULL)
    )
);

-- ============================================================
-- 11. WATCHLIST (Movies & TV Shows support)
-- ============================================================
CREATE TABLE user_watchlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    movie_id BIGINT REFERENCES movies(id) ON DELETE CASCADE,
    tv_id BIGINT REFERENCES tv_shows(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT watchlist_target_check CHECK (
        (movie_id IS NOT NULL AND tv_id IS NULL) OR 
        (movie_id IS NULL AND tv_id IS NOT NULL)
    )
);

-- ============================================================
-- 12. REFRESH TOKENS
-- ============================================================
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
-- +goose Up

-- Votes (up/down)
CREATE TABLE review_votes (
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    vote VARCHAR(4) NOT NULL CHECK (vote IN ('up', 'down')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (review_id, user_id)
);

-- Likes (heart)
CREATE TABLE review_likes (
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (review_id, user_id)
);

-- Comments
CREATE TABLE review_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Reports
CREATE TABLE review_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    read BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    event_id UUID REFERENCES webhook_events(id) ON DELETE SET NULL
);

CREATE INDEX idx_notifications_created_at ON notifications(created_at DESC);
-- Indexes
CREATE INDEX idx_review_votes_review_id ON review_votes(review_id);
CREATE INDEX idx_review_likes_review_id ON review_likes(review_id);
CREATE INDEX idx_review_comments_review_id ON review_comments(review_id);

-- ============================================================
-- INDEXES
-- ============================================================
-- Movies
CREATE INDEX idx_movies_popularity ON movies(popularity DESC);
CREATE INDEX idx_movies_vote_average ON movies(vote_average DESC);
CREATE INDEX idx_movies_release_date ON movies(release_date DESC);
CREATE INDEX idx_movies_genres ON movies USING GIN(genre_ids);

-- Credits
CREATE INDEX idx_credits_movie_id ON movie_credits(movie_id);
CREATE INDEX idx_credits_person_id ON movie_credits(person_id);
CREATE INDEX idx_credits_type ON movie_credits(type);
CREATE INDEX idx_credits_credit_id ON movie_credits(credit_id);

-- Genres & Persons
CREATE INDEX idx_genres_name ON genres(name);
CREATE INDEX idx_persons_name ON persons(name);

-- TV Shows & Seasons / Episodes
CREATE INDEX idx_tv_name ON tv_shows(name);
CREATE INDEX idx_tv_popularity ON tv_shows(popularity DESC);
CREATE INDEX idx_tv_rating ON tv_shows(vote_average DESC);
CREATE INDEX idx_tv_first_air_date ON tv_shows(first_air_date DESC);
CREATE INDEX idx_tv_genres ON tv_shows USING GIN(genre_ids);

CREATE INDEX idx_tv_seasons_tv_id ON tv_seasons(tv_id);
CREATE INDEX idx_tv_episodes_tv_id ON tv_episodes(tv_id);
CREATE INDEX idx_tv_episodes_season_id ON tv_episodes(season_id);

-- Videos
CREATE INDEX idx_movie_videos_movie_id ON movie_videos(movie_id);
CREATE INDEX idx_movie_videos_type ON movie_videos(type);

-- Reviews & Watchlist
CREATE INDEX idx_reviews_movie_id ON reviews(movie_id);
CREATE INDEX idx_reviews_tv_id ON reviews(tv_id);
CREATE INDEX idx_reviews_user_id ON reviews(user_id);

CREATE INDEX idx_watchlist_user_id ON user_watchlist(user_id);
CREATE INDEX idx_watchlist_movie_id ON user_watchlist(movie_id);
CREATE INDEX idx_watchlist_tv_id ON user_watchlist(tv_id);

-- Refresh tokens
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- +goose Down
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS user_watchlist CASCADE;
DROP TABLE IF EXISTS reviews CASCADE;
DROP TABLE IF EXISTS tv_episodes CASCADE;
DROP TABLE IF EXISTS tv_seasons CASCADE;
DROP TABLE IF EXISTS tv_shows CASCADE;
DROP TABLE IF EXISTS movie_videos CASCADE;
DROP TABLE IF EXISTS movie_credits CASCADE;
DROP TABLE IF EXISTS persons CASCADE;
DROP TABLE IF EXISTS genres CASCADE;
DROP TABLE IF EXISTS movies CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tv_videos CASCADE;
DROP TABLE IF EXISTS tv_credits CASCADE;
DROP TABLE IF EXISTS webhook_events  CASCADE;
DROP TABLE IF EXISTS notifications  CASCADE;
DROP TABLE IF EXISTS review_reports, review_comments, review_likes, review_votes;
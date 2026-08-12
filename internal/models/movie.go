package models

import (
	"time"

	gen "github.com/internal/db/gen"
)

// Genre represents the genres table
type Genre struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Movie represents the movies table
type Movie struct {
	ID               int64        `json:"id" db:"id"`
	Title            string       `json:"title" db:"title"`
	OriginalLanguage string       `json:"original_language" db:"original_language"`
	OriginalTitle    string       `json:"original_title" db:"original_title"`
	Overview         *string      `json:"overview" db:"overview"`         // nullable
	ReleaseDate      *string      `json:"release_date" db:"release_date"` // nullable (DATE can be NULL)
	Popularity       float64      `json:"popularity" db:"popularity"`
	VoteAverage      float64      `json:"vote_average" db:"vote_average"`
	VoteCount        int          `json:"vote_count" db:"vote_count"`
	PosterPath       *string      `json:"poster_path" db:"poster_path"`     // nullable
	BackdropPath     *string      `json:"backdrop_path" db:"backdrop_path"` // nullable
	Adult            bool         `json:"adult" db:"adult"`
	GenreIDs         []int64      `json:"genre_ids" db:"genre_ids"` // PostgreSQL integer array
	Softcore         bool         `json:"softcore" db:"softcore"`
	Video            bool         `json:"video" db:"video"`
	CreatedAt        time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at" db:"updated_at"`
	MovieVideo       []MovieVideo `json:"movie_trailers" db:"trailers"`
}

type MovieVideo struct {
	ID          string     `json:"id"`
	MovieID     int64      `json:"movie_id"`
	Iso639_1    *string    `json:"iso_639_1,omitempty"`
	Iso3166_1   *string    `json:"iso_3166_1,omitempty"`
	Name        string     `json:"name"`
	Key         string     `json:"key"`
	Site        string     `json:"site"`
	Size        *int       `json:"size,omitempty"`
	Type        string     `json:"type"`
	Official    bool       `json:"official"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

type CreditType string

const (
	CreditTypeCast CreditType = "cast"
	CreditTypeCrew CreditType = "crew"
)

type Credit struct {
	ID                int64       `json:"id"`
	MovieID           int         `json:"movie_id"`
	PersonID          int         `json:"person_id"`
	Role              string      `json:"role"`
	Type              CreditType  `json:"type"`
	Order             int         `json:"order"`
	Department        string `json:"department"`
	PersonName        string      `json:"person_name"`
	PersonProfilePath string `json:"person_profile_path"`
}

type Person struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	ProfilePath        *string   `json:"profile_path,omitempty"`
	Popularity         *float64  `json:"popularity,omitempty"`
	KnownForDepartment *string   `json:"known_for_department,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func ToMovieResponse(m gen.Movie) Movie {
	var releaseDate string
	if m.ReleaseDate.Valid {
		releaseDate = m.ReleaseDate.Time.Format("02-01-2006")
	}
	genreIDs := make([]int64, len(m.GenreIds))
	for i, v := range m.GenreIds {
		genreIDs[i] = int64(v)
	}
	return Movie{
		ID:               m.ID,
		Title:            m.Title,
		OriginalLanguage: m.OriginalLanguage,
		Overview:         &m.Overview.String,
		ReleaseDate:      &releaseDate,
		Popularity:       m.Popularity.Float64,
		VoteAverage:      m.VoteAverage.Float64,
		VoteCount:        int(m.VoteCount.Int32),
		PosterPath:       &m.PosterPath.String,
		BackdropPath:     &m.BackdropPath.String,
		Adult:            m.Adult.Bool,
		GenreIDs:         genreIDs,
		OriginalTitle:    m.OriginalTitle,
		Softcore:         m.Softcore.Bool,
		Video:            m.Video.Bool,
		CreatedAt:        m.CreatedAt.Time,
		UpdatedAt:        m.UpdatedAt.Time,
	}
}
func ToPersonResponse(p gen.Person) Person {
	return Person{
		ID:                 int(p.ID),
		Name:               p.Name,
		ProfilePath:        &p.ProfilePath.String,
		Popularity:         &p.Popularity.Float64,
		KnownForDepartment: &p.KnownForDepartment.String,
	}
}
func ToGenreResponse(m gen.Genre) Genre {
	return Genre{
		ID:        int(m.ID),
		Name:      m.Name,
		CreatedAt: m.CreatedAt.Time,
		UpdatedAt: m.UpdatedAt.Time,
	}
}
func ToVideoResponse(v gen.MovieVideo) MovieVideo {
	var size *int
	if v.Size.Valid {
		s := int(v.Size.Int32)
		size = &s
	}
	var publishedAt *time.Time
	if v.PublishedAt.Valid {
		publishedAt = &v.PublishedAt.Time
	}

	return MovieVideo{
		ID:          v.ID,
		MovieID:     v.MovieID,
		Iso639_1:    &v.Iso6391.String,
		Iso3166_1:   &v.Iso31661.String,
		Name:        v.Name,
		Key:         v.Key,
		Site:        v.Site,
		Size:        size,
		Type:        v.Type,
		Official:    v.Official.Bool,
		PublishedAt: publishedAt,
	}
}

// ToVideosResponse converts a slice of videos.
func ToVideosResponse(videos []gen.MovieVideo) []MovieVideo {
	result := make([]MovieVideo, len(videos))
	for i, v := range videos {
		result[i] = ToVideoResponse(v)
	}
	return result
}

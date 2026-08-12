package models

import (
	gen "github.com/internal/db/gen"
	"time"
)

type TVShow struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	OriginalName     string    `json:"original_name"`
	Overview         *string   `json:"overview"`
	OriginalLanguage string    `json:"original_language"`
	OriginCountry    []string  `json:"origin_country"`
	PosterPath       *string   `json:"poster_path"`
	BackdropPath     *string   `json:"backdrop_path"`
	FirstAirDate     *string   `json:"first_air_date"`
	LastAirDate      *string   `json:"last_air_date"`
	Popularity       float64   `json:"popularity"`
	VoteAverage      float64   `json:"vote_average"`
	VoteCount        int       `json:"vote_count"`
	Adult            bool      `json:"adult"`
	InProduction     bool      `json:"in_production"`
	NumberOfSeasons  int       `json:"number_of_seasons"`
	NumberOfEpisodes int       `json:"number_of_episodes"`
	GenreIDs         []int64   `json:"genre_ids"`
	Status           *string   `json:"status"`
	Type             *string   `json:"type"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func ToTVShowResponse(s gen.TvShow) TVShow {
	var firstAirDate, lastAirDate *string
	if s.FirstAirDate.Valid {
		date := s.FirstAirDate.Time.Format("2006-01-02")
		firstAirDate = &date
	}
	if s.LastAirDate.Valid {
		date := s.LastAirDate.Time.Format("2006-01-02")
		lastAirDate = &date
	}

	genreIDs := make([]int64, len(s.GenreIds))
	for i, v := range s.GenreIds {
		genreIDs[i] = int64(v)
	}

	return TVShow{
		ID:               s.ID,
		Name:             s.Name,
		OriginalName:     s.OriginalName,
		Overview:         &s.Overview.String,
		OriginalLanguage: s.OriginalLanguage,
		OriginCountry:    s.OriginCountry, // assuming []string
		PosterPath:       &s.PosterPath.String,
		BackdropPath:     &s.BackdropPath.String,
		FirstAirDate:     firstAirDate,
		LastAirDate:      lastAirDate,
		Popularity:       s.Popularity.Float64,
		VoteAverage:      s.VoteAverage.Float64,
		VoteCount:        int(s.VoteCount.Int32),
		Adult:            s.Adult.Bool,
		InProduction:     s.InProduction.Bool,
		NumberOfSeasons:  int(s.NumberOfSeasons.Int32),
		NumberOfEpisodes: int(s.NumberOfEpisodes.Int32),
		GenreIDs:         genreIDs,
		Status:           &s.Status.String,
		Type:             &s.Type.String,
		CreatedAt:        s.CreatedAt.Time,
		UpdatedAt:        s.UpdatedAt.Time,
	}
}

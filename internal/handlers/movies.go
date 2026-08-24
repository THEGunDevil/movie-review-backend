package handlers

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	gen "github.com/internal/db/gen"
	"github.com/internal/service"
	"github.com/jackc/pgx/v5"

	"github.com/gin-gonic/gin"
	"github.com/internal/models"
)

type MoviesHandler struct {
	Queries *gen.Queries
}

// ---------- Helper ----------

func Error(err error, query string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%v not found", query)
	}
	return err
}
func (h *MoviesHandler) GetMovieByID(c *gin.Context) {
	idStr := c.Param("id")
	movieID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}
	ctx := c.Request.Context()

	row, err := h.Queries.GetMovieByID(ctx, int64(movieID))
	Error(err, "movie")
	c.JSON(http.StatusOK, gin.H{
		"movies": models.ToMovieResponse(row),
	})

}
func (h *MoviesHandler) GetPersonByCreditID(c *gin.Context) {
	idStr := c.Param("id")
	creditID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}
	ctx := c.Request.Context()

	row, err := h.Queries.GetPersonByID(ctx, int64(creditID))
	Error(err, "person")
	c.JSON(http.StatusOK, gin.H{
		"person": models.ToPersonResponse(row),
	})

}
func (h *MoviesHandler) GetCastsByCreditID(c *gin.Context) {
	idStr := c.Param("id")
	creditID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}
	ctx := c.Request.Context()
	row, err := h.Queries.GetCastByMovieID(ctx, int64(creditID))
	Error(err, "casts")
	casts := make([]gen.GetCastByMovieIDRow, 0, len(row))
	c.JSON(http.StatusOK, gin.H{
		"casts": casts,
	})

}

func (h *MoviesHandler) GetCreditsByMovieID(c *gin.Context) {
	idStr := c.Param("id")
	movieID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}
	page := 1
	limit := 50

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	// 1️⃣ মোট মুভির সংখ্যা বের করো
	query := c.Query("type")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}
	totalCredits, err := h.Queries.CountCreditsByType(c.Request.Context(), gen.CountCreditsByTypeParams{
		MovieID: int64(movieID),
		Type:    query,
	})
	totalPages := int(math.Ceil(float64(totalCredits) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	ctx := c.Request.Context()
	row, err := h.Queries.GetCreditsByMovieID(ctx, gen.GetCreditsByMovieIDParams{
		MovieID: int64(movieID),
		Type:    query,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	Error(err, "credits")
	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"credits":     row,
	})

}
func (h *MoviesHandler) GetVideosByMovieID(c *gin.Context) {
	idStr := c.Param("id")

	movieID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid movie id",
		})
		return
	}

	page := 1
	limit := 10

	if p := c.Query("page"); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid page",
			})
			return
		}

		page = parsed
	}

	if l := c.Query("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid limit",
			})
			return
		}

		limit = parsed
	}

	videoType := c.Query("type")

	if videoType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "video type is required",
		})
		return
	}

	ctx := c.Request.Context()

	log.Printf("🔥 video type: %s", videoType)
	log.Printf("🔥 movie ID: %d", movieID)
	log.Printf("🔥 page: %d, limit: %d", page, limit)

	totalVideos, err := h.Queries.CountVideosByType(
		ctx,
		gen.CountVideosByTypeParams{
			MovieID: int64(movieID),
			Type:    videoType,
		},
	)

	if err != nil {
		log.Printf("❌ CountVideosByType error: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to count videos",
		})
		return
	}

	totalPages := int(math.Ceil(
		float64(totalVideos) / float64(limit),
	))

	if totalPages == 0 {
		totalPages = 1
	}

	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * limit

	videos, err := h.Queries.GetVideosByMovieID(
		ctx,
		gen.GetVideosByMovieIDParams{
			MovieID: int64(movieID),
			Type:    videoType,
			Limit:   int32(limit),
			Offset:  int32(offset),
		},
	)

	if err != nil {
		log.Printf("❌ GetVideosByMovieID error: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch videos",
		})
		return
	}

	log.Printf("🔥 total videos: %d", totalVideos)
	log.Printf("🔥 total pages: %d", totalPages)
	log.Printf("🔥 videos returned: %d", len(videos))

	if videos == nil {
		videos = make([]gen.MovieVideo, 0)
	}

	c.JSON(http.StatusOK, gin.H{
		"page":         page,
		"limit":        limit,
		"total_pages":  totalPages,
		"total_videos": totalVideos,
		"videos":       videos,
	})
}

// GetMoviesPaginated returns a paginated list of movies (popularity desc).
func (h *MoviesHandler) GetMoviesPaginated(c *gin.Context) {
	page := 1
	limit := 50

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	// 1️⃣ মোট মুভির সংখ্যা বের করো
	totalMovies, err := h.Queries.CountMovies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count movies"})
		return
	}

	// 2️⃣ মোট পেজ বের করো
	totalPages := int(math.Ceil(float64(totalMovies) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	// 3️⃣ নির্দিষ্ট পেজের মুভি আনো
	rows, err := h.Queries.ListMovies(c.Request.Context(), gen.ListMoviesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch movies"})
		return
	}

	result := make([]models.Movie, 0, len(rows))
	for _, m := range rows {
		result = append(result, models.ToMovieResponse(m))
	}

	// 4️⃣ পূর্ণাঙ্গ পেজিনেশন রেসপন্স পাঠাও
	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"movies":      result,
	})
}

// SearchMovies searches movies by title (ILIKE).
func (h *MoviesHandler) SearchMovies(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}

	page := 1
	limit := 50
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	rows, err := h.Queries.SearchMovies(c.Request.Context(), gen.SearchMoviesParams{
		Column1: service.StringToPGText(query),
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	result := make([]models.Movie, 0, len(rows))
	for _, m := range rows {
		result = append(result, models.ToMovieResponse(m))
	}

	c.JSON(http.StatusOK, gin.H{
		"query":  query,
		"page":   page,
		"limit":  limit,
		"movies": result,
	})
}

// ListMoviesByGenre returns movies filtered by a genre ID.
func (h *MoviesHandler) ListMoviesByGenre(c *gin.Context) {
	genreIDStr := c.Param("genre_id")
	genreID, err := strconv.Atoi(genreIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid genre id"})
		return
	}

	page := 1
	limit := 50 // আপনার ফ্রন্টএন্ড limit 300 পাঠাচ্ছে, তাই এটি সে অনুযায়ী অ্যাডজাস্ট হবে
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	totalMovies, err := h.Queries.CountMoviesByGenre(c.Request.Context(), []int32{int32(genreID)})
	if err != nil {
		// Count না পেলে ডিফল্ট 0 ধরতে পারেন অথবা এরর থ্রো করতে পারেন
		totalMovies = 0
	}

	// ২. Total Pages ক্যালকুলেশন
	totalPages := int(math.Ceil(float64(totalMovies) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	// ৩. মুভিগুলো ফেচ করা
	rows, err := h.Queries.ListMoviesByGenre(c.Request.Context(), gen.ListMoviesByGenreParams{
		Column1: []int32{int32(genreID)},
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch movies by genre"})
		return
	}

	result := make([]models.Movie, 0, len(rows))
	for _, m := range rows {
		result = append(result, models.ToMovieResponse(m))
	}

	// ৪. ফ্রন্টএন্ডের রিকোয়ারমেন্ট অনুযায়ী রেসপন্স পাঠানো
	c.JSON(http.StatusOK, gin.H{
		"genre_id":      genreID,
		"page":          page,
		"limit":         limit,
		"total_pages":   totalPages, // ফ্রন্টএন্ডের জন্য এটি সবচেয়ে জরুরি
		"total_results": totalMovies,
		"movies":        result,
	})
}

// GetMoviesByIDs returns multiple movies by their IDs (for batch requests).
func (h *MoviesHandler) GetMoviesByIDs(c *gin.Context) {
	idsStr := c.Query("ids") // comma-separated
	if idsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ids parameter is required"})
		return
	}

	idsParts := strings.Split(idsStr, ",")
	ids := make([]int64, 0, len(idsParts))
	for _, part := range idsParts {
		if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid ids provided"})
		return
	}

	rows, err := h.Queries.GetMoviesByIDs(c.Request.Context(), ids)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch movies"})
		return
	}

	result := make([]models.Movie, 0, len(rows))
	for _, m := range rows {
		result = append(result, models.ToMovieResponse(m))
	}

	c.JSON(http.StatusOK, gin.H{"movies": result})
}

// DeleteMovie removes a movie by ID (admin only).
func (h *MoviesHandler) DeleteMovie(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}

	err = h.Queries.DeleteMovie(c.Request.Context(), int64(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete movie"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "movie deleted"})
}
func (h *MoviesHandler) GetTopMovies(c *gin.Context) {
	rows,err := h.Queries.ListTopMovies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to top 3 movie"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": rows})
}
func (h *MoviesHandler) AllGenres(c *gin.Context) {

	genre, err := h.Queries.ListGenres(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete movie"})
		return
	}
	result := make([]models.Genre, 0, len(genre))
	for _, m := range genre {
		result = append(result, models.ToGenreResponse(m))
	}
	c.JSON(http.StatusOK, result)
}

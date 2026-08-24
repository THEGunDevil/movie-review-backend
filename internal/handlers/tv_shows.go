package handlers

import (
	"errors"
	"math"
	"math/big"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	gen "github.com/internal/db/gen"
	"github.com/internal/models"
	"github.com/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TVShowsHandler struct {
	Queries *gen.Queries
}

func (h *TVShowsHandler) GetTVShowsPaginated(c *gin.Context) {
	page, limit := 1, 50
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

	total, _ := h.Queries.CountTVShows(c.Request.Context())
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	rows, err := h.Queries.ListTVShows(c.Request.Context(), gen.ListTVShowsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tv shows"})
		return
	}

	// Convert to response model (you'll need a ToTVShowResponse function)
	result := make([]models.TVShow, 0, len(rows))
	for _, m := range rows {
		result = append(result, models.ToTVShowResponse(m))
	}

	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"tv_shows":    result,
	})
}

// SearchTVShows searches TV shows by name.
func (h *TVShowsHandler) SearchTVShows(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}
	page, limit := 1, 50
	if p := c.Query("page"); p != "" {
		if parsed, _ := strconv.Atoi(p); parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, _ := strconv.Atoi(l); parsed > 0 {
			limit = parsed
		}
	}
	offset := (page - 1) * limit

	rows, err := h.Queries.SearchTVShows(c.Request.Context(), gen.SearchTVShowsParams{
		Column1: service.StringToPGText(query),
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil { /* handle */
	}

	result := make([]models.TVShow, 0, len(rows))
	for _, m := range rows {
		result = append(result, models.ToTVShowResponse(m))
	}

	c.JSON(http.StatusOK, gin.H{"query": query, "page": page, "limit": limit, "tv_shows": result})
}

// ListTVShowsByGenre returns TV shows filtered by genre.
func (h *TVShowsHandler) ListTVShowsByGenre(c *gin.Context) {
	genreID, err := strconv.Atoi(c.Param("genre_id"))
	if err != nil { /* 400 */
	}

	page, limit := 1, 50
	// parse page/limit similar to above
	offset := (page - 1) * limit

	genreArr := []int32{int32(genreID)}
	total, _ := h.Queries.CountTVShowsByGenre(c.Request.Context(), genreArr)
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	rows, err := h.Queries.ListTVShowsByGenre(c.Request.Context(), gen.ListTVShowsByGenreParams{
		Column1: genreArr,
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil { /* handle */
	}

	result := make([]models.TVShow, 0, len(rows))
	for _, m := range rows {
		result = append(result, models.ToTVShowResponse(m))
	}

	c.JSON(http.StatusOK, gin.H{
		"genre_id":    genreID,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"tv_shows":    result,
	})
}

// GetTVShowByID returns a single TV show.
func (h *TVShowsHandler) GetTVShowByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tv show id"})
		return
	}

	show, err := h.Queries.GetTVShowByID(c.Request.Context(), int64(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tv show not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tv show"})
		return
	}

	// Convert the database row to the API response model
	// (you need to create a ToTVShowResponse function similar to ToMovieResponse)
	c.JSON(http.StatusOK, models.ToTVShowResponse(show))
}

// GetTVPersonByCreditID is placeholder – you'd join through tv_credits.
func (h *TVShowsHandler) GetTVPersonByCreditID(c *gin.Context) {
	idStr := c.Param("id")
	creditID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credit id"})
		return
	}

	person, err := h.Queries.GetPersonByTVCreditID(c.Request.Context(), int64(creditID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "person not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch person"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"person": models.ToPersonResponse(person)})
}

// GetTVCreditsByShowID returns paginated credits for a TV show.
func (h *TVShowsHandler) GetTVCreditsByShowID(c *gin.Context) {
	idStr := c.Param("id")
	showID, err := strconv.Atoi(idStr)
	if err != nil { /* 400 */
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

	total, _ := h.Queries.CountTVCreditsByType(c.Request.Context(), gen.CountTVCreditsByTypeParams{
		TvID: int64(showID),
		Type: query,
	})
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	rows, err := h.Queries.GetTVCreditsByShowID(c.Request.Context(), gen.GetTVCreditsByShowIDParams{
		TvID:   int64(showID),
		Type:   query,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	Error(err, "credits")
	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"credits":     rows,
	})
}

// GetTVVideosByShowID returns paginated videos for a TV show.
func (h *TVShowsHandler) GetTVVideosByShowID(c *gin.Context) {
	idStr := c.Param("id")
	tvID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tv show id"})
		return
	}

	page := 1
	limit := 10
	if p := c.Query("page"); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page"})
			return
		}
		page = parsed
	}
	if l := c.Query("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		limit = parsed
	}

	videoType := c.Query("type")
	if videoType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "video type is required"})
		return
	}

	ctx := c.Request.Context()

	totalVideos, err := h.Queries.CountTVVideosByType(ctx, gen.CountTVVideosByTypeParams{
		TvID: int64(tvID),
		Type: videoType,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count videos"})
		return
	}

	totalPages := int(math.Ceil(float64(totalVideos) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * limit

	videos, err := h.Queries.GetTVVideosByShowID(ctx, gen.GetTVVideosByShowIDParams{
		TvID:   int64(tvID),
		Type:   videoType,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch videos"})
		return
	}

	if videos == nil {
		videos = make([]gen.TvVideo, 0) // use generated type for tv_videos
	}

	c.JSON(http.StatusOK, gin.H{
		"page":         page,
		"limit":        limit,
		"total_pages":  totalPages,
		"total_videos": totalVideos,
		"videos":       videos,
	})
}

// DeleteTVShow deletes a TV show by ID.
func (h *TVShowsHandler) DeleteTVShow(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { /* 400 */
	}

	if err := h.Queries.DeleteTVShow(c.Request.Context(), int64(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "tv show deleted"})
}
func (h *MoviesHandler) GetTVReviewsByShow(c *gin.Context) {
	idStr := c.Param("id")
	tvID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tv show id"})
		return
	}

	page, limit := 1, 20
	if p := c.Query("page"); p != "" {
		if parsed, _ := strconv.Atoi(p); parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, _ := strconv.Atoi(l); parsed > 0 {
			limit = parsed
		}
	}
	offset := (page - 1) * limit

	total, _ := h.Queries.CountTVReviews(c.Request.Context(), pgtype.Int8{Int64: int64(tvID), Valid: true})
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	rows, err := h.Queries.GetTVReviewsByShow(c.Request.Context(), gen.GetTVReviewsByShowParams{
		TvID:   pgtype.Int8{Int64: int64(tvID), Valid: true},
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch reviews"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews":     rows,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"total":       total,
	})
}
func (h *TVShowsHandler) CreateTVReview(c *gin.Context) {
	idStr := c.Param("id")
	tvID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tv show id"})
		return
	}

	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req struct {
		Rating           float64 `json:"rating" binding:"required,min=0,max=10"`
		Content          string  `json:"content" binding:"required"`
		ContainsSpoilers bool    `json:"contains_spoilers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ratingNumeric := pgtype.Numeric{
		Int:   big.NewInt(int64(req.Rating * 10)),
		Exp:   -1,
		Valid: true,
	}

	review, err := h.Queries.CreateTVReview(c.Request.Context(), gen.CreateTVReviewParams{
		UserID:           service.UUIDToPGType(userID),
		TvID:             pgtype.Int8{Int64: int64(tvID), Valid: true},
		Rating:           ratingNumeric,
		Content:          service.StringToPGText(req.Content),
		ContainsSpoilers: pgtype.Bool{Bool: req.ContainsSpoilers, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"review": review})
}

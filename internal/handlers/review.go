package handlers

import (
	"database/sql"
	"errors"
	"math"
	"math/big"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
)

func parseReviewID(idStr string) (pgtype.UUID, error) {
	uid, err := uuid.Parse(idStr)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{
		Bytes: uid,
		Valid: true,
	}, nil
}
func (h *MoviesHandler) CreateReview(c *gin.Context) {
	// 1. Get movie ID from URL
	idStr := c.Param("id")
	movieID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}

	// 2. Get user ID from authenticated context
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// 3. Bind request body
	var req struct {
		Rating           float64 `json:"rating" binding:"required,min=0,max=10"`
		Content          string  `json:"content" binding:"required"`
		ContainsSpoilers bool    `json:"contains_spoilers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Insert review into database
	review, err := h.Queries.CreateReview(c.Request.Context(), gen.CreateReviewParams{
		UserID:           service.UUIDToPGType(userID),
		MovieID:          pgtype.Int8{Int64: int64(movieID), Valid: true},
		Rating:           pgtype.Numeric{Int: big.NewInt(int64(req.Rating * 10)), Exp: -1, Valid: true},
		Content:          service.StringToPGText(req.Content),
		ContainsSpoilers: pgtype.Bool{Bool: req.ContainsSpoilers, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create review"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"review": review})
}

func (h *MoviesHandler) GetReviewsByMovie(c *gin.Context) {
	idStr := c.Param("id")
	movieID, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie id"})
		return
	}

	page := 1
	limit := 20
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

	ctx := c.Request.Context()

	// Total count for pagination
	total, err := h.Queries.CountReviewsByMovieID(ctx, pgtype.Int8{Int64: int64(movieID), Valid: true})
	if err != nil {
		total = 0
	}

	// Fetch reviews
	reviews, err := h.Queries.GetReviewsByMovieID(ctx, gen.GetReviewsByMovieIDParams{
		MovieID: pgtype.Int8{Int64: int64(movieID), Valid: true},
		Limit:   int32(limit),
		Offset:  int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch reviews"})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews":     reviews,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}
func (h *MoviesHandler) UpdateReview(c *gin.Context) {
	reviewID, err := parseReviewID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}

	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req struct {
		Rating           *float64 `json:"rating" binding:"omitempty,min=0,max=10"`
		Content          *string  `json:"content" binding:"omitempty"`
		ContainsSpoilers *bool    `json:"contains_spoilers"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch current review to get existing values for fields not sent
	currentReview, err := h.Queries.GetReviewByID(c.Request.Context(), reviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "review not found"})
		return
	}

	rating := currentReview.Rating
	content := currentReview.Content
	spoilers := currentReview.ContainsSpoilers.Bool

	if req.Rating != nil {
		rating = pgtype.Numeric{
			Int:   big.NewInt(int64(*req.Rating * 10)),
			Exp:   -1,
			Valid: true,
		}
	}
	if req.Content != nil {
		content = service.StringToPGText(*req.Content)
	}
	if req.ContainsSpoilers != nil {
		spoilers = *req.ContainsSpoilers
	}

	err = h.Queries.UpdateReview(c.Request.Context(), gen.UpdateReviewParams{
		ID:               reviewID,
		Rating:           rating,
		Content:          content,
		ContainsSpoilers: pgtype.Bool{Bool: spoilers, Valid: true},
		UserID:           service.UUIDToPGType(userID),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	// Fetch the updated review to return complete data
	updatedReview, err := h.Queries.GetReviewByID(c.Request.Context(), reviewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "review updated but could not fetch"})
		return
	}

	c.JSON(http.StatusOK, updatedReview)
}

// ──── Delete Review (only owner) ───────────────────────────────
func (h *MoviesHandler) DeleteReview(c *gin.Context) {
	reviewID, err := parseReviewID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid review id",
		})
		return
	}

	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "authentication required",
		})
		return
	}

	err = h.Queries.DeleteReview(
		c.Request.Context(),
		gen.DeleteReviewParams{
			ID:     reviewID,
			UserID: service.UUIDToPGType(userID),
		},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "review not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete review",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "review deleted successfully",
	})
}
func (h *MoviesHandler) GetReviewByID(c *gin.Context) {
	// ✅ Parse the UUID string (not int)
	reviewUUID, err := parseReviewID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid review id"})
		return
	}

	review, err := h.Queries.GetReviewByID(c.Request.Context(), reviewUUID)
	if err != nil {
		Error(err, "review")
	}
	c.JSON(http.StatusOK, review)
}
func (h *MoviesHandler) GetReviewsByUserID(c *gin.Context) {
    // Parse target user ID (UUID)
    userID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
        return
    }

    // Parse pagination
    page := 1
    limit := 20
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

    // Get current logged-in user ID from context (set by your auth middleware)
    currentUserID, _ := c.Get("user_id")
    var currentUUID uuid.UUID
    if currentUserID != nil {
        currentUUID = currentUserID.(uuid.UUID)
    } else {
        currentUUID = uuid.Nil // or use pgtype.UUID{Valid:false} for NULL
    }

    ctx := c.Request.Context()

    // Total count
    total, err := h.Queries.CountReviewsByUserID(ctx, service.UUIDToPGType(userID))
    if err != nil {
        total = 0
    }
    totalPages := int(math.Ceil(float64(total) / float64(limit)))
    if totalPages == 0 {
        totalPages = 1
    }

    // Fetch reviews with full details
    reviews, err := h.Queries.ListReviewsByUserID(ctx, gen.ListReviewsByUserIDParams{
        UserID:        service.UUIDToPGType(userID),
        UserID_2: service.UUIDToPGType(currentUUID),
        Limit:         int32(limit),
        Offset:        int32(offset),
    })
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch reviews"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "reviews":     reviews,
        "total":       total,
        "page":        page,
        "limit":       limit,
        "total_pages": totalPages,
    })
}

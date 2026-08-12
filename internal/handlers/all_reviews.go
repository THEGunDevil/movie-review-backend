package handlers

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ReviewsHandler struct {
	Queries *gen.Queries
}

// POST /reviews/:id/vote  (body: {"vote":"up"})
func (rh *ReviewsHandler) VoteOnReview(c *gin.Context) {
	reviewID := c.Param("id")
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var req struct {
		Vote string `json:"vote" binding:"required,oneof=up down"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := rh.Queries.VoteOnReview(c.Request.Context(), gen.VoteOnReviewParams{
		ReviewID: service.UUIDToPGType(uuid.MustParse(reviewID)),
		UserID:   service.UUIDToPGType(userID),
		Vote:     req.Vote,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to vote"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DELETE /reviews/:id/vote
func (rh *ReviewsHandler) RemoveVote(c *gin.Context) {
	reviewID := c.Param("id")
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	err := rh.Queries.RemoveVote(c.Request.Context(), gen.RemoveVoteParams{
		ReviewID: service.UUIDToPGType(uuid.MustParse(reviewID)),
		UserID:   service.UUIDToPGType(userID),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove vote"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /reviews/:id/like
func (rh *ReviewsHandler) LikeReview(c *gin.Context) {
	reviewID := c.Param("id")
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	err := rh.Queries.LikeReview(c.Request.Context(), gen.LikeReviewParams{
		ReviewID: service.UUIDToPGType(uuid.MustParse(reviewID)),
		UserID:   service.UUIDToPGType(userID),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to like"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DELETE /reviews/:id/like
func (rh *ReviewsHandler) UnlikeReview(c *gin.Context) {
	reviewID := c.Param("id")
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	err := rh.Queries.UnlikeReview(c.Request.Context(), gen.UnlikeReviewParams{
		ReviewID: service.UUIDToPGType(uuid.MustParse(reviewID)),
		UserID:   service.UUIDToPGType(userID),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unlike"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /reviews/:id/report  (body: {"reason":"spam"})
func (rh *ReviewsHandler) ReportReview(c *gin.Context) {
	reviewID := c.Param("id")
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := rh.Queries.ReportReview(c.Request.Context(), gen.ReportReviewParams{
		ReviewID: service.UUIDToPGType(uuid.MustParse(reviewID)),
		UserID:   service.UUIDToPGType(userID),
		Reason:   req.Reason,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to report"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /reviews/:id/comments  (body: {"content":"..."})
func (rh *ReviewsHandler) AddComment(c *gin.Context) {
	reviewID := c.Param("id")
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	comment, err := rh.Queries.AddComment(c.Request.Context(), gen.AddCommentParams{
		ReviewID: service.UUIDToPGType(uuid.MustParse(reviewID)),
		UserID:   service.UUIDToPGType(userID),
		Content:  req.Content,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add comment"})
		return
	}
	c.JSON(http.StatusCreated, comment)
}

// GET /reviews/:id/comments
func (rh *ReviewsHandler) GetComments(c *gin.Context) {
	reviewID := c.Param("id")
	comments, err := rh.Queries.GetCommentsByReview(c.Request.Context(), service.UUIDToPGType(uuid.MustParse(reviewID)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"comments": comments, "total": len(comments)})
}
func (rh *ReviewsHandler) GetAllReviews(c *gin.Context) {
	// Parse pagination
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

	// Optional: get current user for personalised data (vote, like)
	var userID uuid.UUID
	if uid, ok := service.UserIDFromContext(c); ok {
		userID = uid
	} // else zero UUID → no user-specific data

	rows, err := rh.Queries.GetAllReviewsWithUser(c.Request.Context(), gen.GetAllReviewsWithUserParams{
		UserID: service.UUIDToPGType(userID),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch reviews"})
		return
	}

	// Total count (you can keep a separate CountAllReviews query or just use the length if full scan is ok)
	total, _ := rh.Queries.CountAllReviews(c.Request.Context())
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	c.JSON(http.StatusOK, gin.H{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
		"reviews":     rows,
	})
}

// GET /api/top-review
func (rh *ReviewsHandler) GetTopReviewHandler(c *gin.Context) {
	period := c.DefaultQuery("period", "week")

	var start, end time.Time
	now := time.Now()
	if period == "month" {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, -1).Add(24*time.Hour - time.Second)
	} else {
		end = now
		start = now.AddDate(0, 0, -7)
	}

	top, err := rh.Queries.GetTopRatedMediaByPeriod(c.Request.Context(), gen.GetTopRatedMediaByPeriodParams{
		CreatedAt: pgtype.Timestamptz{Time: start, Valid: true},
		CreatedAt_2:   pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{"data": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch top review"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": map[string]interface{}{
			"media_id":     top.MediaID,
			"media_title":  top.MediaTitle,
			"media_type":   top.MediaType,
			"poster_path":  top.PosterPath,
			"avg_rating":   top.AvgRating,
			"review_count": top.ReviewCount,
			"top_review":   top.TopReview,
			"user_name":    top.UserName,
			"user_id":      top.UserID,
			"created_at":   top.CreatedAt,
			"genres":       top.Genres,
		},
	})
}

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	Hub     *NotificationHub
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

	// ── রিসপন্স আগেই পাঠিয়ে দিন ────────────────
	c.JSON(http.StatusCreated, comment)

	// ── ব্যাকগ্রাউন্ডে নোটিফিকেশন পাঠান (goroutine) ─────
	go rh.sendNotificationForComment(reviewID, userID, comment.Content)
}

func (rh *ReviewsHandler) sendNotificationForComment(
	reviewID string,
	commenterID uuid.UUID,
	commentContent string,
) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	reviewUUID, err := uuid.Parse(reviewID)
	if err != nil {
		log.Printf("Invalid review ID: %v", err)
		return
	}

	// Review বের করি
	review, err := rh.Queries.GetReviewByID(
		ctx,
		service.UUIDToPGType(reviewUUID),
	)

	if err != nil {
		log.Printf("Failed to get review for notification: %v", err)
		return
	}

	// নিজের review-এ নিজে comment করলে notification নয়
	reviewOwnerID := uuid.UUID(review.UserID.Bytes)

	if reviewOwnerID == commenterID {
		return
	}

	// Commenter
	commenter, err := rh.Queries.GetUserByID(
		ctx,
		service.UUIDToPGType(commenterID),
	)

	if err != nil {
		log.Printf("Failed to get commenter: %v", err)
		return
	}

	// Media title
	var mediaTitle string

	if review.MovieID.Valid {
		movie, err := rh.Queries.GetMovieByID(
			ctx,
			review.MovieID.Int64,
		)

		if err == nil {
			mediaTitle = movie.Title
		} else {
			mediaTitle = "your movie review"
		}

	} else if review.TvID.Valid {
		tv, err := rh.Queries.GetTVShowByID(
			ctx,
			review.TvID.Int64,
		)

		if err == nil {
			mediaTitle = tv.Name
		} else {
			mediaTitle = "your TV review"
		}
	} else {
		mediaTitle = "your review"
	}

	title := "New Comment"

	message := fmt.Sprintf(
		"%s commented on %s",
		commenter.UserName,
		mediaTitle,
	)

	payload := map[string]interface{}{
		"type":         "comment",
		"link":         fmt.Sprintf("/reviews/%s", reviewID),
		"avatar":       commenter.ProfilePicture.String,
		"comment":      commentContent,
		"commenter_id": commenterID.String(),
		"review_id":    reviewID,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal notification payload: %v", err)
		return
	}

	log.Printf("Notification payload: %s", string(payloadBytes))

	createdNotification, err := rh.Queries.CreateNotificationWithEvent(
		ctx,
		gen.CreateNotificationWithEventParams{
			EventType:   "comment",
			Payload:     string(payloadBytes),
			RecipientID: service.UUIDToPGType(reviewOwnerID),
			Title:       title,
			Message:     message,
		},
	)

	if err != nil {
		log.Printf(
			"Failed to create notification: %v",
			err,
		)

		return
	}

	notificationID := uuid.UUID(
		createdNotification.ID.Bytes,
	)

	// ========================================
	// SSE event
	// ========================================

	event := NotificationEvent{
		ID:        notificationID.String(),
		Type:      "comment",
		Title:     title,
		Message:   message,
		CreatedAt: time.Now().Format(time.RFC3339),
		Read:      false,
		Avatar:    commenter.ProfilePicture.String,
		Link:      fmt.Sprintf("/reviews/%s", reviewID),
	}

	// একই user-এর connected SSE clients-কে পাঠাও
	rh.Hub.Publish(
		reviewOwnerID,
		event,
	)
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
	ctx := c.Request.Context()

	// -----------------------------
	// Pagination
	// -----------------------------
	page := 1
	limit := 20

	if value := c.Query("page"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if value := c.Query("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	// -----------------------------
	// Filters
	// -----------------------------
	search := strings.TrimSpace(c.Query("search"))

	mediaType := c.Query("media_type")
	if mediaType == "" {
		mediaType = "all"
	}

	switch mediaType {
	case "all", "movie", "tv":
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid media_type",
		})
		return
	}

	sort := c.Query("sort")
	if sort == "" {
		sort = "newest"
	}

	switch sort {
	case "newest", "popular", "discussed", "highest_rated":
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid sort",
		})
		return
	}

	// -----------------------------
	// Rating
	// -----------------------------
	var minRating pgtype.Numeric

	if value := c.Query("min_rating"); value != "" {
		if err := minRating.Scan(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid min_rating",
			})
			return
		}
	}

	// -----------------------------
	// Current user
	// -----------------------------
	var userID pgtype.UUID

	if uid, ok := service.UserIDFromContext(c); ok {
		userID = service.UUIDToPGType(uid)
	}

	// -----------------------------
	// Get reviews
	// -----------------------------
	rows, err := rh.Queries.GetAllReviews(
		ctx,
		gen.GetAllReviewsParams{
			UserID:      userID,
			MediaType:   mediaType,
			Search:      search,
			MinRating:   minRating,
			Sort:        sort,
			LimitCount:  int32(limit),
			OffsetCount: int32(offset),
		},
	)

	if err != nil {
		log.Printf("GetAllReviews error: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch reviews",
		})
		return
	}

	// -----------------------------
	// Count
	// -----------------------------
	total, err := rh.Queries.CountAllReviews(
		ctx,
		gen.CountAllReviewsParams{
			MediaType: mediaType,
			Search:    search,
			MinRating: minRating,
		},
	)

	if err != nil {
		log.Printf("CountAllReviews error: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to count reviews",
		})
		return
	}

	// -----------------------------
	// Total pages
	// -----------------------------
	totalPages := int(
		(total + int64(limit) - 1) / int64(limit),
	)

	if totalPages == 0 {
		totalPages = 1
	}

	// -----------------------------
	// Response
	// -----------------------------
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
		CreatedAt:   pgtype.Timestamptz{Time: start, Valid: true},
		CreatedAt_2: pgtype.Timestamptz{Time: end, Valid: true},
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

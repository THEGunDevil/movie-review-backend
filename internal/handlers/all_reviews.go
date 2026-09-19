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

	// 🔔 Notification Trigger
	rh.sendNotificationForAction(reviewID, userID, "vote", req.Vote)

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

	// 🔔 Notification Trigger
	rh.sendNotificationForAction(reviewID, userID, "like", "")

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

	// 🔔 Notification Trigger
	rh.sendNotificationForAction(reviewID, userID, "report", req.Reason)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// POST /reviews/:id/comments
func (rh *ReviewsHandler) AddComment(c *gin.Context) {
	reviewID := c.Param("id")
	reviewUUID, err := uuid.Parse(reviewID)
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

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	comment, err := rh.Queries.AddComment(
		c.Request.Context(),
		gen.AddCommentParams{
			ReviewID: service.UUIDToPGType(reviewUUID),
			UserID:   service.UUIDToPGType(userID),
			Content:  req.Content,
		},
	)

	if err != nil {
		log.Printf("❌ AddComment failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to add comment",
		})
		return
	}

	log.Printf("✅ Comment created: review=%s commenter=%s", reviewID, userID.String())

	// 🔔 Notification Trigger
	rh.sendNotificationForAction(reviewID, userID, "comment", comment.Content)

	c.JSON(http.StatusCreated, comment)
}

// Helper to format, build payload, save, and stream notifications via SSE
func (rh *ReviewsHandler) sendNotificationForAction(
	reviewID string,
	actorID uuid.UUID,
	eventType string, // "comment", "like", "vote", "report"
	detail string, // content for comment, vote type (up/down) for vote, reason for report
) {
	log.Printf("🔔 Notification started: type=%s review=%s actor=%s", eventType, reviewID, actorID.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reviewUUID, err := uuid.Parse(reviewID)
	if err != nil {
		log.Printf("❌ Invalid review UUID: %v", err)
		return
	}

	review, err := rh.Queries.GetReviewByID(ctx, service.UUIDToPGType(reviewUUID))
	if err != nil {
		log.Printf("❌ GetReviewByID failed: %v", err)
		return
	}

	if !review.UserID.Valid {
		log.Printf("❌ Review UserID is invalid")
		return
	}

	reviewOwnerID := uuid.UUID(review.UserID.Bytes)

	// Don't notify if user is performing actions on their own review
	if reviewOwnerID == actorID {
		log.Printf("ℹ️ Actor is owner of the review. Skipping notification.")
		return
	}

	actor, err := rh.Queries.GetUserByID(ctx, service.UUIDToPGType(actorID))
	if err != nil {
		log.Printf("❌ GetUserByID failed: %v", err)
		return
	}

	// Resolve Media Title
	mediaTitle := "your review"
	if review.MovieID.Valid {
		if movie, err := rh.Queries.GetMovieByID(ctx, review.MovieID.Int64); err == nil {
			mediaTitle = movie.Title
		}
	} else if review.TvID.Valid {
		if tv, err := rh.Queries.GetTVShowByID(ctx, review.TvID.Int64); err == nil {
			mediaTitle = tv.Name
		}
	}

	// Resolve titles and messages based on eventType
	var title, message string
	payload := map[string]interface{}{
		"type":       eventType,
		"link":       fmt.Sprintf("/reviews/%s", reviewID),
		"avatar":     actor.ProfilePicture.String,
		"actor_id":   actorID.String(),
		"review_id":  reviewID,
		"actor_name": actor.UserName,
	}

	switch eventType {
	case "comment":
		title = "New Comment"
		message = fmt.Sprintf("%s commented on %s", actor.UserName, mediaTitle)
		payload["comment"] = detail

	case "like":
		title = "New Like"
		message = fmt.Sprintf("%s liked your review on %s", actor.UserName, mediaTitle)

	case "vote":
		title = "New Vote"
		message = fmt.Sprintf("%s voted %s on your review for %s", actor.UserName, detail, mediaTitle)
		payload["vote"] = detail

	case "report":
		title = "Review Reported"
		message = fmt.Sprintf("Your review for %s was reported", mediaTitle)
		payload["reason"] = detail
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ JSON marshal failed: %v", err)
		return
	}

	// Save to DB
	createdNotification, err := rh.Queries.CreateNotificationWithEvent(
		ctx,
		gen.CreateNotificationWithEventParams{
			EventType:   eventType,
			Payload:     string(payloadBytes),
			RecipientID: service.UUIDToPGType(reviewOwnerID),
			Title:       title,
			Message:     message,
		},
	)

	if err != nil {
		log.Printf("❌ CreateNotificationWithEvent failed: %v", err)
		return
	}

	notificationID := uuid.UUID(createdNotification.ID.Bytes)

	// Publish via SSE
	event := NotificationEvent{
		ID:        notificationID.String(),
		Type:      eventType,
		Title:     title,
		Message:   message,
		CreatedAt: time.Now().Format(time.RFC3339),
		Read:      false,
		Avatar:    actor.ProfilePicture.String,
		Link:      fmt.Sprintf("/reviews/%s", reviewID),
	}

	if rh.Hub == nil {
		log.Printf("❌ NotificationHub is nil")
		return
	}

	rh.Hub.Publish(reviewOwnerID, event)
	log.Printf("✅ Notification sent successfully to user=%s", reviewOwnerID.String())
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

	search := strings.TrimSpace(c.Query("search"))

	mediaType := c.Query("media_type")
	if mediaType == "" {
		mediaType = "all"
	}

	switch mediaType {
	case "all", "movie", "tv":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media_type"})
		return
	}

	sort := c.Query("sort")
	if sort == "" {
		sort = "newest"
	}

	switch sort {
	case "newest", "popular", "discussed", "highest_rated":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sort"})
		return
	}

	var minRating pgtype.Numeric

	if value := c.Query("min_rating"); value != "" {
		if err := minRating.Scan(value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid min_rating"})
			return
		}
	}

	var userID pgtype.UUID

	if uid, ok := service.UserIDFromContext(c); ok {
		userID = service.UUIDToPGType(uid)
	}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch reviews"})
		return
	}

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count reviews"})
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
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

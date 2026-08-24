package handlers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
)

type WatchlistHandler struct {
	Queries *gen.Queries
}

func (wl *WatchlistHandler) GetWatchlistByUserID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
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

	total, err := wl.Queries.CountWatchlistByUserID(ctx, service.UUIDToPGType(id))
	if err != nil {
		// If count fails, set total to 0 (or return error)
		total = 0
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	watchlist, err := wl.Queries.ListWatchlistByUserID(ctx, gen.ListWatchlistByUserIDParams{
		UserID: service.UUIDToPGType(id),
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"watchlist":   watchlist,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
	})
}

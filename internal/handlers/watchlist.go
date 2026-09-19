package handlers

import (
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	gen "github.com/internal/db/gen"
	"github.com/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
func (wl *WatchlistHandler) GetMyWatchlist(c *gin.Context) {
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
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

	total, err := wl.Queries.CountWatchlistByUserID(ctx, service.UUIDToPGType(userID))
	if err != nil {
		// If count fails, set total to 0 (or return error)
		total = 0
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	watchlist, err := wl.Queries.ListWatchlistByUserID(ctx, gen.ListWatchlistByUserIDParams{
		UserID: service.UUIDToPGType(userID),
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

type AddWatchlistRequest struct {
	MovieID *int64 `json:"movie_id"`
	TVID    *int64 `json:"tv_id"`
}

func (wl *WatchlistHandler) AddToWatchlist(c *gin.Context) {
	// --------------------------------------------------
	// Get authenticated user
	// --------------------------------------------------
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})
		return
	}

	// --------------------------------------------------
	// Parse request body
	// --------------------------------------------------
	var req AddWatchlistRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// --------------------------------------------------
	// Exactly one target must be provided
	// --------------------------------------------------
	if (req.MovieID == nil && req.TVID == nil) ||
		(req.MovieID != nil && req.TVID != nil) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "provide either movie_id or tv_id",
		})
		return
	}

	ctx := c.Request.Context()
	dbUserID := service.UUIDToPGType(userID)

	// --------------------------------------------------
	// Movie
	// --------------------------------------------------
	if req.MovieID != nil {
		movieID := pgtype.Int8{
			Int64: *req.MovieID,
			Valid: true,
		}

		// Optional: make sure movie exists
		_, err := wl.Queries.GetMovieByID(ctx, *req.MovieID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "movie not found",
			})
			return
		}

		// Check duplicate
		exists, err := wl.Queries.IsMovieInWatchlist(
			ctx,
			gen.IsMovieInWatchlistParams{
				UserID:  dbUserID,
				MovieID: movieID,
			},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to check watchlist",
			})
			return
		}

		if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error": "movie is already in your watchlist",
			})
			return
		}

		item, err := wl.Queries.AddMovieToWatchlist(
			ctx,
			gen.AddMovieToWatchlistParams{
				UserID:  dbUserID,
				MovieID: movieID,
			},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to add movie to watchlist",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":   "movie added to watchlist",
			"watchlist": item,
		})
		return
	}

	// --------------------------------------------------
	// TV show
	// --------------------------------------------------
	if req.TVID != nil {
		tvID := pgtype.Int8{
			Int64: *req.TVID,
			Valid: true,
		}

		// Optional: make sure TV show exists
		_, err := wl.Queries.GetTVShowByID(ctx, *req.TVID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "tv show not found",
			})
			return
		}

		// Check duplicate
		exists, err := wl.Queries.IsTVInWatchlist(
			ctx,
			gen.IsTVInWatchlistParams{
				UserID: dbUserID,
				TvID:   tvID,
			},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to check watchlist",
			})
			return
		}

		if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error": "tv show is already in your watchlist",
			})
			return
		}

		item, err := wl.Queries.AddTVToWatchlist(
			ctx,
			gen.AddTVToWatchlistParams{
				UserID: dbUserID,
				TvID:   tvID,
			},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to add tv show to watchlist",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":   "tv show added to watchlist",
			"watchlist": item,
		})
	}
}

// CheckMovieInWatchlist checks whether the authenticated user has saved a movie.
func (wl *WatchlistHandler) CheckMovieInWatchlist(c *gin.Context) {
	movieIDStr := c.Param("movieID")

	movieID, err := strconv.Atoi(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid movie ID",
		})
		return
	}

	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	exists, err := wl.Queries.IsMovieInWatchlist(
		c.Request.Context(),
		gen.IsMovieInWatchlistParams{
			UserID:  service.UUIDToPGType(userID),
			MovieID: pgtype.Int8{Int64: int64(movieID), Valid: true},
		},
	)
	if err != nil {
		log.Printf("failed to check movie watchlist: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to check movie watchlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"in_watchlist": exists,
	})
}

// CheckTVInWatchlist checks whether the authenticated user has saved a TV show.
func (wl *WatchlistHandler) CheckTVInWatchlist(c *gin.Context) {
	tvIDStr := c.Param("tvID")

	tvID, err := strconv.Atoi(tvIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid TV ID",
		})
		return
	}
	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	exists, err := wl.Queries.IsTVInWatchlist(
		c.Request.Context(),
		gen.IsTVInWatchlistParams{
			UserID: service.UUIDToPGType(userID),
			TvID:   pgtype.Int8{Int64: int64(tvID), Valid: true},
		},
	)
	if err != nil {
		log.Printf("failed to check TV watchlist: %v", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to check TV watchlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"in_watchlist": exists,
	})
}
func (wl *WatchlistHandler) RemoveMovieFromWatchlist(c *gin.Context) {
	movieIDStr := c.Param("movieID")

	movieID, err := strconv.Atoi(movieIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid movie ID",
		})
		return
	}

	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	_, err = wl.Queries.RemoveMovieFromWatchlist(
		c.Request.Context(),
		gen.RemoveMovieFromWatchlistParams{
			UserID:  service.UUIDToPGType(userID),
			MovieID: pgtype.Int8{Int64: int64(movieID), Valid: true},
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "Movie is not in your watch list",
			})
			return
		}

		log.Printf(
			"failed to remove movie %d from watchlist for user %s: %v",
			movieID,
			userID,
			err,
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to remove movie from watchlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Movie removed from watch list successfully",
	})
}
func (wl *WatchlistHandler) RemoveTVFromWatchlist(c *gin.Context) {
	tvIDStr := c.Param("tvID")

	tvID, err := strconv.Atoi(tvIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Invalid TV ID",
		})
		return
	}

	userID, ok := service.UserIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	_, err = wl.Queries.RemoveTVFromWatchlist(
		c.Request.Context(),
		gen.RemoveTVFromWatchlistParams{
			UserID: service.UUIDToPGType(userID),
			TvID:   pgtype.Int8{Int64: int64(tvID), Valid: true},
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "TV show is not in your watch list",
			})
			return
		}

		log.Printf(
			"failed to remove TV %d from watchlist for user %s: %v",
			tvID,
			userID,
			err,
		)

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to remove TV show from watchlist",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "TV show removed from watch list successfully",
	})
}

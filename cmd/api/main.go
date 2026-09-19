package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/internal/config"
	"github.com/internal/db"
	"github.com/internal/handlers"
	"github.com/internal/middleware"
	"github.com/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	// ── Initialize Services ─────────────────────────
	if err := service.Init(); err != nil {
		log.Fatal("Failed to initialize services:", err)
	}

	// ── Database ─────────────────────────────────────
	cfg := config.LoadConfig()
	db.Connect(cfg)
	// db.LocalConnect(cfg)
	defer db.Close()

	store := db.NewStore(db.DB)

	// ── HTTP Router ──────────────────────────────────
	r := gin.New()
	r.RedirectTrailingSlash = false

	r.Use(
		cors.New(cors.Config{
			AllowOrigins: []string{
				"http://localhost:3000",
				"http://192.168.1.103:3000",
			},

			AllowMethods: []string{
				"GET",
				"POST",
				"PUT",
				"PATCH",
				"DELETE",
				"OPTIONS",
			},

			AllowHeaders: []string{
				"Origin",
				"Content-Type",
				"Accept",
				"Authorization",
			},

			ExposeHeaders: []string{
				"Content-Length",
			},

			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		gin.Logger(),
		gin.Recovery(),
	)

	// ── JWT Secret ───────────────────────────────────
	jwtSecret := os.Getenv("JWT_ACCESS_SECRET")

	if jwtSecret == "" {
		log.Fatal("JWT_ACCESS_SECRET is not set")
	}

	_ = jwtSecret

	// ── Handlers ─────────────────────────────────────
	moviesHandler := &handlers.MoviesHandler{
		Queries: store.Queries,
	}

	tvShowHandler := &handlers.TVShowsHandler{
		Queries: store.Queries,
	}

	watchlistHandler := &handlers.WatchlistHandler{
		Queries: store.Queries,
	}

	// ── Notification Hub ────────────────────────────
	notificationHub := handlers.NewNotificationHub()

	notificationsHandler := &handlers.NotificationsHandler{
		Queries: store.Queries,
		Hub:     notificationHub,
	}

	reviewsHandler := &handlers.ReviewsHandler{
		Queries: store.Queries,
		Hub:     notificationHub,
	}

	// ── Register Routes ─────────────────────────────
	registerRoutes(
		r,
		store,
		moviesHandler,
		tvShowHandler,
		reviewsHandler,
		notificationsHandler,
		watchlistHandler,
	)

	// ── Server ───────────────────────────────────────
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr: ":" + port,

		Handler: r,

		ReadTimeout: 15 * time.Second,

		// IMPORTANT:
		// SSE connection long-running হওয়ায় WriteTimeout
		// 15 seconds রাখা যাবে না।
		WriteTimeout: 0,

		IdleTimeout: 60 * time.Second,
	}

	// ── Start Server ─────────────────────────────────
	go func() {
		log.Printf("🚀 Server listening on :%s", port)

		if err := srv.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {

			log.Fatalf("Server error: %v", err)
		}
	}()

	// ── Graceful Shutdown ───────────────────────────
	quit := make(chan os.Signal, 1)

	signal.Notify(
		quit,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-quit

	log.Println("⏳ Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}

	log.Println("✅ Server stopped cleanly")
}

func registerRoutes(
	r *gin.Engine,
	store *db.Store,

	h *handlers.MoviesHandler,
	t *handlers.TVShowsHandler,
	rh *handlers.ReviewsHandler,

	n *handlers.NotificationsHandler,

	wl *handlers.WatchlistHandler,
) {

	log.Println("✅ Registering routes...")

	// ── Health ───────────────────────────────────────
	r.GET("/health", func(c *gin.Context) {
		c.JSON(
			http.StatusOK,
			gin.H{
				"status": "ok",
			},
		)
	})

	// ── Comments ─────────────────────────────────────
	r.GET(
		"/reviews/:id/comments",
		rh.GetComments,
	)

	// ── Notifications ────────────────────────────────
	//
	notifications := r.Group("/notifications")

	// Get current user's notifications
	notifications.GET(
		"",
		middleware.AuthMiddleware(),
		n.NotificationsHandler,
	)

	// Realtime SSE
	notifications.GET(
		"/stream",
		middleware.SSEAuthMiddleware(),
		middleware.AuthMiddleware(),
		n.NotificationsStreamHandler,
	)

	// Mark one read
	notifications.PATCH(
		"/:id/read",
		middleware.AuthMiddleware(),
		n.MarkReadHandler,
	)

	// Mark all read
	notifications.POST(
		"/read-all",
		middleware.AuthMiddleware(),
		n.MarkAllReadHandler,
	)
	// ── Top Review ───────────────────────────────────
	r.GET(
		"/api/top-review",
		rh.GetTopReviewHandler,
	)

	// ── Auth ─────────────────────────────────────────
	auth := r.Group("/auth")

	auth.Use(
		middleware.RateLimiter(),
	)

	{
		auth.POST(
			"/register",
			handlers.RegisterHandler,
		)

		auth.POST(
			"/signin",
			handlers.LoginHandler,
		)

		auth.POST(
			"/refresh",
			handlers.RefreshHandler,
		)

		auth.POST(
			"/logout",
			handlers.LogoutHandler,
		)
	}

	// ── Users ────────────────────────────────────────
	users := r.Group("/users")

	users.Use(
		middleware.AuthMiddleware(),
		middleware.RateLimiter(),
	)

	{
		users.GET(
			"/user/:id",
			handlers.GetUserByIDHandler,
		)

		users.GET(
			"/user_profile/:id",
			handlers.GetUserProfileByID,
		)

		users.PATCH(
			"/user/:id",
			handlers.UpdateUserByIDHandler,
		)
	}

	// ── Movies ───────────────────────────────────────
	movies := r.Group("/movies")

	{
		movies.GET(
			"",
			h.GetMoviesPaginated,
		)

		movies.GET(
			"/search",
			h.SearchMovies,
		)

		movies.GET(
			"/genre/:genre_id",
			h.ListMoviesByGenre,
		)

		movies.GET(
			"/top_movies",
			h.GetTopMovies,
		)

		movies.GET(
			"/genre",
			h.AllGenres,
		)

		movies.GET(
			"/batch",
			h.GetMoviesByIDs,
		)

		movies.GET(
			"/movie/:id",
			h.GetMovieByID,
		)

		movies.GET(
			"/movie/person/:id",
			h.GetPersonByCreditID,
		)

		movies.GET(
			"/movie/movie_credits/:id",
			h.GetCreditsByMovieID,
		)

		movies.GET(
			"/movie/movie_videos/:id",
			h.GetVideosByMovieID,
		)

		movies.DELETE(
			"/:id",
			h.DeleteMovie,
		)

		// Public
		movies.GET(
			"/movie/:id/reviews",
			h.GetReviewsByMovie,
		)

		// Authenticated
		movies.POST(
			"/movie/:id/reviews",
			middleware.AuthMiddleware(),
			h.CreateReview,
		)
	}

	// ── Reviews ──────────────────────────────────────
	reviews := r.Group("/reviews")

	reviews.Use(
		middleware.AuthMiddleware(),
	)

	{
		reviews.GET(
			"",
			rh.GetAllReviews,
		)

		reviews.GET(
			"/:id",
			h.GetReviewByID,
		)

		reviews.GET(
			"/:id/user",
			h.GetReviewsByUserID,
		)

		reviews.PATCH(
			"/:id",
			h.UpdateReview,
		)

		reviews.DELETE(
			"/:id",
			h.DeleteReview,
		)

		reviews.POST(
			"/:id/vote",
			rh.VoteOnReview,
		)

		reviews.DELETE(
			"/:id/vote",
			rh.RemoveVote,
		)

		reviews.POST(
			"/:id/like",
			rh.LikeReview,
		)

		reviews.DELETE(
			"/:id/like",
			rh.UnlikeReview,
		)

		reviews.POST(
			"/:id/report",
			rh.ReportReview,
		)

		reviews.POST(
			"/:id/comments",
			rh.AddComment,
		)
	}

	// ── TV Shows ─────────────────────────────────────
	tv := r.Group("/tv_shows")

	{
		tv.GET(
			"",
			t.GetTVShowsPaginated,
		)

		tv.GET(
			"/search",
			t.SearchTVShows,
		)

		tv.GET(
			"/genre/:genre_id",
			t.ListTVShowsByGenre,
		)

		tv.GET(
			"/tv_show/:id",
			t.GetTVShowByID,
		)

		tv.GET(
			"/tv_show/person/:id",
			t.GetTVPersonByCreditID,
		)

		tv.GET(
			"/tv_show/credits/:id",
			t.GetTVCreditsByShowID,
		)

		tv.GET(
			"/tv_show/videos/:id",
			t.GetTVVideosByShowID,
		)

		tv.DELETE(
			"/:id",
			t.DeleteTVShow,
		)

		// Public
		tv.GET(
			"/tv_show/:id/reviews",
			h.GetTVReviewsByShow,
		)

		// Authenticated
		tv.POST(
			"/tv_show/:id/reviews",
			middleware.AuthMiddleware(),
			t.CreateTVReview,
		)
	}

	// ── Watchlist ────────────────────────────────────
	watchlist := r.Group("/watchlist")

	watchlist.Use(
		middleware.AuthMiddleware(),
	)

	{
		watchlist.POST(
			"",
			wl.AddToWatchlist,
		)

		watchlist.GET(
			"",
			wl.GetMyWatchlist,
		)

		watchlist.GET(
			"/user/:id",
			wl.GetWatchlistByUserID,
		)

		watchlist.DELETE(
			"/movie/:movieID",
			wl.RemoveMovieFromWatchlist,
		)

		watchlist.DELETE(
			"/tv/:tvID",
			wl.RemoveTVFromWatchlist,
		)

		watchlist.GET(
			"/movie/:movieID",
			wl.CheckMovieInWatchlist,
		)

		watchlist.GET(
			"/tv/:tvID",
			wl.CheckTVInWatchlist,
		)
	}

	// ── Admin ────────────────────────────────────────
	/*
		adminGroup := r.Group("/admin")

		adminGroup.Use(
			middleware.AuthMiddleware(),
			middleware.AdminOnly(),
		)

		{
			adminHandler := handlers.NewAdminHandler(store.Queries)

			adminGroup.GET(
				"/dashboard",
				adminHandler.GetAdminDashboard,
			)
		}
	*/
}

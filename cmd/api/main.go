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

	// ── Environment ──────────────────────────────────
	if err := service.Init(); err != nil {
		log.Fatal("Failed to initialize services:", err)
	}

	// ── Database ─────────────────────────────────────
	cfg := config.LoadConfig()
	// db.Connect(cfg) // now returns a pool
	db.LocalConnect(cfg) // now returns a pool
	defer db.Close()

	store := db.NewStore(db.DB) // create Store

	// ── HTTP Router ─────────────────────────────────
	r := gin.New()
	r.RedirectTrailingSlash = false
	r.Use(
		cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:3000", "http://192.168.1.103:3000", "https://trading-platform-nu-seven.vercel.app"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		gin.Logger(),
		gin.Recovery(),
	)

	// JWT secret
	jwtSecret := os.Getenv("JWT_ACCESS_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_ACCESS_SECRET is not set")
	}

	moviesHandler := &handlers.MoviesHandler{Queries: store.Queries}
	TVShowHandler := &handlers.TVShowsHandler{Queries: store.Queries}
	AllReviewsHandler := &handlers.ReviewsHandler{Queries: store.Queries}
	WebhookHandler := &handlers.WebhookHandler{Queries: store.Queries}
	NotificationsHandler := &handlers.NotificationsHandler{Queries: store.Queries}
	WatchlistHandler := &handlers.WatchlistHandler{Queries: store.Queries}
	registerRoutes(r, store, moviesHandler, TVShowHandler, AllReviewsHandler, WebhookHandler, NotificationsHandler, WatchlistHandler)
	// ── Server ───────────────────────────────────────
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("🚀 Server listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// ── Graceful Shutdown ───────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("⏳ Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}
	log.Println("✅ Server stopped cleanly")
}

func registerRoutes(r *gin.Engine, store *db.Store, h *handlers.MoviesHandler, t *handlers.TVShowsHandler, rh *handlers.ReviewsHandler, w *handlers.WebhookHandler, n *handlers.NotificationsHandler, wl *handlers.WatchlistHandler) {
	log.Println("✅ Registering routes...")

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/reviews/:id/comments", rh.GetComments)

	webhookSecret := os.Getenv("WEBHOOK_SECRET")
	if webhookSecret == "" {
		webhookSecret = "change-me"
	}
	r.POST("/webhook", w.WebhookHandler)
	r.GET("/notifications", n.NotificationsHandler)
	r.PATCH("/notifications/:id/read", n.MarkReadHandler)
	r.GET("/api/top-review", rh.GetTopReviewHandler)
	// Auth
	auth := r.Group("/auth")
	auth.Use(middleware.RateLimiter())
	{
		auth.POST("/register", handlers.RegisterHandler)
		auth.POST("/signin", handlers.LoginHandler)
		auth.POST("/refresh", handlers.RefreshHandler)
		auth.POST("/logout", handlers.LogoutHandler)
	}

	// Users
	users := r.Group("/users")
	users.Use(middleware.AuthMiddleware(), middleware.RateLimiter())
	{
		users.GET("/user/:id", handlers.GetUserByIDHandler)
		users.GET("/user_profile/:id", handlers.GetUserProfileByID)
		users.PATCH("/user/:id", handlers.UpdateUserByIDHandler)
	}
	movies := r.Group("/movies")
	{
		movies.GET("", h.GetMoviesPaginated)                          // GET /movies?page=1&limit=20
		movies.GET("/search", h.SearchMovies)                         // GET /movies/search?q=batman
		movies.GET("/genre/:genre_id", h.ListMoviesByGenre)           // GET /movies/genre/28
		movies.GET("/top_movies", h.GetTopMovies)                     // GET /movies/genre/28
		movies.GET("/genre", h.AllGenres)                             // GET /movies/genre/28
		movies.GET("/batch", h.GetMoviesByIDs)                        // GET /movies/batch?ids=1,2,3
		movies.GET("/movie/:id", h.GetMovieByID)                      // GET /movies/550
		movies.GET("/movie/person/:id", h.GetPersonByCreditID)        // GET /movies/550
		movies.GET("/movie/movie_credits/:id", h.GetCreditsByMovieID) // GET /movies/550
		movies.GET("/movie/movie_videos/:id", h.GetVideosByMovieID)   // GET /movies/550
		movies.DELETE("/:id", h.DeleteMovie)                          // DELETE /movies/550 (admin)
		movies.GET("/movie/:id/reviews", h.GetReviewsByMovie)         // public
		movies.POST("/movie/:id/reviews", middleware.AuthMiddleware(), h.CreateReview)
	}
	reviews := r.Group("/reviews")
	reviews.Use(middleware.AuthMiddleware())
	{
		reviews.GET("", rh.GetAllReviews)
		reviews.GET("/:id", h.GetReviewByID)
		reviews.GET("/:id/user", h.GetReviewsByUserID)
		reviews.PATCH("/:id", h.UpdateReview)
		reviews.DELETE("/:id", h.DeleteReview)
		reviews.POST("/:id/vote", rh.VoteOnReview)
		reviews.DELETE("/:id/vote", rh.RemoveVote)
		reviews.POST("/:id/like", rh.LikeReview)
		reviews.DELETE("/:id/like", rh.UnlikeReview)
		reviews.POST("/:id/report", rh.ReportReview)
		reviews.POST("/:id/comments", rh.AddComment)
	}
	tv := r.Group("/tv_shows")
	{
		tv.GET("", t.GetTVShowsPaginated)                                              // GET /tv?page=1&limit=20
		tv.GET("/search", t.SearchTVShows)                                             // GET /tv/search?q=batman
		tv.GET("/genre/:genre_id", t.ListTVShowsByGenre)                               // GET /tv/genre/28                                          // GET /tv/batch?ids=1,2,3
		tv.GET("/tv_show/:id", t.GetTVShowByID)                                        // GET /tv/show/550
		tv.GET("/tv_show/person/:id", t.GetTVPersonByCreditID)                         // GET /tv/show/person/123
		tv.GET("/tv_show/credits/:id", t.GetTVCreditsByShowID)                         // GET /tv/show/credits/550
		tv.GET("/tv_show/videos/:id", t.GetTVVideosByShowID)                           // GET /tv/show/videos/550
		tv.DELETE("/:id", t.DeleteTVShow)                                              // DELETE /tv/550 (admin)
		tv.GET("/tv_show/:id/reviews", h.GetTVReviewsByShow)                           // public
		tv.POST("/tv_show/:id/reviews", middleware.AuthMiddleware(), t.CreateTVReview) // authenticated
	}

	watchlist := r.Group("/watchlist")
	{
		watchlist.GET("/:id", wl.GetWatchlistByUserID)
	}

	// // Admin
	// adminGroup := r.Group("/admin")
	// adminGroup.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	// {
	// 	_ = handlers.NewAdminHandler(store.Queries)
	// 	// adminGroup.GET("/dashboard", adminHandler.GetAdminDashboard)
	// 	// adminGroup.GET("/users", adminHandler.GetUsersHandler)
	// 	// adminGroup.GET("/users/banned", adminHandler.GetBannedUsersHandler)
	// 	// adminGroup.GET("/users/search", adminHandler.SearchUsersHandler)
	// 	// adminGroup.PATCH("/users/ban/:id", adminHandler.BanUser)
	// 	// adminGroup.PATCH("/users/unban/:id", adminHandler.UnbanUser)
	// 	// adminGroup.PATCH("/users/role/:id", adminHandler.UpdateUserRole)
	// }
}

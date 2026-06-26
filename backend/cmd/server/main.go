package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
	"github.com/trello-clone/backend/internal/auth"
	"github.com/trello-clone/backend/internal/config"
	"github.com/trello-clone/backend/internal/database"
	"github.com/trello-clone/backend/internal/email"
	"github.com/trello-clone/backend/internal/handler"
	authmw "github.com/trello-clone/backend/internal/middleware"
	"github.com/trello-clone/backend/internal/seed"
	"github.com/trello-clone/backend/internal/store"
	"github.com/trello-clone/backend/internal/ws"
)

func main() {
	cfg := config.Load()

	if err := database.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	var rdb *redis.Client
	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Fatalf("redis parse: %v", err)
		}
		rdb = redis.NewClient(opt)
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("redis unavailable, running without pub/sub: %v", err)
			rdb = nil
		}
	}

	st := store.New(pool)
	tm := auth.NewTokenManager(cfg.JWTSecret)
	seed.AdminUser(ctx, st, tm, cfg)
	mailer := email.New(cfg)
	hub := ws.NewHub(rdb, tm)

	corsOrigins := config.AllowedWebOrigins(cfg.WebURL)
	log.Printf("WEB_URL=%q API_URL=%q CORS=%v", cfg.WebURL, cfg.APIURL, corsOrigins)

	authH := handler.NewAuthHandler(st, tm, cfg, mailer)
	notifH := handler.NewNotificationHandler(st, hub)
	wsH := handler.NewWorkspaceHandler(st, cfg, hub, mailer, notifH)
	boardH := handler.NewBoardHandler(st, hub, notifH)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/ws", hub.HandleWS)
	r.Get("/ws/notifications", hub.HandleUserWS)

	authMiddleware := authmw.Auth(tm)
	verifiedMiddleware := authmw.RequireVerified(st)

	r.Route("/api/v1", func(api chi.Router) {
		handler.RegisterAuthRoutes(api, authH, authMiddleware)
		api.Group(func(protected chi.Router) {
			protected.Use(authMiddleware)
			protected.Use(verifiedMiddleware)
			handler.RegisterWorkspaceRoutes(protected, wsH)
			handler.RegisterBoardRoutes(protected, boardH)
		})
		api.Group(func(authOnly chi.Router) {
			authOnly.Use(authMiddleware)
			handler.RegisterInvitationRoutes(authOnly, wsH)
			handler.RegisterNotificationRoutes(authOnly, notifH)
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("server listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

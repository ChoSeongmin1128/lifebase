package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	authhttp "lifebase/internal/auth/adapter/in/http"
	authpg "lifebase/internal/auth/adapter/out/postgres"
	authusecase "lifebase/internal/auth/usecase"
	cloudhttp "lifebase/internal/cloud/adapter/in/http"
	"lifebase/internal/cloud/adapter/out/filesystem"
	cloudpg "lifebase/internal/cloud/adapter/out/postgres"
	cloudusecase "lifebase/internal/cloud/usecase"
	galleryhttp "lifebase/internal/gallery/adapter/in/http"
	gallerypg "lifebase/internal/gallery/adapter/out/postgres"
	galleryusecase "lifebase/internal/gallery/usecase"
	sharinghttp "lifebase/internal/sharing/adapter/in/http"
	sharingpg "lifebase/internal/sharing/adapter/out/postgres"
	sharingusecase "lifebase/internal/sharing/usecase"
	"lifebase/internal/shared/config"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
	"lifebase/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Database
	dbpool, err := pgxpool.New(context.Background(), cfg.Database.URL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(context.Background()); err != nil {
		slog.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("database connected")

	// Repositories
	userRepo := authpg.NewUserRepo(dbpool)
	googleAccountRepo := authpg.NewGoogleAccountRepo(dbpool)
	refreshTokenRepo := authpg.NewRefreshTokenRepo(dbpool)
	folderRepo := cloudpg.NewFolderRepo(dbpool)
	fileRepo := cloudpg.NewFileRepo(dbpool)

	// Storage
	storage := filesystem.NewLocalStorage(cfg.Storage.DataPath)

	// Asynq
	asynqClient := worker.NewAsynqClient(cfg.Redis.URL)
	if asynqClient != nil {
		defer asynqClient.Close()
	}

	// Worker server
	workerSrv := worker.StartWorkerServer(cfg.Redis.URL, dbpool, cfg.Storage.DataPath, cfg.Storage.ThumbPath)
	_ = workerSrv

	// Gallery repos
	mediaRepo := gallerypg.NewMediaRepo(dbpool)

	// Sharing repos
	shareRepo := sharingpg.NewShareRepo(dbpool)
	inviteRepo := sharingpg.NewInviteRepo(dbpool)

	// Use Cases
	authUC := authusecase.NewAuthUseCase(cfg, userRepo, googleAccountRepo, refreshTokenRepo)
	cloudUC := cloudusecase.NewCloudUseCase(folderRepo, fileRepo, storage, asynqClient)
	galleryUC := galleryusecase.NewGalleryUseCase(mediaRepo)
	sharingUC := sharingusecase.NewSharingUseCase(shareRepo, inviteRepo)

	// Handlers
	authHandler := authhttp.NewAuthHandler(authUC)
	cloudHandler := cloudhttp.NewCloudHandler(cloudUC)
	galleryHandler := galleryhttp.NewGalleryHandler(galleryUC, cfg.Storage.ThumbPath)
	sharingHandler := sharinghttp.NewSharingHandler(sharingUC)

	// Router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.NewRateLimiter(100).Handler)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.Server.WebURL()},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			response.JSON(w, http.StatusOK, map[string]string{
				"status": "ok",
				"time":   time.Now().UTC().Format(time.RFC3339),
			})
		})

		// Auth (public)
		r.Route("/auth", func(r chi.Router) {
			r.Get("/url", authHandler.GetAuthURL)
			r.Post("/callback", authHandler.HandleCallback)
			r.Post("/refresh", authHandler.RefreshToken)
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWT.Secret))

			r.Post("/auth/logout", authHandler.Logout)

			r.Get("/me", func(w http.ResponseWriter, r *http.Request) {
				userID := middleware.GetUserID(r.Context())
				response.JSON(w, http.StatusOK, map[string]string{
					"user_id": userID,
				})
			})

			// Cloud
			r.Route("/cloud", func(r chi.Router) {
				// Folders
				r.Post("/folders", cloudHandler.CreateFolder)
				r.Get("/folders/{folderID}", cloudHandler.GetFolder)
				r.Get("/folders", cloudHandler.ListFolder)
				r.Patch("/folders/{folderID}/rename", cloudHandler.RenameFolder)
				r.Patch("/folders/{folderID}/move", cloudHandler.MoveFolder)
				r.Delete("/folders/{folderID}", cloudHandler.DeleteFolder)

				// Files
				r.Post("/files/upload", cloudHandler.UploadFile)
				r.Get("/files/{fileID}", cloudHandler.GetFile)
				r.Get("/files/{fileID}/download", cloudHandler.DownloadFile)
				r.Patch("/files/{fileID}/rename", cloudHandler.RenameFile)
				r.Patch("/files/{fileID}/move", cloudHandler.MoveFile)
				r.Delete("/files/{fileID}", cloudHandler.DeleteFile)

				// Trash
				r.Get("/trash", cloudHandler.ListTrash)
				r.Post("/trash/restore", cloudHandler.RestoreItem)
				r.Delete("/trash", cloudHandler.EmptyTrash)

				// Search
				r.Get("/search", cloudHandler.SearchFiles)
			})

			// Gallery
			r.Route("/gallery", func(r chi.Router) {
				r.Get("/", galleryHandler.ListMedia)
				r.Get("/thumbnails/{fileID}/{size}", galleryHandler.GetThumbnail)
			})

			// Sharing
			r.Route("/shares", func(r chi.Router) {
				r.Post("/invite", sharingHandler.CreateInvite)
				r.Post("/accept", sharingHandler.AcceptInvite)
				r.Get("/", sharingHandler.ListShares)
				r.Get("/shared-with-me", sharingHandler.ListSharedWithMe)
				r.Delete("/{shareID}", sharingHandler.RemoveShare)
			})
		})
	})

	// Server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		slog.Info("server starting", "addr", addr, "env", cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

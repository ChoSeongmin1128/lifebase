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

	adminhttp "lifebase/internal/admin/adapter/in/http"
	adminpg "lifebase/internal/admin/adapter/out/postgres"
	adminusecase "lifebase/internal/admin/usecase"
	authhttp "lifebase/internal/auth/adapter/in/http"
	authbootstrap "lifebase/internal/auth/adapter/out/bootstrap"
	authgoogle "lifebase/internal/auth/adapter/out/google"
	authpg "lifebase/internal/auth/adapter/out/postgres"
	authusecase "lifebase/internal/auth/usecase"
	calendarhttp "lifebase/internal/calendar/adapter/in/http"
	calendarpg "lifebase/internal/calendar/adapter/out/postgres"
	calendarusecase "lifebase/internal/calendar/usecase"
	cloudhttp "lifebase/internal/cloud/adapter/in/http"
	cloudasynq "lifebase/internal/cloud/adapter/out/asynq"
	"lifebase/internal/cloud/adapter/out/filesystem"
	cloudpg "lifebase/internal/cloud/adapter/out/postgres"
	cloudusecase "lifebase/internal/cloud/usecase"
	galleryhttp "lifebase/internal/gallery/adapter/in/http"
	gallerypg "lifebase/internal/gallery/adapter/out/postgres"
	galleryusecase "lifebase/internal/gallery/usecase"
	settingshttp "lifebase/internal/settings/adapter/in/http"
	settingspg "lifebase/internal/settings/adapter/out/postgres"
	settingsusecase "lifebase/internal/settings/usecase"
	"lifebase/internal/shared/config"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
	sharinghttp "lifebase/internal/sharing/adapter/in/http"
	sharingpg "lifebase/internal/sharing/adapter/out/postgres"
	sharingusecase "lifebase/internal/sharing/usecase"
	todohttp "lifebase/internal/todo/adapter/in/http"
	todopg "lifebase/internal/todo/adapter/out/postgres"
	todousecase "lifebase/internal/todo/usecase"
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
	adminRepo := adminpg.NewAdminRepo(dbpool)
	storageResetRepo := adminpg.NewStorageResetRepo(dbpool)
	folderRepo := cloudpg.NewFolderRepo(dbpool)
	fileRepo := cloudpg.NewFileRepo(dbpool)
	cloudSharedRepo := cloudpg.NewSharedRepo(dbpool)
	starRepo := cloudpg.NewStarRepo(dbpool)

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

	// Calendar repos
	calendarRepo := calendarpg.NewCalendarRepo(dbpool)
	eventRepo := calendarpg.NewEventRepo(dbpool)
	reminderRepo := calendarpg.NewReminderRepo(dbpool)

	// Todo repos
	todoListRepo := todopg.NewListRepo(dbpool)
	todoItemRepo := todopg.NewTodoRepo(dbpool)

	// Sharing repos
	shareRepo := sharingpg.NewShareRepo(dbpool)
	inviteRepo := sharingpg.NewInviteRepo(dbpool)

	// Use Cases
	redirects := map[string]string{
		"web":   cfg.Server.WebOrigin + "/auth/callback",
		"admin": cfg.Server.AdminOrigin + "/admin/auth/callback",
	}
	authOAuthClient := authgoogle.NewOAuthClient(cfg.Google.ClientID, cfg.Google.ClientSecret, redirects)
	authBootstrapper := authbootstrap.NewTodoBootstrapper(todoListRepo)
	authUC := authusecase.NewAuthUseCase(
		authusecase.JWTOptions{
			Secret:        cfg.JWT.Secret,
			AccessExpiry:  cfg.JWT.AccessExpiry,
			RefreshExpiry: cfg.JWT.RefreshExpiry,
		},
		userRepo,
		googleAccountRepo,
		refreshTokenRepo,
		authOAuthClient,
		authBootstrapper,
	)
	thumbnailQueue := cloudasynq.NewThumbnailQueue(asynqClient)
	cloudUC := cloudusecase.NewCloudUseCase(folderRepo, fileRepo, cloudSharedRepo, starRepo, storage, thumbnailQueue)
	galleryUC := galleryusecase.NewGalleryUseCase(mediaRepo)
	calendarUC := calendarusecase.NewCalendarUseCase(calendarRepo, eventRepo, reminderRepo)
	todoUC := todousecase.NewTodoUseCase(todoListRepo, todoItemRepo)
	sharingUC := sharingusecase.NewSharingUseCase(shareRepo, inviteRepo)
	settingsUC := settingsusecase.NewSettingsUseCase(settingspg.NewSettingsRepo(dbpool))
	adminUC := adminusecase.NewAdminUseCase(
		adminRepo,
		userRepo,
		adminRepo,
		storageResetRepo,
		cfg.Storage.DataPath,
		cfg.Storage.ThumbPath,
	)

	// Handlers
	authHandler := authhttp.NewAuthHandler(authUC, cfg.StateHMACKey)
	cloudHandler := cloudhttp.NewCloudHandler(cloudUC)
	galleryHandler := galleryhttp.NewGalleryHandler(galleryUC, cfg.Storage.ThumbPath)
	calendarHandler := calendarhttp.NewCalendarHandler(calendarUC)
	settingsHandler := settingshttp.NewSettingsHandler(settingsUC)
	todoHandler := todohttp.NewTodoHandler(todoUC)
	sharingHandler := sharinghttp.NewSharingHandler(sharingUC)
	adminHandler := adminhttp.NewAdminHandler(adminUC)

	// Router
	r := chi.NewRouter()

	// Middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(middleware.NewRateLimiter(100).Handler)
	allowedOrigins := []string{cfg.Server.WebURL()}
	if cfg.Server.AdminOrigin != "" && cfg.Server.AdminOrigin != cfg.Server.WebURL() {
		allowedOrigins = append(allowedOrigins, cfg.Server.AdminOrigin)
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
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

				// Views
				r.Get("/recent", cloudHandler.ListRecent)
				r.Get("/shared", cloudHandler.ListShared)
				r.Get("/starred", cloudHandler.ListStarred)

				// Stars
				r.Get("/stars", cloudHandler.ListStars)
				r.Post("/stars", cloudHandler.StarItem)
				r.Delete("/stars", cloudHandler.UnstarItem)

				// Search
				r.Get("/search", cloudHandler.SearchFiles)
			})

			// Gallery
			r.Route("/gallery", func(r chi.Router) {
				r.Get("/", galleryHandler.ListMedia)
				r.Get("/thumbnails/{fileID}/{size}", galleryHandler.GetThumbnail)
			})

			// Calendar
			r.Route("/calendars", func(r chi.Router) {
				r.Post("/", calendarHandler.CreateCalendar)
				r.Get("/", calendarHandler.ListCalendars)
				r.Patch("/{calendarID}", calendarHandler.UpdateCalendar)
				r.Delete("/{calendarID}", calendarHandler.DeleteCalendar)
			})
			r.Route("/events", func(r chi.Router) {
				r.Post("/", calendarHandler.CreateEvent)
				r.Get("/", calendarHandler.ListEvents)
				r.Get("/{eventID}", calendarHandler.GetEvent)
				r.Patch("/{eventID}", calendarHandler.UpdateEvent)
				r.Delete("/{eventID}", calendarHandler.DeleteEvent)
			})

			// Todo
			r.Route("/todo", func(r chi.Router) {
				r.Post("/lists", todoHandler.CreateList)
				r.Get("/lists", todoHandler.ListLists)
				r.Patch("/lists/{listID}", todoHandler.UpdateList)
				r.Delete("/lists/{listID}", todoHandler.DeleteList)

				r.Patch("/reorder", todoHandler.ReorderTodos)
				r.Post("/", todoHandler.CreateTodo)
				r.Get("/", todoHandler.ListTodos)
				r.Get("/{todoID}", todoHandler.GetTodo)
				r.Patch("/{todoID}", todoHandler.UpdateTodo)
				r.Delete("/{todoID}", todoHandler.DeleteTodo)
			})

			// Settings
			r.Route("/settings", func(r chi.Router) {
				r.Get("/", settingsHandler.GetAll)
				r.Patch("/", settingsHandler.Update)
			})

			// Sharing
			r.Route("/shares", func(r chi.Router) {
				r.Post("/invite", sharingHandler.CreateInvite)
				r.Post("/accept", sharingHandler.AcceptInvite)
				r.Get("/", sharingHandler.ListShares)
				r.Get("/shared-with-me", sharingHandler.ListSharedWithMe)
				r.Delete("/{shareID}", sharingHandler.RemoveShare)
			})

			// Admin
			r.Route("/admin", func(r chi.Router) {
				r.Use(middleware.Admin(adminRepo))
				r.Use(middleware.NewRateLimiter(30).Handler)

				r.Get("/users", adminHandler.ListUsers)
				r.Get("/users/{userID}", adminHandler.GetUser)
				r.Patch("/users/{userID}/quota", adminHandler.UpdateQuota)
				r.Post("/users/{userID}/recalculate-storage", adminHandler.RecalculateStorage)
				r.Post("/users/{userID}/reset-storage", adminHandler.ResetStorage)
				r.Patch("/users/{userID}/google-accounts/{accountID}/status", adminHandler.UpdateGoogleAccountStatus)

				r.Get("/admins", adminHandler.ListAdmins)
				r.Post("/admins", adminHandler.CreateAdmin)
				r.Patch("/admins/{adminID}/role", adminHandler.UpdateAdminRole)
				r.Patch("/admins/{adminID}/deactivate", adminHandler.DeactivateAdmin)
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

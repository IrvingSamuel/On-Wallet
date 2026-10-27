package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"carteira/internal/assistant"
	"carteira/internal/auth"
	"carteira/internal/collections"
	"carteira/internal/config"
	"carteira/internal/db"
	"carteira/internal/documents"
	"carteira/internal/httpx"
	"carteira/internal/shares"
	"carteira/internal/storage"
	"carteira/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bootCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	pool, err := db.Connect(bootCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(bootCtx, pool); err != nil {
		return err
	}

	store, err := storage.New(bootCtx, cfg.MinIO)
	if err != nil {
		return err
	}

	docsSvc := documents.NewService(documents.NewRepository(pool), store, cfg.MaxUploadBytes, cfg.PublicOrigin)
	sharesSvc := shares.NewService(shares.NewRepository(pool), store)
	collSvc := collections.NewService(collections.NewRepository(pool))
	authSvc := auth.NewService(auth.NewRepository(pool))
	groq := assistant.NewGroqClient(cfg.GroqAPIKey, cfg.GroqChatModel, cfg.GroqVisionModel)
	assistSvc := assistant.NewService(assistant.NewRepository(pool), store, groq)
	docsSvc.SetShareRevoker(sharesSvc)
	docsSvc.SetAssistAnalyzer(assistSvc)
	docsSvc.StartExpiryJob(ctx, time.Minute)
	docs := documents.NewHandler(docsSvc)
	shareH := shares.NewHandler(sharesSvc)
	collH := collections.NewHandler(collSvc)
	assistH := assistant.NewHandler(assistSvc)
	secureCookie := strings.HasPrefix(cfg.PublicOrigin, "https://")
	authH := auth.NewHandler(authSvc, secureCookie)

	api := http.NewServeMux()
	api.HandleFunc("GET /api/v1/health", health(pool, store))
	authH.Register(api)
	docs.Register(api)
	shareH.Register(api)
	collH.Register(api)
	assistH.Register(api)
	if groq.Enabled() {
		slog.Info("assistant enabled", "chat_model", cfg.GroqChatModel, "vision_model", cfg.GroqVisionModel)
	} else {
		slog.Warn("assistant disabled: GROQ_API_KEY not set")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", authH.Middleware(api))
	mux.Handle("/", web.SPA())

	srv := &http.Server{
		Addr:              "127.0.0.1:" + cfg.Port,
		Handler:           httpx.Logger(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	slog.Info("listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func health(pool *pgxpool.Pool, store *storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		checks := map[string]string{"postgres": "ok", "minio": "ok"}
		status := http.StatusOK
		if err := pool.Ping(ctx); err != nil {
			checks["postgres"] = err.Error()
			status = http.StatusServiceUnavailable
		}
		if err := store.Ping(ctx); err != nil {
			checks["minio"] = err.Error()
			status = http.StatusServiceUnavailable
		}
		state := "ok"
		if status != http.StatusOK {
			state = "degraded"
		}
		httpx.JSON(w, status, map[string]any{"status": state, "checks": checks})
	}
}

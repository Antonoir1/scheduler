package main

import (
	"context"
	"os"
	"strings"

	"github.com/Antonoir1/scheduler/internal/handlers"
	"github.com/Antonoir1/scheduler/internal/service"
	"github.com/Antonoir1/scheduler/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

func main() {
	// LOG_LEVEL controls verbosity and defaults to info when unset or invalid.
	level := zerolog.InfoLevel
	if configured, err := zerolog.ParseLevel(strings.ToLower(os.Getenv("LOG_LEVEL"))); err == nil && os.Getenv("LOG_LEVEL") != "" {
		level = configured
	}
	console := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}
	logger := zerolog.New(console).Level(level).With().Timestamp().Logger()
	ctx := context.Background()
	store, err := storage.NewMongoStore(ctx, env("MONGO_URI", "mongodb://localhost:27017"), env("MONGO_DATABASE", "scheduler"), "jobs")
	if err != nil {
		logger.Fatal().Err(err).Msg("connect to MongoDB")
	}
	defer store.Close(ctx)

	natsConnection, err := nats.Connect(env("NATS_URL", nats.DefaultURL))
	if err != nil {
		logger.Fatal().Err(err).Msg("connect to NATS")
	}
	defer natsConnection.Drain()

	jobs := service.New(store, natsConnection, logger)
	if err := jobs.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("start scheduler")
	}
	defer jobs.Stop()
	router := gin.Default()
	handlers.RegisterHandlers(router, jobs, logger)
	logger.Info().Str("address", env("HTTP_ADDR", ":8080")).Msg("start HTTP server")
	if err := router.Run(env("HTTP_ADDR", ":8080")); err != nil {
		logger.Fatal().Err(err).Msg("run HTTP server")
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

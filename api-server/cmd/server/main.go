package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/api"
	"github.com/mungch0120/qsim-cluster/api-server/internal/store"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "qsim-api-server",
		Short: "Quantum Simulator Cluster API Server",
		Long:  "REST API server for managing quantum circuit simulation jobs",
		Run:   runServer,
	}

	// Add flags
	rootCmd.Flags().String("config", "", "config file (default is ./config.yaml)")
	rootCmd.Flags().String("port", "8080", "server port")
	rootCmd.Flags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.Flags().String("postgres-url", "", "PostgreSQL connection string")
	rootCmd.Flags().String("redis-url", "", "Redis connection string")
	rootCmd.Flags().String("analyzer-url", "", "Circuit analyzer service URL")

	viper.BindPFlags(rootCmd.Flags())

	// Execute
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	// Initialize configuration
	initConfig()

	// Initialize logger
	logger := initLogger()
	defer logger.Sync()

	logger.Info("Starting Quantum Simulator API Server",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("date", date),
	)

	// Initialize stores
	stores, err := initStores(logger)
	if err != nil {
		logger.Fatal("Failed to initialize stores", zap.Error(err))
	}

	// Initialize API router
	router := api.NewRouter(stores, logger)

	// Create HTTP server
	port := viper.GetString("port")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", zap.String("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initConfig() {
	// Set config file
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
	}

	// Environment variables
	viper.AutomaticEnv()
	viper.SetEnvPrefix("QSIM")

	// Read config file if present
	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Set defaults
	viper.SetDefault("port", "8080")
	viper.SetDefault("log-level", "info")
	viper.SetDefault("postgres-url", "postgres://localhost:5432/qsim?sslmode=disable")
	viper.SetDefault("redis-url", "redis://localhost:6379/0")
	viper.SetDefault("analyzer-url", "http://localhost:8081")
}

func initLogger() *zap.Logger {
	level := viper.GetString("log-level")
	
	var config zap.Config
	if gin.Mode() == gin.ReleaseMode {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Set log level
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	return logger
}

func initStores(logger *zap.Logger) (*store.Stores, error) {
	postgresURL := viper.GetString("postgres-url")
	redisURL := viper.GetString("redis-url")

	stores, err := store.New(store.Config{
		PostgresURL: postgresURL,
		RedisURL:    redisURL,
	}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize stores: %w", err)
	}

	return stores, nil
}
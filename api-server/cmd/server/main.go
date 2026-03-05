package main

import (
	"context"
	"fmt"
	"strings"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/mungch0120/qsim-cluster/api-server/internal/analyzer"
	"github.com/mungch0120/qsim-cluster/api-server/internal/api"
	"github.com/mungch0120/qsim-cluster/api-server/internal/k8s"
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
	rootCmd.Flags().String("kubeconfig", "", "path to kubeconfig file")
	rootCmd.Flags().Bool("in-cluster", false, "use in-cluster Kubernetes configuration")

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

	// Initialize Kubernetes client
	k8sClient, err := initK8sClient(logger)
	if err != nil {
		logger.Fatal("Failed to initialize Kubernetes client", zap.Error(err))
	}

	// Initialize analyzer client
	analyzerClient := initAnalyzerClient(logger)

	// Initialize API router
	router := api.NewRouter(stores, k8sClient, analyzerClient, logger)

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
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

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
	viper.SetDefault("in-cluster", false)
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

func initK8sClient(logger *zap.Logger) (*k8s.Client, error) {
	kubeconfig := viper.GetString("kubeconfig")
	inCluster := viper.GetBool("in-cluster")

	config := k8s.Config{
		KubeConfig: kubeconfig,
		InCluster:  inCluster,
	}

	client, err := k8s.NewClient(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Kubernetes client: %w", err)
	}

	return client, nil
}

func initAnalyzerClient(logger *zap.Logger) *analyzer.Client {
	analyzerURL := viper.GetString("analyzer-url")

	config := analyzer.Config{
		BaseURL: analyzerURL,
		Timeout: 30 * time.Second,
	}

	client := analyzer.NewClient(config, logger)

	// Test connectivity (non-blocking)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Health(ctx); err != nil {
			logger.Warn("Analyzer service health check failed", 
				zap.String("url", analyzerURL),
				zap.Error(err),
			)
		} else {
			logger.Info("Analyzer service is healthy", zap.String("url", analyzerURL))
		}
	}()

	return client
}
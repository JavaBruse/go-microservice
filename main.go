package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-microservice/handlers"
	"go-microservice/metrics"
	"go-microservice/services"
	"go-microservice/utils"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func initRedis() *redis.Client {
	// Читаем пароль из переменной окружения
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

	// Если хост/порт не заданы - используем дефолтные
	if redisHost == "" {
		redisHost = "redis"
	}
	if redisPort == "" {
		redisPort = "6379"
	}

	redisAddr := redisHost + ":" + redisPort

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
		PoolSize: 100,
	})

	// Проверка подключения
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis not available: %v", err)
		return nil
	}

	log.Println("✅ Redis connected successfully")
	return redisClient
}

func main() {

	// Initialize Redis
	redisClient := initRedis()

	// Initialize services
	integrationService := services.NewIntegrationService()
	analyticsService := services.NewAnalyticsService(50, redisClient) // window size = 50

	// Initialize handlers
	integrationHandler := handlers.NewIntegrationHandler(integrationService)
	metricsHandler := handlers.NewMetricsHandler(analyticsService)

	r := mux.NewRouter()

	// Middleware
	r.Use(utils.RateLimitMiddleware)
	r.Use(metrics.MetricsMiddleware)

	// User routes

	// Integration routes
	integrationRouter := r.PathPrefix("/api/integrations").Subrouter()
	integrationRouter.HandleFunc("/external", integrationHandler.CallExternalAPI).Methods("POST")

	// Metrics and Analytics routes
	analyticsRouter := r.PathPrefix("/api/analytics").Subrouter()
	analyticsRouter.HandleFunc("/metrics", metricsHandler.ReceiveMetrics).Methods("POST")
	analyticsRouter.HandleFunc("/stats", metricsHandler.GetAnalytics).Methods("GET")
	analyticsRouter.HandleFunc("/anomalies", func(w http.ResponseWriter, r *http.Request) {
		anomalies, err := analyticsService.GetRecentAnomalies(10)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"recent_anomalies": anomalies,
		})
	}).Methods("GET")

	analyticsRouter.HandleFunc("/cache/stats", func(w http.ResponseWriter, r *http.Request) {
		cacheSize, err := analyticsService.GetCacheStats()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"cache_size_bytes": cacheSize,
			"cache_size_mb":    float64(cacheSize) / 1024 / 1024,
		})
	}).Methods("GET")

	// Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Проверяем Redis
		redisStatus := "healthy"
		if redisClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := redisClient.Ping(ctx).Err(); err != nil {
				redisStatus = "unhealthy"
			}
		}

		response := map[string]interface{}{
			"status":      "OK",
			"timestamp":   time.Now(),
			"redis":       redisStatus,
			"environment": os.Getenv("ENVIRONMENT"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// Redis info endpoint
	r.HandleFunc("/redis/info", func(w http.ResponseWriter, r *http.Request) {
		if redisClient == nil {
			http.Error(w, "Redis not available", http.StatusServiceUnavailable)
			return
		}

		ctx := context.Background()
		info, err := redisClient.Info(ctx).Result()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(info))
	}).Methods("GET")

	// Start server with graceful shutdown
	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")

	// Stop analytics service
	analyticsService.Stop()

	// Close Redis connection
	if redisClient != nil {
		redisClient.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

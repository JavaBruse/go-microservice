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
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")

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
	redisClient := initRedis()
	analyticsService := services.NewAnalyticsService(50, redisClient)
	metricsHandler := handlers.NewMetricsHandler(analyticsService)

	r := mux.NewRouter()

	// Middleware ДО объявления маршрутов
	routerWithMiddleware := r.PathPrefix("").Subrouter()
	routerWithMiddleware.Use(utils.RateLimitMiddleware)
	routerWithMiddleware.Use(metrics.MetricsMiddleware)

	// Все маршруты через routerWithMiddleware
	routerWithMiddleware.HandleFunc("/api/analytics/metrics", metricsHandler.ReceiveMetrics).Methods("POST")
	routerWithMiddleware.HandleFunc("/api/analytics/stats", metricsHandler.GetAnalytics).Methods("GET")
	routerWithMiddleware.HandleFunc("/api/analytics/anomalies", func(w http.ResponseWriter, r *http.Request) {
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

	routerWithMiddleware.HandleFunc("/api/analytics/cache/stats", func(w http.ResponseWriter, r *http.Request) {
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

	// Метрики Prometheus и health без middleware
	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
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

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

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

	analyticsService.Stop()

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

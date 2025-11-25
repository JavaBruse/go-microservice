package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"go-microservice/handler"
	"go-microservice/metrics"
	"go-microservice/services"
	"go-microservice/utils"

	"github.com/gorilla/mux"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	minioClient, err := minio.New("minio:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	bucketName := "users"
	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
		if !exists || errBucketExists != nil {
			log.Fatalf("Failed to create bucket: %v", err)
		}
	}

	userService := services.NewUserService(minioClient, bucketName)
	integrationService := services.NewIntegrationService()

	userHandler := handlers.NewUserHandler(userService)
	integrationHandler := handlers.NewIntegrationHandler(integrationService)

	r := mux.NewRouter()

	r.Use(utils.RateLimitMiddleware)
	r.Use(metrics.MetricsMiddleware)

	userRouter := r.PathPrefix("/api/users").Subrouter()
	userRouter.HandleFunc("", userHandler.CreateUser).Methods("POST")
	userRouter.HandleFunc("", userHandler.GetAllUsers).Methods("GET")
	userRouter.HandleFunc("/{id}", userHandler.GetUser).Methods("GET")
	userRouter.HandleFunc("/{id}", userHandler.UpdateUser).Methods("PUT")
	userRouter.HandleFunc("/{id}", userHandler.DeleteUser).Methods("DELETE")

	integrationRouter := r.PathPrefix("/api/integrations").Subrouter()
	integrationRouter.HandleFunc("/external", integrationHandler.CallExternalAPI).Methods("POST")

	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Println("Server starting on :8080")
	log.Fatal(srv.ListenAndServe())
}

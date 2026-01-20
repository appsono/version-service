package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"sono-version-service/config"
	"sono-version-service/database"
	"sono-version-service/handlers"
	"sono-version-service/middleware"
	"sono-version-service/models"
	"sono-version-service/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.VersionsFile), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		log.Println("Continuing without database logging...")
	}
	if db != nil {
		defer db.Close()
	}

	versionStore, err := models.NewVersionStore(cfg.VersionsFile)
	if err != nil {
		log.Fatalf("Failed to initialize version store: %v", err)
	}

	var store storage.Storage

	switch cfg.StorageType {
	case "s3":
		s3Store, err := storage.NewS3Storage(storage.S3Config{
			Endpoint:        cfg.S3Endpoint,
			Region:          cfg.S3Region,
			Bucket:          cfg.S3Bucket,
			AccessKeyID:     cfg.S3AccessKeyID,
			SecretAccessKey: cfg.S3SecretAccessKey,
			UsePathStyle:    cfg.S3UsePathStyle,
		})
		if err != nil {
			log.Fatalf("Failed to initialize S3 storage: %v", err)
		}
		store = s3Store

	case "local":
		localStore, err := storage.NewLocalStorage(cfg.LocalStorePath)
		if err != nil {
			log.Fatalf("Failed to initialize local storage: %v", err)
		}
		store = localStore

	case "both":
		var s3Store *storage.S3Storage
		if cfg.S3Endpoint != "" && cfg.S3Bucket != "" {
			s3Store, err = storage.NewS3Storage(storage.S3Config{
				Endpoint:        cfg.S3Endpoint,
				Region:          cfg.S3Region,
				Bucket:          cfg.S3Bucket,
				AccessKeyID:     cfg.S3AccessKeyID,
				SecretAccessKey: cfg.S3SecretAccessKey,
				UsePathStyle:    cfg.S3UsePathStyle,
			})
			if err != nil {
				log.Printf("Warning: Failed to initialize S3 storage: %v", err)
			}
		}

		localStore, err := storage.NewLocalStorage(cfg.LocalStorePath)
		if err != nil {
			log.Fatalf("Failed to initialize local storage: %v", err)
		}

		store = storage.NewFallbackStorage(s3Store, localStore)

	default:
		log.Fatalf("Invalid storage type: %s", cfg.StorageType)
	}

	uploadHandler := handlers.NewUploadHandler(store, versionStore, db, cfg.BaseURL)
	versionHandler := handlers.NewVersionHandler(versionStore)
	downloadHandler := handlers.NewDownloadHandler(store, versionStore, db)

	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(corsMiddleware)

	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"database":  db != nil,
		})
	})

	r.Get("/api/v1/version/{channel}", versionHandler.Handle)
	r.Get("/api/v1/download/{channel}", downloadHandler.Handle)

	r.Group(func(r chi.Router) {
		r.Use(middleware.WebhookAuth(cfg.WebhookSecret))
		r.Post("/api/v1/upload", uploadHandler.Handle)
	})

	if db != nil {
		r.Get("/api/v1/stats", func(w http.ResponseWriter, r *http.Request) {
			stats, err := db.GetDownloadStats(r.Context())
			if err != nil {
				http.Error(w, "Failed to fetch stats", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(stats)
		})
	}

	log.Printf("Starting server on :%s", cfg.Port)
	log.Printf("Base URL: %s", cfg.BaseURL)
	log.Printf("Storage type: %s", cfg.StorageType)
	log.Printf("Database connected: %v", db != nil)

	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Webhook-Secret")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
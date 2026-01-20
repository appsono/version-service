package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	//server
	Port     string
	BaseURL  string

	//storage
	StorageType    string //"s3", "local", or "both"
	LocalStorePath string

	//S3 config
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3UsePathStyle    bool

	//security
	WebhookSecret string

	//metadata
	VersionsFile string

	//database
	DatabaseURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		Port:              getEnv("PORT", "8080"),
		BaseURL:           getEnv("BASE_URL", "http://localhost:8080"),
		StorageType:       getEnv("STORAGE_TYPE", "both"),
		LocalStorePath:    getEnv("LOCAL_STORE_PATH", "./data/apks"),
		S3Endpoint:        getEnv("S3_ENDPOINT", ""),
		S3Region:          getEnv("S3_REGION", "auto"),
		S3Bucket:          getEnv("S3_BUCKET", ""),
		S3AccessKeyID:     getEnv("S3_ACCESS_KEY_ID", ""),
		S3SecretAccessKey: getEnv("S3_SECRET_ACCESS_KEY", ""),
		S3UsePathStyle:    getEnvBool("S3_USE_PATH_STYLE", true),
		WebhookSecret:     getEnv("WEBHOOK_SECRET", ""),
		VersionsFile:      getEnv("VERSIONS_FILE", "./data/versions.json"),
		DatabaseURL:       getEnv("DATABASE_URL", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fallback
		}
		return b
	}
	return fallback
}
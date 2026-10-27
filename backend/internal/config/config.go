package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type MinIO struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PathStyle bool
}

type Config struct {
	Port           string
	DatabaseURL    string
	MaxUploadBytes int64
	DevOwnerID     string
	PublicOrigin   string
	MinIO          MinIO
	GroqAPIKey     string
	GroqChatModel  string
	GroqVisionModel string
}

// Load reads the .env from the current dir or its parent (repo root when running from backend/).
func Load() (Config, error) {
	for _, p := range []string{".env", "../.env"} {
		if _, err := os.Stat(p); err == nil {
			if err := godotenv.Load(p); err != nil {
				return Config{}, fmt.Errorf("load %s: %w", p, err)
			}
			break
		}
	}

	maxMB, err := strconv.Atoi(getenv("MAX_UPLOAD_MB", "20"))
	if err != nil || maxMB <= 0 {
		return Config{}, errors.New("MAX_UPLOAD_MB must be a positive integer")
	}

	cfg := Config{
		Port:            getenv("PORT", "7546"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		MaxUploadBytes:  int64(maxMB) << 20,
		DevOwnerID:      getenv("DEV_OWNER_ID", "dev-titular"),
		PublicOrigin:    getenv("PUBLIC_ORIGIN", "https://carteira.eflow.space"),
		GroqAPIKey:      os.Getenv("GROQ_API_KEY"),
		GroqChatModel:   getenv("GROQ_API_MODEL", "openai/gpt-oss-120b"),
		GroqVisionModel: getenv("GROQ_VISION_MODEL", "meta-llama/llama-4-scout-17b-16e-instruct"),
		MinIO: MinIO{
			AccessKey: os.Getenv("MINIO_ACCESS_KEY"),
			SecretKey: os.Getenv("MINIO_SECRET_KEY"),
			Bucket:    os.Getenv("MINIO_BUCKET"),
			UseSSL:    getenv("MINIO_USE_SSL", "false") == "true",
			PathStyle: getenv("MINIO_USE_PATH_STYLE_ENDPOINT", "false") == "true",
		},
	}

	// MINIO_ENDPOINT may be a bare host:port or a full URL; the scheme decides TLS.
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		cfg.MinIO.UseSSL = u.Scheme == "https"
		endpoint = u.Host
	}
	cfg.MinIO.Endpoint = endpoint

	missing := []string{}
	for name, v := range map[string]string{
		"DATABASE_URL":     cfg.DatabaseURL,
		"MINIO_ENDPOINT":   cfg.MinIO.Endpoint,
		"MINIO_ACCESS_KEY": cfg.MinIO.AccessKey,
		"MINIO_SECRET_KEY": cfg.MinIO.SecretKey,
		"MINIO_BUCKET":     cfg.MinIO.Bucket,
	} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %v", missing)
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

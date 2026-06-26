package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func Load() *Config {
	for _, path := range []string{".env", "../.env", "../../.env"} {
		_ = godotenv.Load(path)
	}

	smtpPassword := strings.ReplaceAll(os.Getenv("SMTP_PASSWORD"), " ", "")

	return &Config{
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://trello:trello@localhost:5432/trello?sslmode=disable"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:          getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		WebURL:             strings.TrimRight(getEnv("WEB_URL", "http://localhost:3000"), "/"),
		APIURL:             apiURL(),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           getEnv("SMTP_PORT", "587"),
		SMTPUser:           os.Getenv("SMTP_USER"),
		SMTPPassword:       smtpPassword,
		SMTPFrom:           smtpFrom(),
		Port:               getEnv("PORT", "8080"),
		SeedAdmin:          getEnv("SEED_ADMIN", "true") == "true",
		AdminEmail:         getEnv("ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", "admin123456"),
		AdminName:          getEnv("ADMIN_NAME", "Admin"),
	}
}

func AllowedWebOrigins(webURL string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(strings.TrimRight(s, "/"))
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, part := range strings.Split(webURL, ",") {
		add(part)
	}
	if extra := os.Getenv("ALLOWED_ORIGINS"); extra != "" {
		for _, part := range strings.Split(extra, ",") {
			add(part)
		}
	}
	add("http://localhost:3000")
	return out
}

// OriginAllowed checks CORS. On Render, any *.onrender.com origin is allowed
// so the app works before WEB_URL is configured.
func OriginAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return true
	}
	origin = strings.TrimRight(origin, "/")
	for _, a := range allowed {
		if origin == strings.TrimRight(a, "/") {
			return true
		}
	}
	if os.Getenv("RENDER_EXTERNAL_URL") != "" || os.Getenv("RENDER") == "true" {
		if strings.HasSuffix(origin, ".onrender.com") {
			return true
		}
	}
	return false
}

func smtpFrom() string {
	if from := os.Getenv("SMTP_FROM"); from != "" {
		return from
	}
	if user := os.Getenv("SMTP_USER"); user != "" {
		return user
	}
	return "noreply@localhost"
}

type Config struct {
	DatabaseURL        string
	RedisURL           string
	JWTSecret          string
	WebURL             string
	APIURL             string
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPassword       string
	SMTPFrom           string
	Port               string
	SeedAdmin          bool
	AdminEmail         string
	AdminPassword      string
	AdminName          string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func apiURL() string {
	if v := os.Getenv("API_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	if v := os.Getenv("RENDER_EXTERNAL_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:8080"
}

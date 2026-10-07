// Package config reads all runtime configuration from the environment (§54).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                 string // development | staging | production
	HTTPAddr            string
	DatabaseURL         string
	SessionSecret       string
	SessionTTL          time.Duration
	PublicURL           string   // e.g. https://eduplay.id, used for redirects and Origin checks
	AllowedOrigins      []string // extra origins allowed for state-changing requests
	CookieSecure        bool
	PaymentProvider     string // "midtrans" (default) | "mock" (automated tests only, never production)
	PaymentSecret       string // mock provider signing secret
	MidtransServerKey   string
	MidtransProduction  bool
	MonthlyPriceIDR     int64
	FreeDailyChallenges int
	FreeMaxChildren     int
	PaidMaxChildren     int
	StorageDriver       string // local | s3
	StorageDir          string
	S3Endpoint          string
	S3Region            string
	S3Bucket            string
	S3UseSSL            bool
	StorageAccessKey    string
	StorageSecretKey    string
	Timezone            *time.Location
}

func Load() (Config, error) {
	c := Config{
		Env:                 env("APP_ENV", "development"),
		HTTPAddr:            env("HTTP_ADDR", ":8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		SessionSecret:       os.Getenv("SESSION_SECRET"),
		SessionTTL:          time.Duration(envInt("SESSION_TTL_HOURS", 24*30)) * time.Hour,
		PublicURL:           strings.TrimRight(env("PUBLIC_URL", "http://localhost:5173"), "/"),
		PaymentProvider:     env("PAYMENT_PROVIDER", "midtrans"),
		MidtransServerKey:   os.Getenv("MIDTRANS_SERVER_KEY"),
		MidtransProduction:  os.Getenv("MIDTRANS_PRODUCTION") == "true",
		PaymentSecret:       os.Getenv("PAYMENT_SECRET"),
		MonthlyPriceIDR:     int64(envInt("MONTHLY_PRICE_IDR", 49000)),
		FreeDailyChallenges: envInt("FREE_DAILY_CHALLENGES", 3),
		FreeMaxChildren:     envInt("FREE_MAX_CHILDREN", 1),
		PaidMaxChildren:     envInt("PAID_MAX_CHILDREN", 5),
		StorageDriver:       env("STORAGE_DRIVER", "local"),
		StorageDir:          env("STORAGE_DIR", "./data/storage"),
		S3Endpoint:          os.Getenv("S3_ENDPOINT"),
		S3Region:            env("S3_REGION", "us-east-1"),
		S3Bucket:            os.Getenv("S3_BUCKET"),
		S3UseSSL:            env("S3_USE_SSL", "true") == "true",
		StorageAccessKey:    os.Getenv("STORAGE_ACCESS_KEY"),
		StorageSecretKey:    os.Getenv("STORAGE_SECRET_KEY"),
	}
	for _, o := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.AllowedOrigins = append(c.AllowedOrigins, strings.TrimRight(o, "/"))
		}
	}
	c.CookieSecure = strings.HasPrefix(c.PublicURL, "https://")
	tz, err := time.LoadLocation(env("APP_TIMEZONE", "Asia/Jakarta"))
	if err != nil {
		return c, fmt.Errorf("APP_TIMEZONE: %w", err)
	}
	c.Timezone = tz

	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.SessionSecret) < 32 {
		return c, fmt.Errorf("SESSION_SECRET must be at least 32 characters")
	}
	if c.IsProduction() && c.StorageDriver != "s3" {
		return c, fmt.Errorf("production requires STORAGE_DRIVER=s3")
	}
	switch c.PaymentProvider {
	case "midtrans":
		if c.IsProduction() && (c.MidtransServerKey == "" || !c.MidtransProduction) {
			return c, fmt.Errorf("production requires MIDTRANS_SERVER_KEY and MIDTRANS_PRODUCTION=true")
		}
	case "mock":
		if c.IsProduction() {
			return c, fmt.Errorf("PAYMENT_PROVIDER=mock is not allowed in production")
		}
		if c.PaymentSecret == "" {
			c.PaymentSecret = "test-payment-secret"
		}
	default:
		return c, fmt.Errorf("unknown PAYMENT_PROVIDER %q", c.PaymentProvider)
	}
	return c, nil
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

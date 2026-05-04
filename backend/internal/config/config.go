package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration loaded from environment variables.
type Config struct {
	KISAppKey      string
	KISAppSecret   string
	KISAccountNo   string
	KISAccountType string
	KISBaseURL     string
	KISHTSID       string // HTS ID — used as tr_key for 체결통보 (H0STCNI0)

	FirebaseProjectID      string // GCP project ID
	FirebaseCredentialsJSON string // service account JSON (file path or inline JSON)

	DiscordWebhookURL string // Discord incoming webhook URL for notifications
	FrontendOrigin    string // Firebase Hosting origin for CORS (e.g. https://xxx.web.app)

	ServerPort   string
	FrontendDist string
}

// Load reads .env file (if present) and returns a populated Config.
// Actual secrets must never be hardcoded; they come from the environment only.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		KISAppKey:      getEnv("KIS_APP_KEY", ""),
		KISAppSecret:   getEnv("KIS_APP_SECRET", ""),
		KISAccountNo:   getEnv("KIS_ACCOUNT_NO", ""),
		KISAccountType: getEnv("KIS_ACCOUNT_TYPE", "01"),
		KISBaseURL:     getEnv("KIS_BASE_URL", "https://openapi.koreainvestment.com:9443"),
		KISHTSID:       getEnv("KIS_HTS_ID", ""),

		FirebaseProjectID:       getEnv("FIREBASE_PROJECT_ID", ""),
		FirebaseCredentialsJSON: getEnv("FIREBASE_CREDENTIALS_JSON", ""),

		DiscordWebhookURL: getEnv("DISCORD_WEBHOOK_URL", ""),
		FrontendOrigin:    getEnv("FRONTEND_ORIGIN", ""),

		ServerPort:   getEnv("SERVER_PORT", "8080"),
		FrontendDist: getEnv("FRONTEND_DIST_PATH", "./frontend/dist"),
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

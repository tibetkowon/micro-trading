package config

import (
	"os"
	"strconv"

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

	FirebaseProjectID       string // GCP project ID
	FirebaseCredentialsJSON string // service account JSON (file path or inline JSON)
	SQLitePath              string // SQLITE_PATH env var
	AgentAPIKey             string // AGENT_API_KEY env var
	AdminAPIKey             string // ADMIN_API_KEY env var

	SlackBotToken          string // SLACK_BOT_TOKEN
	SlackDefaultChannelID  string // SLACK_DEFAULT_CHANNEL_ID
	SlackAlertChannelID    string // SLACK_ALERT_CHANNEL_ID
	SlackKISChannelID      string // SLACK_KIS_CHANNEL_ID
	SlackTradeChannelID    string // SLACK_TRADE_CHANNEL_ID
	SlackPositionChannelID string // SLACK_POSITION_CHANNEL_ID
	SlackKISFailThreshold  int    // SLACK_KIS_FAIL_THRESHOLD
	SlackPositionSnapshotM int    // SLACK_POSITION_SNAPSHOT_MIN

	FrontendOrigin string // Firebase Hosting origin for CORS (e.g. https://xxx.web.app)

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
		SQLitePath:              getEnv("SQLITE_PATH", "/data/trading.db"),
		AgentAPIKey:             getEnv("AGENT_API_KEY", ""),
		AdminAPIKey:             getEnv("ADMIN_API_KEY", ""),

		SlackBotToken:          getEnv("SLACK_BOT_TOKEN", ""),
		SlackDefaultChannelID:  getEnv("SLACK_DEFAULT_CHANNEL_ID", ""),
		SlackAlertChannelID:    getEnv("SLACK_ALERT_CHANNEL_ID", ""),
		SlackKISChannelID:      getEnv("SLACK_KIS_CHANNEL_ID", ""),
		SlackTradeChannelID:    getEnv("SLACK_TRADE_CHANNEL_ID", ""),
		SlackPositionChannelID: getEnv("SLACK_POSITION_CHANNEL_ID", ""),
		SlackKISFailThreshold:  getEnvInt("SLACK_KIS_FAIL_THRESHOLD", 5),
		SlackPositionSnapshotM: getEnvInt("SLACK_POSITION_SNAPSHOT_MIN", 30),

		FrontendOrigin: getEnv("FRONTEND_ORIGIN", ""),

		ServerPort:   getEnv("SERVER_PORT", "8080"),
		FrontendDist: getEnv("FRONTEND_DIST_PATH", "./frontend/dist"),
	}

	return cfg, nil
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

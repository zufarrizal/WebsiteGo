package config

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort            string
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetimeM int
	SessionSecret      string
	CookieSecure       bool
	CookieSameSiteMode string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppPort:            getEnv("APP_PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "127.0.0.1"),
		DBPort:             getEnv("DB_PORT", "3306"),
		DBUser:             getEnv("DB_USER", "root"),
		DBPassword:         getEnv("DB_PASSWORD", ""),
		DBName:             getEnv("DB_NAME", "websitego"),
		DBMaxOpenConns:     getEnvInt("DB_MAX_OPEN_CONNS", 50),
		DBMaxIdleConns:     getEnvInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetimeM: getEnvInt("DB_CONN_MAX_LIFETIME_MIN", 10),
		SessionSecret:      getEnv("SESSION_SECRET", "change-this-secret"),
		CookieSecure:       strings.EqualFold(getEnv("COOKIE_SECURE", "false"), "true"),
		CookieSameSiteMode: strings.ToLower(getEnv("COOKIE_SAMESITE", "lax")),
	}

	if cfg.SessionSecret == "" {
		return nil, errors.New("SESSION_SECRET cannot be empty")
	}

	return cfg, nil
}

func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func (c *Config) MigrationDSN() string {
	return "mysql://" + c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

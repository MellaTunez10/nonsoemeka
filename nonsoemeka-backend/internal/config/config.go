package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	JWT            JWTConfig
	Logging        LoggingConfig
	RateLimit      RateLimitConfig
	Seed           SeedConfig
	AllowedOrigins []string
}

type SeedConfig struct {
	AdminPassword string
	StaffPassword string
}

type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type LoggingConfig struct {
	Level  string // "debug" | "info" | "warn" | "error"
	Format string // "json" | "text"
}

type RateLimitConfig struct {
	LoginPerMinute  int
	GlobalPerMinute int
}

func Load() (*Config, error) {
	// Attempt to load .env file if available (ignore error if missing, e.g. in prod environment)
	_ = godotenv.Load()

	var errs []string

	serverPort := getEnvInt("SERVER_PORT", 8080, &errs)
	readTimeout := getEnvDuration("SERVER_READ_TIMEOUT", 10*time.Second, &errs)
	writeTimeout := getEnvDuration("SERVER_WRITE_TIMEOUT", 10*time.Second, &errs)
	shutdownTimeout := getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second, &errs)

	dbHost := getRequiredEnv("DB_HOST", &errs)
	dbPort := getEnvInt("DB_PORT", 5432, &errs)
	dbUser := getRequiredEnv("DB_USER", &errs)
	dbPassword := getRequiredEnv("DB_PASSWORD", &errs)
	dbName := getRequiredEnv("DB_NAME", &errs)
	dbMaxOpen := getEnvInt("DB_MAX_OPEN_CONNS", 25, &errs)
	dbMaxIdle := getEnvInt("DB_MAX_IDLE_CONNS", 10, &errs)
	dbConnLifetime := getEnvDuration("DB_CONN_MAX_LIFETIME", 15*time.Minute, &errs)

	jwtAccessSecret := getRequiredEnv("JWT_ACCESS_SECRET", &errs)
	jwtRefreshSecret := getRequiredEnv("JWT_REFRESH_SECRET", &errs)
	jwtAccessTTL := getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute, &errs)
	jwtRefreshTTL := getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour, &errs)

	logLevel := getEnvString("LOG_LEVEL", "info")
	logFormat := getEnvString("LOG_FORMAT", "json")

	rateLogin := getEnvInt("RATE_LIMIT_LOGIN_PER_MINUTE", 5, &errs)
	rateGlobal := getEnvInt("RATE_LIMIT_GLOBAL_PER_MINUTE", 300, &errs)

	seedAdminPassword := getRequiredEnv("SEED_ADMIN_PASSWORD", &errs)
	seedStaffPassword := getRequiredEnv("SEED_STAFF_PASSWORD", &errs)

	allowedOriginsStr := getEnvString("ALLOWED_ORIGINS", "")
	var allowedOrigins []string
	for _, origin := range strings.Split(allowedOriginsStr, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			allowedOrigins = append(allowedOrigins, trimmed)
		}
	}
	if len(allowedOrigins) == 0 {
		errs = append(errs, "ALLOWED_ORIGINS is required (comma-separated list of allowed origins, e.g. https://pos.nonsoemeka.com)")
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("configuration errors:\n  - %s", strings.Join(errs, "\n  - "))
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:            serverPort,
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
		},
		Database: DatabaseConfig{
			Host:            dbHost,
			Port:            dbPort,
			User:            dbUser,
			Password:        dbPassword,
			Name:            dbName,
			MaxOpenConns:    dbMaxOpen,
			MaxIdleConns:    dbMaxIdle,
			ConnMaxLifetime: dbConnLifetime,
		},
		JWT: JWTConfig{
			AccessSecret:  jwtAccessSecret,
			RefreshSecret: jwtRefreshSecret,
			AccessTTL:     jwtAccessTTL,
			RefreshTTL:    jwtRefreshTTL,
		},
		Logging: LoggingConfig{
			Level:  logLevel,
			Format: logFormat,
		},
		RateLimit: RateLimitConfig{
			LoginPerMinute:  rateLogin,
			GlobalPerMinute: rateGlobal,
		},
		Seed: SeedConfig{
			AdminPassword: seedAdminPassword,
			StaffPassword: seedStaffPassword,
		},
		AllowedOrigins: allowedOrigins,
	}

	return cfg, nil
}

func getRequiredEnv(key string, errs *[]string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		*errs = append(*errs, fmt.Sprintf("missing required environment variable: %s", key))
	}
	return val
}

func getEnvString(key, defaultValue string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return defaultValue
	}
	return val
}

func getEnvInt(key string, defaultValue int, errs *[]string) int {
	valStr := strings.TrimSpace(os.Getenv(key))
	if valStr == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		*errs = append(*errs, fmt.Sprintf("invalid integer for environment variable %s: %s", key, valStr))
		return defaultValue
	}
	return val
}

func getEnvDuration(key string, defaultValue time.Duration, errs *[]string) time.Duration {
	valStr := strings.TrimSpace(os.Getenv(key))
	if valStr == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(valStr)
	if err != nil {
		*errs = append(*errs, fmt.Sprintf("invalid duration for environment variable %s: %s", key, valStr))
		return defaultValue
	}
	if d <= 0 {
		*errs = append(*errs, fmt.Sprintf("duration for environment variable %s must be positive: %s", key, valStr))
		return defaultValue
	}
	return d
}

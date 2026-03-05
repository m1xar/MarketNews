package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Mode string

const (
	ModeAPI      Mode = "api"
	ModeSync     Mode = "sync"
	ModeReimport Mode = "reimport"
)

type Config struct {
	Mode                  Mode
	HTTPAddr              string
	DeltasMinutes         []int
	CronEnabled           bool
	CronSpec              string
	SyncStartDate         *time.Time
	ReimportYears         int
	UpcomingEnabled       bool
	UpcomingWindowDays    int
	UpcomingManualImport  bool
	TwelveDataAPIKey      string
	OpenAIAPIKey          string
	OpenAIModel           string
	OpenAIBaseURL         string
	OpenAIMaxOutputTokens int
}

func Load() (Config, error) {
	modeRaw, err := envRequired("APP_MODE")
	if err != nil {
		return Config{}, err
	}
	debugEnv("APP_MODE", modeRaw, "")
	mode := strings.ToLower(strings.TrimSpace(modeRaw))
	cfg := Config{
		Mode:                  Mode(mode),
		ReimportYears:         5,
		OpenAIMaxOutputTokens: envOrInt("APP_OPENAI_MAX_OUTPUT_TOKENS", 2000),
	}

	if cfg.OpenAIAPIKey == "" {
		cfg.OpenAIAPIKey = envOr("OPENAI_API_KEY", "")
	}

	if err := validateMode(cfg.Mode); err != nil {
		return cfg, err
	}

	if cfg.Mode == ModeAPI {
		cfg.HTTPAddr, err = envRequired("APP_HTTP_ADDR")
		if err != nil {
			return cfg, err
		}
		debugEnv("APP_HTTP_ADDR", cfg.HTTPAddr, "")
	}

	cfg.CronEnabled, err = envRequiredBool("APP_CRON_ENABLED")
	if err != nil {
		return cfg, err
	}
	debugEnv("APP_CRON_ENABLED", boolToStr(cfg.CronEnabled), "")
	if cfg.CronEnabled {
		cfg.CronSpec, err = envRequired("APP_CRON_SPEC")
		if err != nil {
			return cfg, err
		}
		debugEnv("APP_CRON_SPEC", cfg.CronSpec, "")
	}

	cfg.UpcomingEnabled, err = envRequiredBool("APP_UPCOMING_ENABLED")
	if err != nil {
		return cfg, err
	}
	debugEnv("APP_UPCOMING_ENABLED", boolToStr(cfg.UpcomingEnabled), "")
	if cfg.UpcomingEnabled {
		cfg.UpcomingWindowDays, err = envRequiredInt("APP_UPCOMING_DAYS")
		if err != nil {
			return cfg, err
		}
		if cfg.UpcomingWindowDays <= 0 {
			return cfg, fmt.Errorf("APP_UPCOMING_DAYS must be > 0")
		}
		debugEnv("APP_UPCOMING_DAYS", strconv.Itoa(cfg.UpcomingWindowDays), "")
	}
	cfg.UpcomingManualImport, err = envRequiredBool("APP_UPCOMING_MANUAL_IMPORT")
	if err != nil {
		return cfg, err
	}
	debugEnv("APP_UPCOMING_MANUAL_IMPORT", boolToStr(cfg.UpcomingManualImport), "")

	if cfg.CronEnabled || cfg.Mode == ModeSync || cfg.Mode == ModeReimport {
		deltasRaw, err := envRequired("APP_DELTAS")
		if err != nil {
			return cfg, err
		}
		debugEnv("APP_DELTAS", deltasRaw, "")
		deltas, err := parseIntList(deltasRaw)
		if err != nil {
			return cfg, fmt.Errorf("parse APP_DELTAS: %w", err)
		}
		cfg.DeltasMinutes = deltas
	}

	if cfg.Mode == ModeSync || cfg.Mode == ModeReimport || cfg.CronEnabled {
		cfg.TwelveDataAPIKey, err = envRequired("TWELVEDATA_API_KEY")
		if err != nil {
			return cfg, err
		}
		debugEnv("TWELVEDATA_API_KEY", cfg.TwelveDataAPIKey, "secret")
	}

	if v := strings.TrimSpace(os.Getenv("APP_SYNC_START_DATE")); v != "" {
		dt, err := time.Parse("2006-01-02", v)
		if err != nil {
			return cfg, fmt.Errorf("parse APP_SYNC_START_DATE: %w", err)
		}
		utc := time.Date(dt.Year(), dt.Month(), dt.Day(), 0, 0, 0, 0, time.UTC)
		cfg.SyncStartDate = &utc
	} else {
		start := dateOnlyUTC(time.Now().UTC()).AddDate(0, 0, -7)
		cfg.SyncStartDate = &start
	}

	if v := strings.TrimSpace(os.Getenv("APP_REIMPORT_YEARS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return cfg, fmt.Errorf("parse APP_REIMPORT_YEARS: invalid value %q", v)
		}
		cfg.ReimportYears = n
	}

	if cfg.Mode == ModeReimport {
		if v := strings.TrimSpace(os.Getenv("APP_REIMPORT_YEARS")); v == "" {
			return cfg, fmt.Errorf("APP_REIMPORT_YEARS is required for reimport mode")
		}
		debugEnv("APP_REIMPORT_YEARS", strconv.Itoa(cfg.ReimportYears), "")
	}

	aiRequired := cfg.Mode == ModeAPI || cfg.UpcomingEnabled
	if aiRequired {
		cfg.OpenAIAPIKey, err = envRequired("APP_OPENAI_API_KEY")
		if err != nil {
			return cfg, err
		}
		cfg.OpenAIModel, err = envRequired("APP_OPENAI_MODEL")
		if err != nil {
			return cfg, err
		}
		cfg.OpenAIBaseURL, err = envRequired("APP_OPENAI_BASE_URL")
		if err != nil {
			return cfg, err
		}
		debugEnv("APP_OPENAI_API_KEY", cfg.OpenAIAPIKey, "secret")
		debugEnv("APP_OPENAI_MODEL", cfg.OpenAIModel, "")
		debugEnv("APP_OPENAI_BASE_URL", cfg.OpenAIBaseURL, "")
	}

	return cfg, nil
}

func validateMode(mode Mode) error {
	switch mode {
	case ModeAPI, ModeSync, ModeReimport:
		return nil
	default:
		return fmt.Errorf("invalid APP_MODE: %s", mode)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envRequired(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		debugMissing(key)
		return "", fmt.Errorf("%s is required", key)
	}
	return strings.TrimSpace(v), nil
}

func envRequiredBool(key string) (bool, error) {
	v, err := envRequired(key)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		debugInvalid(key, v, "bool")
		return false, fmt.Errorf("%s must be a boolean", key)
	}
}

func envRequiredInt(key string) (int, error) {
	v, err := envRequired(key)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		debugInvalid(key, v, "int")
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return n, nil
}

func envOrBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func envOrInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func dateOnlyUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func parseIntList(v string) ([]int, error) {
	parts := strings.Split(v, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty list")
	}
	return out, nil
}

func debugEnv(key, value, kind string) {
	if kind == "secret" {
		fmt.Printf("env ok | %s=(redacted)\n", key)
		return
	}
	fmt.Printf("env ok | %s=%s\n", key, value)
}

func debugMissing(key string) {
	fmt.Printf("env missing | %s\n", key)
}

func debugInvalid(key, value, expected string) {
	fmt.Printf("env invalid | %s=%s | expected=%s\n", key, value, expected)
}

func boolToStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

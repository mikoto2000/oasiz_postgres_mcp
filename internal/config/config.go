package config

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/mikoto2000/oasiz_postgres_mcp/internal/metadata"
)

const (
	defaultMaxTables  = 500
	defaultMaxColumns = 200
	defaultMaxIndexes = 100
)

type Config struct {
	PostgresDSN string          `json:"postgres_dsn"`
	Metadata    metadata.Config `json:"metadata"`
}

func Load() (Config, error) {
	cfg := Config{
		PostgresDSN: os.Getenv("POSTGRES_DSN"),
		Metadata: metadata.Config{
			ExcludeSchemas: splitCSV(os.Getenv("OASIZ_EXCLUDE_SCHEMAS")),
			ExcludeTables:  splitCSV(os.Getenv("OASIZ_EXCLUDE_TABLES")),
			MaxTables:      envInt("OASIZ_MAX_TABLES", defaultMaxTables),
			MaxColumns:     envInt("OASIZ_MAX_COLUMNS", defaultMaxColumns),
			MaxIndexes:     envInt("OASIZ_MAX_INDEXES", defaultMaxIndexes),
		},
	}

	if path := os.Getenv("OASIZ_CONFIG"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return Config{}, err
		}
		if err := json.Unmarshal(b, &cfg); err != nil {
			return Config{}, err
		}
		applyDefaults(&cfg)
	}

	if strings.TrimSpace(cfg.PostgresDSN) == "" {
		return Config{}, errors.New("POSTGRES_DSN is required")
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Metadata.MaxTables <= 0 {
		cfg.Metadata.MaxTables = defaultMaxTables
	}
	if cfg.Metadata.MaxColumns <= 0 {
		cfg.Metadata.MaxColumns = defaultMaxColumns
	}
	if cfg.Metadata.MaxIndexes <= 0 {
		cfg.Metadata.MaxIndexes = defaultMaxIndexes
	}
}

func envInt(name string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func splitCSV(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

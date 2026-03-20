package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// Database
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string

	// LLM
	LLMBaseURL       string
	LLMModel         string
	LLMTemperature   float64
	LLMTopP          float64
	LLMMaxTokens     int
	LLMTimeoutSeconds int

	// Job processing
	WorkerCount        int
	BatchSize          int
	GenerationRetryMax int
	VendorRetryMax     int
	RowsToGenerate     int
	GenerationVersion  int

	// HTTP API mode (Go calls PHP instead of writing DB directly)
	UseHTTPAPI      bool
	PHPBaseURL      string
	PHPInternalToken string

	// Debug
	DryRun         bool
	LogLevelString string
}

func Load() (*Config, error) {
	// Best-effort load of .env
	_ = godotenv.Load()

	cfg := &Config{}
	var errs []string

	// Database
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.DBHost = v
	} else {
		errs = append(errs, "DB_HOST is required")
	}

	cfg.DBPort = 3306
	if v := os.Getenv("DB_PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			errs = append(errs, fmt.Sprintf("DB_PORT must be numeric: %v", err))
		} else {
			cfg.DBPort = port
		}
	}

	if v := os.Getenv("DB_USER"); v != "" {
		cfg.DBUser = v
	} else {
		errs = append(errs, "DB_USER is required")
	}

	cfg.DBPassword = os.Getenv("DB_PASSWORD")

	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.DBName = v
	} else {
		errs = append(errs, "DB_NAME is required")
	}

	// LLM
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		cfg.LLMBaseURL = v
	} else {
		errs = append(errs, "LLM_BASE_URL is required")
	}

	if v := os.Getenv("LLM_MODEL"); v != "" {
		cfg.LLMModel = v
	} else {
		errs = append(errs, "LLM_MODEL is required")
	}

	cfg.LLMTemperature = 0.4
	if v := os.Getenv("LLM_TEMPERATURE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.LLMTemperature = f
		} else {
			errs = append(errs, fmt.Sprintf("LLM_TEMPERATURE must be numeric: %v", err))
		}
	}

	cfg.LLMTopP = 0.7
	if v := os.Getenv("LLM_TOP_P"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.LLMTopP = f
		} else {
			errs = append(errs, fmt.Sprintf("LLM_TOP_P must be numeric: %v", err))
		}
	}

	cfg.LLMMaxTokens = 1024
	if v := os.Getenv("LLM_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LLMMaxTokens = n
		} else {
			errs = append(errs, fmt.Sprintf("LLM_MAX_TOKENS must be numeric: %v", err))
		}
	}

	cfg.LLMTimeoutSeconds = 60
	if v := os.Getenv("LLM_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.LLMTimeoutSeconds = n
		} else {
			errs = append(errs, fmt.Sprintf("LLM_TIMEOUT_SECONDS must be numeric: %v", err))
		}
	}

	// Job processing
	cfg.WorkerCount = 1
	if v := os.Getenv("WORKER_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.WorkerCount = n
		} else {
			errs = append(errs, fmt.Sprintf("WORKER_COUNT must be numeric: %v", err))
		}
	}

	cfg.BatchSize = 5
	if v := os.Getenv("BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.BatchSize = n
		} else {
			errs = append(errs, fmt.Sprintf("BATCH_SIZE must be numeric: %v", err))
		}
	}

	cfg.GenerationRetryMax = 2
	if v := os.Getenv("GENERATION_RETRY_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GenerationRetryMax = n
		} else {
			errs = append(errs, fmt.Sprintf("GENERATION_RETRY_MAX must be numeric: %v", err))
		}
	}

	cfg.VendorRetryMax = 2
	if v := os.Getenv("VENDOR_RETRY_MAX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.VendorRetryMax = n
		} else {
			errs = append(errs, fmt.Sprintf("VENDOR_RETRY_MAX must be numeric: %v", err))
		}
	}

	cfg.RowsToGenerate = 0
	if v := os.Getenv("ROWS_TO_GENERATE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RowsToGenerate = n
		} else {
			errs = append(errs, fmt.Sprintf("ROWS_TO_GENERATE must be numeric: %v", err))
		}
	}

	cfg.GenerationVersion = 1
	if v := os.Getenv("GENERATION_VERSION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.GenerationVersion = n
		} else {
			errs = append(errs, fmt.Sprintf("GENERATION_VERSION must be numeric: %v", err))
		}
	}

	// HTTP API mode
	cfg.UseHTTPAPI = strings.ToLower(os.Getenv("USE_HTTP_API")) == "true"
	cfg.PHPBaseURL = os.Getenv("PHP_BASE_URL")
	cfg.PHPInternalToken = os.Getenv("PHP_INTERNAL_TOKEN")

	if cfg.UseHTTPAPI {
		if cfg.PHPBaseURL == "" {
			errs = append(errs, "PHP_BASE_URL is required when USE_HTTP_API=true")
		}
		if cfg.PHPInternalToken == "" {
			errs = append(errs, "PHP_INTERNAL_TOKEN is required when USE_HTTP_API=true")
		}
	}

	// Debug
	cfg.DryRun = strings.ToLower(os.Getenv("DRY_RUN")) == "true"

	cfg.LogLevelString = "info"
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevelString = v
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("config errors: %s", strings.Join(errs, "; "))
	}

	return cfg, nil
}

func (c *Config) LogLevel() slog.Level {
	switch strings.ToLower(c.LogLevelString) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

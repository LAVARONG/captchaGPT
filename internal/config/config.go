package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port             string
	ModelName        string
	UpstreamProvider string
	UpstreamBaseURL  string
	NVIDIAAPIKey     string
	UserAPIKey       string
	RequestTimeoutS  int
	StartupSelfTest  bool
	SelfTestTimeoutS int
	EnableThinking   bool
	RateLimitRPS     float64
	RateLimitBurst   int
	MaxImageBytes    int64
	TempDir          string
	LogLevel         string
	BaseDir          string
}

func Load() (Config, error) {
	baseDir := executableDir()
	loadEnvFiles(baseDir)

	cfg := Config{
		BaseDir:          baseDir,
		Port:             getEnv("PORT", "8080"),
		ModelName:        getEnv("MODEL_NAME", "nvidia/nemotron-nano-12b-v2-vl"),
		UpstreamProvider: getEnv("UPSTREAM_PROVIDER", "nvidia"),
		UpstreamBaseURL:  getEnv("UPSTREAM_BASE_URL", "https://integrate.api.nvidia.com/v1"),
		NVIDIAAPIKey:     strings.TrimSpace(os.Getenv("NVIDIA_API_KEY")),
		UserAPIKey:       strings.TrimSpace(os.Getenv("USER_API_KEY")),
		RequestTimeoutS:  getEnvInt("REQUEST_TIMEOUT_SECONDS", 45),
		StartupSelfTest:  getEnvBool("STARTUP_SELF_TEST", true),
		SelfTestTimeoutS: getEnvInt("SELF_TEST_TIMEOUT_SECONDS", 30),
		EnableThinking:   getEnvBool("ENABLE_THINKING", false),
		RateLimitRPS:     getEnvFloat("RATE_LIMIT_RPS", 2),
		RateLimitBurst:   getEnvInt("RATE_LIMIT_BURST", 5),
		MaxImageBytes:    getEnvInt64("MAX_IMAGE_BYTES", 5*1024*1024),
		TempDir:          resolvePath(baseDir, getEnv("TEMP_IMAGE_DIR", "./tmp")),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
	}

	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	switch {
	case c.Port == "":
		return errors.New("PORT is required")
	case c.ModelName == "":
		return errors.New("MODEL_NAME is required")
	case c.UpstreamBaseURL == "":
		return errors.New("UPSTREAM_BASE_URL is required")
	case c.NVIDIAAPIKey == "":
		return errors.New("NVIDIA_API_KEY is required")
	case c.UserAPIKey == "":
		return errors.New("USER_API_KEY is required")
	case c.RequestTimeoutS <= 0:
		return errors.New("REQUEST_TIMEOUT_SECONDS must be > 0")
	case c.SelfTestTimeoutS <= 0:
		return errors.New("SELF_TEST_TIMEOUT_SECONDS must be > 0")
	case c.RateLimitRPS <= 0:
		return errors.New("RATE_LIMIT_RPS must be > 0")
	case c.RateLimitBurst <= 0:
		return errors.New("RATE_LIMIT_BURST must be > 0")
	case c.MaxImageBytes <= 0:
		return errors.New("MAX_IMAGE_BYTES must be > 0")
	case c.TempDir == "":
		return errors.New("TEMP_IMAGE_DIR is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getEnvInt64(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

func getEnvFloat(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return v
}

func getEnvBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, value)
		}
	}
	return scanner.Err()
}

func loadEnvFiles(baseDir string) {
	seen := map[string]struct{}{}
	for _, path := range candidateEnvPaths(baseDir) {
		if path == "" {
			continue
		}
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		_ = loadDotEnv(clean)
	}
}

func candidateEnvPaths(baseDir string) []string {
	paths := []string{}

	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, ".env"))
	}

	if baseDir != "" {
		paths = append(paths, filepath.Join(baseDir, ".env"))
	}

	return paths
}

func executableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		if wd, wdErr := os.Getwd(); wdErr == nil {
			return wd
		}
		return "."
	}
	return filepath.Dir(exePath)
}

func resolvePath(baseDir, value string) string {
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(baseDir, value)
}

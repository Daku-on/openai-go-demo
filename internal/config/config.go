package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	OpenAI   OpenAIConfig   `mapstructure:"openai"`
	SerpAPI  SerpAPIConfig  `mapstructure:"serpapi"`
	Graph    GraphConfig    `mapstructure:"graph"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
	Port     string `mapstructure:"port"`
	Host     string `mapstructure:"host"`
	BasePath string `mapstructure:"base_path"`
}

type OpenAIConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type SerpAPIConfig struct {
	APIKey  string `mapstructure:"api_key"`
	Enabled bool   `mapstructure:"enabled"`
}

type GraphConfig struct {
	MaxSteps int `mapstructure:"max_steps"`
	Timeout  int `mapstructure:"timeout_seconds"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load loads configuration from various sources (env vars, config files, defaults)
func Load() (*Config, error) {
	v := viper.New()
	
	// Set defaults
	setDefaults(v)
	
	// Configure viper
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("$HOME/.research-assistant")
	
	// Enable environment variable binding
	v.AutomaticEnv()
	v.SetEnvPrefix("RESEARCH")
	
	// Map environment variables to config keys
	v.BindEnv("openai.api_key", "OPENAI_API_KEY")
	v.BindEnv("serpapi.api_key", "SERPAPI_KEY")
	v.BindEnv("server.port", "PORT")
	
	// Try to read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK, we'll use env vars and defaults
	}
	
	// Try to load .env file if it exists
	loadDotEnv()
	
	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Validate required fields
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	return &config, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.base_path", "")
	
	// OpenAI defaults
	v.SetDefault("openai.model", "gpt-4o-2024-08-06")
	
	// SerpAPI defaults
	v.SetDefault("serpapi.enabled", true)
	
	// Graph defaults
	v.SetDefault("graph.max_steps", 25)
	v.SetDefault("graph.timeout_seconds", 300)
	
	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")
}

func loadDotEnv() {
	// Try to find .env file in project root
	wd, _ := os.Getwd()
	possiblePaths := []string{
		".env",
		"../.env",
		"../../.env",
		filepath.Join(wd, ".env"),
		filepath.Join(wd, "..", ".env"),
		filepath.Join(wd, "..", "..", ".env"),
	}
	
	for _, envPath := range possiblePaths {
		if _, err := os.Stat(envPath); err == nil {
			// Found .env file, set environment variables manually
			// Note: We don't use godotenv here to reduce dependencies
			// viper will handle environment variables automatically
			break
		}
	}
}

func validateConfig(config *Config) error {
	// Validate required OpenAI API key
	if config.OpenAI.APIKey == "" || config.OpenAI.APIKey == "your-api-key-here" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}
	
	// Validate server port
	if config.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}
	
	// Validate graph settings
	if config.Graph.MaxSteps <= 0 {
		return fmt.Errorf("graph max_steps must be positive")
	}
	
	if config.Graph.Timeout <= 0 {
		return fmt.Errorf("graph timeout must be positive")
	}
	
	return nil
}

// IsSerpAPIEnabled returns true if SerpAPI is configured and enabled
func (c *Config) IsSerpAPIEnabled() bool {
	return c.SerpAPI.Enabled && c.SerpAPI.APIKey != ""
}

// GetServerAddr returns the full server address
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}
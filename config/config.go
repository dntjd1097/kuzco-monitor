package config

import (
	"fmt"
	"os"
	"test/api"

	"gopkg.in/yaml.v3"
)

type KuzcoConfig struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

type VastaiConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Email             string `yaml:"email"`
	Token             string `yaml:"token"`
	IncludeVastaiCost bool   `yaml:"includeVastaiCost"`
}

type AlertConfig struct {
	// MinInstanceCount int     `yaml:"minInstanceCount"`
	// MinCredit        float64 `yaml:"minCredit"`
	Enabled bool `yaml:"enabled"`
}

type TelegramThreads struct {
	Daily   int `yaml:"daily"`
	Hourly  int `yaml:"hourly"`
	Error   int `yaml:"error"`
	Status  int `yaml:"status"`
	Workers int `yaml:"workers"`
}

type TelegramConfig struct {
	Token   string          `yaml:"token"`
	ChatID  string          `yaml:"chat_id"`
	Threads TelegramThreads `yaml:"threads"`
}

type AccountConfig struct {
	Name   string          `yaml:"name"`
	Kuzco  KuzcoConfig     `yaml:"kuzco"`
	Vastai VastaiConfig    `yaml:"vastai"`
	Alerts api.AlertConfig `yaml:"alerts"`
}

type Config struct {
	Accounts []AccountConfig `yaml:"accounts"`
	Telegram TelegramConfig  `yaml:"telegram"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

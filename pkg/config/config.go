package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func LoadConfig[T any]() (*T, error) {
	filename, err := fileConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	data, err := os.ReadFile(*filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config T
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func fileConfig() (*string, error) {
	fileConfig := os.Getenv("FILE_CONFIG")
	var err error
	if fileConfig == "" {
		fileConfig, err = fullPath("config.json")
		if err != nil {
			return nil, err
		}
	}
	return &fileConfig, nil
}

func fullPath(baseDir string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	return filepath.Join(filepath.Dir(exePath), baseDir), nil
}

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the persisted CLI configuration. The access key and context token
// are secrets and the file is written with mode 0600.
type Config struct {
	BaseURL      string `json:"baseUrl,omitempty"`
	AccessKey    string `json:"accessKey,omitempty"`
	ContextToken string `json:"contextToken,omitempty"`
}

// DefaultConfigPath returns $XDG_CONFIG_HOME/strombauer/config.json or the
// platform equivalent.
func DefaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("determine user config directory: %w", err)
	}
	return filepath.Join(dir, "strombauer", "config.json"), nil
}

// LoadConfig reads path. A missing file is not an error and yields an empty
// configuration.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, errors.New("strombauer: config path is empty")
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	config := &Config{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return config, nil
}

// SaveConfig writes path atomically with mode 0600.
func SaveConfig(path string, config *Config) error {
	if path == "" {
		return errors.New("strombauer: config path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	temp, err := os.CreateTemp(dir, ".config-*.json")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tempName := temp.Name()
	defer func() {
		_ = os.Remove(tempName)
	}()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	return nil
}

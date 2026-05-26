package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath = "~/.substrate/config.yaml"
	DefaultDBPath     = "~/.substrate/substrate.db"
	DefaultGitRepoDir = "~/.substrate/repo"
	DefaultServerPort = 7777
)

// Config holds Substrate runtime configuration.
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Storage   StorageConfig   `yaml:"storage"`
	LLM       LLMConfig       `yaml:"llm"`
	Synthesis SynthesisConfig `yaml:"synthesis"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type StorageConfig struct {
	DBPath     string `yaml:"db_path"`
	GitRepoDir string `yaml:"git_repo_dir"`
}

type LLMConfig struct {
	Backend   string `yaml:"backend"`
	Model     string `yaml:"model"`
	Endpoint  string `yaml:"endpoint"`
	APIKeyEnv string `yaml:"api_key_env"`
}

type SynthesisConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Default returns a config with documented defaults.
func Default() Config {
	return Config{
		Server: ServerConfig{Port: DefaultServerPort},
		Storage: StorageConfig{
			DBPath:     DefaultDBPath,
			GitRepoDir: DefaultGitRepoDir,
		},
		LLM: LLMConfig{
			Backend:   "claude_api",
			Model:     "claude-opus-4-7",
			APIKeyEnv: "ANTHROPIC_API_KEY",
		},
		Synthesis: SynthesisConfig{Enabled: false},
	}
}

// ResolvePath returns the config file path from env or default.
func ResolvePath(override string) (string, error) {
	if override != "" {
		return expandHome(override)
	}
	if env := os.Getenv("SUBSTRATE_CONFIG"); env != "" {
		return expandHome(env)
	}
	return expandHome(DefaultConfigPath)
}

// Load reads configuration from path. Missing file yields defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	path, err := expandHome(path)
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

// Save writes configuration to path, creating parent directories.
func Save(path string, cfg Config) error {
	path, err := expandHome(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	cfg.applyDefaults()
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ExpandStoragePaths resolves ~ in storage paths.
func (c *Config) ExpandStoragePaths() error {
	var err error
	c.Storage.DBPath, err = expandHome(c.Storage.DBPath)
	if err != nil {
		return err
	}
	c.Storage.GitRepoDir, err = expandHome(c.Storage.GitRepoDir)
	return err
}

func (c *Config) applyDefaults() {
	d := Default()
	if c.Server.Port == 0 {
		c.Server.Port = d.Server.Port
	}
	if c.Storage.DBPath == "" {
		c.Storage.DBPath = d.Storage.DBPath
	}
	if c.Storage.GitRepoDir == "" {
		c.Storage.GitRepoDir = d.Storage.GitRepoDir
	}
	if c.LLM.Backend == "" {
		c.LLM.Backend = d.LLM.Backend
	}
	if c.LLM.Model == "" {
		c.LLM.Model = d.LLM.Model
	}
	if c.LLM.APIKeyEnv == "" {
		c.LLM.APIKeyEnv = d.LLM.APIKeyEnv
	}
}

func expandHome(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

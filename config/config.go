package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type ServerConfig struct {
	Port int    `toml:"port"`
	Host string `toml:"host"`
}

type RoutesConfig struct {
	Prefixes map[string]string `toml:"prefixes"`
}

type GeminiConfig struct {
	URLTemplate string   `toml:"url_template"`
	Browser     string   `toml:"browser"`
	BrowserArgs []string `toml:"browser_args"`
}

type OpenCodeConfig struct {
	Binary       string   `toml:"binary"`
	Terminal     string   `toml:"terminal"`
	TerminalArgs []string `toml:"terminal_args"`
	ExtraArgs    []string `toml:"extra_args"`
	Dir          string   `toml:"dir"`
	ServerURL    string   `toml:"server_url"`
	ServeDir     string   `toml:"serve_dir"`
}

type LoggingConfig struct {
	Level string `toml:"level"`
}

type EnhanceConfig struct {
	Enabled     bool   `toml:"enabled"`
	Endpoint    string `toml:"endpoint"`
	Model       string `toml:"model"`
	APIKey      string `toml:"api_key"`
	Paste       bool   `toml:"paste"`
	PasteBinary string `toml:"paste_binary"`
	KeyDelay    int    `toml:"key_delay"`
}

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Routes   RoutesConfig   `toml:"routes"`
	Gemini   GeminiConfig   `toml:"gemini"`
	OpenCode OpenCodeConfig `toml:"opencode"`
	Enhance  EnhanceConfig  `toml:"enhance"`
	Logging  LoggingConfig  `toml:"logging"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 11341,
			Host: "127.0.0.1",
		},
		Routes: RoutesConfig{
			Prefixes: map[string]string{
				"transcribe": "enhance",
				"gemini":     "gemini",
				"chat":       "code",
			},
		},
		Gemini: GeminiConfig{
			URLTemplate: "https://gemini.google.com/app?q={query}",
			Browser:     "",
			BrowserArgs: []string{},
		},
		OpenCode: OpenCodeConfig{
			Binary:       "opencode",
			Terminal:     "xterm",
			TerminalArgs: []string{"-e"},
			ExtraArgs:    []string{},
			Dir:          "~/Projects/OpenCode",
			ServerURL:    "http://127.0.0.1:11342",
			ServeDir:     "~/Projects/OpenCode",
		},
		Logging: LoggingConfig{
			Level: "info",
		},
		Enhance: EnhanceConfig{
			Enabled:     true,
			Endpoint:    "https://opencode.ai/zen/go/v1/chat/completions",
			Model:       "deepseek-v4-flash",
			APIKey:      "",
			Paste:       true,
			PasteBinary: "ydotool",
			KeyDelay:    1,
		},
	}
}

func findConfigFile() string {
	// 1. Current directory
	if _, err := os.Stat("config.toml"); err == nil {
		return "config.toml"
	}

	// 2. XDG_CONFIG_HOME
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			xdgConfig = filepath.Join(home, ".config")
		}
	}
	if xdgConfig != "" {
		xdgPath := filepath.Join(xdgConfig, "handy-router", "config.toml")
		if _, err := os.Stat(xdgPath); err == nil {
			return xdgPath
		}
	}

	return ""
}

func Load() (*Config, error) {
	configPath := flag.String("config", "", "path to config file")
	port := flag.Int("port", 0, "override server port")
	host := flag.String("host", "", "override server host")
	flag.Parse()

	cfg := DefaultConfig()

	path := *configPath
	if path == "" {
		path = findConfigFile()
	}

	if path != "" {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config %s: %w", path, err)
		}
		fmt.Printf("Loaded config from %s\n", path)
	}

	// CLI overrides
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *host != "" {
		cfg.Server.Host = *host
	}

	return cfg, nil
}

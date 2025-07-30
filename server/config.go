package server

import "time"

// Config represents the server configuration.
type Config struct {
	Server ServerConfig `mapstructure:"server" yaml:"server" json:"server"`
	Auth   AuthConfig   `mapstructure:"auth" yaml:"auth" json:"auth"`
}

// ServerConfig contains server-specific configuration.
type ServerConfig struct {
	MaxConnections int           `mapstructure:"max_connections" yaml:"max_connections" json:"max_connections"`
	RequestTimeout time.Duration `mapstructure:"request_timeout" yaml:"request_timeout" json:"request_timeout"`
	EnableMCP      bool          `mapstructure:"enable_mcp" yaml:"enable_mcp" json:"enable_mcp"`
	EnableCORS     bool          `mapstructure:"enable_cors" yaml:"enable_cors" json:"enable_cors"`
}

// AuthConfig contains authentication configuration.
type AuthConfig struct {
	EnableOAuth bool   `mapstructure:"enable_oauth" yaml:"enable_oauth" json:"enable_oauth"`
	EnableAPIKey bool   `mapstructure:"enable_api_key" yaml:"enable_api_key" json:"enable_api_key"`
	APIKeyHeader string `mapstructure:"api_key_header" yaml:"api_key_header" json:"api_key_header"`
}
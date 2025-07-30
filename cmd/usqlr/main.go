// Command usqlr is the server version of usql with MCP (Model Context Protocol) support.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xo/usql/server"
	
	// Import all database drivers (same as usql)
	_ "github.com/xo/usql/internal"
)

func main() {
	if err := New().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// New creates the usqlr command.
func New() *cobra.Command {
	var configFile string
	var addr string
	var port int

	cmd := &cobra.Command{
		Use:           "usqlr",
		Short:         "usqlr is the server version of usql with MCP support",
		Long:          "usqlr transforms usql from a CLI tool into a server that supports multiple database connections and exposes MCP capabilities for AI integration.",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(configFile, addr, port)
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "config file path")
	cmd.Flags().StringVarP(&addr, "addr", "a", "0.0.0.0", "server listening address")
	cmd.Flags().IntVarP(&port, "port", "p", 8080, "server listening port")

	return cmd
}

func run(configFile, addr string, port int) error {

	// Load configuration
	config, err := loadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create server
	srv, err := server.New(config)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		cancel()

		// Give server 30 seconds to shutdown gracefully
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	// Start server
	log.Printf("Starting usqlr server on %s:%d", addr, port)
	return srv.Listen(ctx, fmt.Sprintf("%s:%d", addr, port))
}

func loadConfig(configFile string) (*server.Config, error) {
	v := viper.New()
	
	// Set defaults
	v.SetDefault("server.max_connections", 100)
	v.SetDefault("server.request_timeout", "30s")
	v.SetDefault("server.enable_mcp", true)
	v.SetDefault("server.enable_cors", true)

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	// Environment variables
	v.AutomaticEnv()
	v.SetEnvPrefix("USQLR")

	var config server.Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
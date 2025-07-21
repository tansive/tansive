package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/server"
	"github.com/tansive/tansive/internal/tangent/session"
	"github.com/tansive/tansive/internal/tangent/session/mcpservice"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"

	"github.com/rs/zerolog/log"
)

func init() {
	tangentcommon.InitLogger()
}

type cmdoptions struct {
	configFile string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := run(ctx); err != nil {
		log.Error().Err(err).Msg("server failed")
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	slog := log.With().Str("state", "init").Logger()

	opt := parseFlags()

	slog.Info().Str("config_file", opt.configFile).Msg("loading config file")
	if err := config.LoadConfig(opt.configFile); err != nil {
		return fmt.Errorf("loading config file: %w", err)
	}
	if config.Config().ServerPort == "" {
		return fmt.Errorf("server port not defined")
	}
	if err := config.RegisterTangent(); err != nil {
		return fmt.Errorf("registering tangent: %w", err)
	}
	session.Init()

	// Start the tangent server
	serverErrors, shutdownTangent, err := createTangentServer(ctx)
	if err != nil {
		return fmt.Errorf("creating tangent server: %w", err)
	}

	// Start the MCP server
	mcpErrors, shutdownMCP, err := createMCPServer(ctx)
	if err != nil {
		return fmt.Errorf("creating MCP server: %w", err)
	}

	// Start the skill service
	skillService, err := session.CreateSkillService()
	if err != nil {
		return fmt.Errorf("creating skill service: %w", err)
	}

	// Channel to listen for an interrupt or terminate signal from the OS.
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Wait forever until shutdown
	select {
	case err := <-serverErrors:
		skillService.StopServer()
		shutdownMCP()
		return fmt.Errorf("server error: %w", err)

	case err := <-mcpErrors:
		skillService.StopServer()
		shutdownTangent()
		return fmt.Errorf("mcp server error: %w", err)

	case sig := <-shutdown:
		slog.Info().Str("signal", sig.String()).Msg("shutdown signal received")
		skillService.StopServer()
		shutdownMCP()
		shutdownTangent()
	}

	slog.Info().Msg("server stopped")
	return nil
}

func createTangentServer(ctx context.Context) (chan error, func(), error) {
	slog := log.With().Str("state", "init").Logger()
	s, err := server.CreateNewServer()
	if err != nil {
		return nil, nil, fmt.Errorf("creating server: %w", err)
	}
	s.MountHandlers()

	srv := &http.Server{
		Addr:              ":" + config.Config().ServerPort,
		Handler:           s.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		if config.Config().SupportTLS {
			slog.Info().Str("port", config.Config().ServerPort).Msg("server started with TLS")

			// Create TLS config from PEM certificates
			tlsConfig, err := createTLSConfig()
			if err != nil {
				serverErrors <- fmt.Errorf("creating TLS config: %w", err)
				return
			}

			// Create listener with TLS
			listener, err := tls.Listen("tcp", srv.Addr, tlsConfig)
			if err != nil {
				serverErrors <- fmt.Errorf("creating TLS listener: %w", err)
				return
			}

			serverErrors <- srv.Serve(listener)
		} else {
			slog.Info().Str("port", config.Config().ServerPort).Msg("server started")
			serverErrors <- srv.ListenAndServe()
		}
	}()

	shutdown := func() {
		// Give outstanding requests 5 seconds to complete and initiate the shutdown.
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error().Err(err).Msg("could not stop server gracefully")
			if err := srv.Close(); err != nil {
				slog.Error().Err(err).Msg("could not stop server")
			}
		}
	}

	return serverErrors, shutdown, nil
}

func createMCPServer(ctx context.Context) (chan error, func(), error) {
	slog := log.With().Str("state", "init").Logger()
	s, err := mcpservice.CreateMCPService()
	if err != nil {
		return nil, nil, fmt.Errorf("creating server: %w", err)
	}

	srv := &http.Server{
		Addr:              ":" + config.Config().MCP.Port,
		Handler:           s.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		slog.Info().Str("port", config.Config().MCP.Port).Msg("mcp server started")
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := func() {
		// Give outstanding requests 5 seconds to complete and initiate the shutdown.
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error().Err(err).Msg("could not stop server gracefully")
		}
	}

	return serverErrors, shutdown, nil
}

// createTLSConfig creates a TLS configuration from the PEM certificates in the config
func createTLSConfig() (*tls.Config, error) {
	cfg := config.Config()

	cert, err := tls.X509KeyPair(cfg.TLSCertPEM, cfg.TLSKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parsing TLS certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}

const DefaultConfigFile = "/etc/tansive/tangent.conf"

func parseFlags() cmdoptions {
	var opt cmdoptions
	flag.StringVar(&opt.configFile, "config", DefaultConfigFile, "Path to the config file")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n\n", os.Args[0])
		fmt.Println("Options:")
		flag.PrintDefaults()
	}
	flag.Parse()
	return opt
}

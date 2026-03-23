package main

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/HT4w5/socks5/cmd/socks5s/meta"
	"github.com/HT4w5/socks5/pkg/log"
	"github.com/HT4w5/socks5/pkg/method"
	"github.com/HT4w5/socks5/pkg/resolver"
	"github.com/HT4w5/socks5/pkg/server"
	"github.com/spf13/pflag"
)

// Config holds all configuration options
type Config struct {
	// Server configuration
	ListenAddr           string
	LogLevel             string
	TCPBufferSize        int
	UDPBufferSize        int
	UDPNATBehavior       string
	UDPNATMapSize        int
	TCPKeepAlive         bool
	TCPKeepAliveIdle     time.Duration
	TCPKeepAliveInterval time.Duration
	TCPKeepAliveCount    int

	// Resolver configuration
	ResolverTTL    time.Duration
	ResolverMaxLen int

	// Methods
	Methods []string
}

// parseUDPNATBehavior converts string to UDPNATBehavior
func parseUDPNATBehavior(behavior string) server.UDPNATBehavior {
	switch strings.ToLower(behavior) {
	case "endpoint-independent":
		return server.EndpointIndependent
	case "address-dependent":
		return server.AddressDependent
	case "address-and-port-dependent":
		fallthrough
	default:
		return server.AddressAndPortDependent
	}
}

// parseLogLevel converts string to log.Level
func parseLogLevel(level string) log.Level {
	switch strings.ToLower(level) {
	case "none":
		return log.None
	case "error":
		return log.Error
	case "warn":
		return log.Warn
	case "info":
		return log.Info
	case "debug":
		fallthrough
	default:
		return log.Debug
	}
}

func main() {
	var config Config

	pflag.StringVarP(&config.ListenAddr, "addr", "a", "0.0.0.0:1080", "listen address (host:port)")
	pflag.StringVarP(&config.LogLevel, "log-level", "l", "info", "log level (none, error, warn, info, debug)")
	pflag.IntVar(&config.TCPBufferSize, "tcp-buffer-size", 32*1024, "TCP buffer size in bytes")
	pflag.IntVar(&config.UDPBufferSize, "udp-buffer-size", 1500-20-8, "UDP buffer size in bytes")
	pflag.StringVar(
		&config.UDPNATBehavior,
		"udp-nat-behavior",
		"address-and-port-dependent",
		"UDP NAT behavior (endpoint-independent, address-dependent, address-and-port-dependent)",
	)
	pflag.IntVar(&config.UDPNATMapSize, "udp-nat-map-size", 128, "UDP NAT map size")
	pflag.BoolVar(&config.TCPKeepAlive, "tcp-keepalive", true, "enable TCP keep-alive")
	pflag.DurationVar(&config.TCPKeepAliveIdle, "tcp-keepalive-idle", 0, "TCP keep-alive idle time")
	pflag.DurationVar(&config.TCPKeepAliveInterval, "tcp-keepalive-interval", 0, "TCP keep-alive interval")
	pflag.IntVar(&config.TCPKeepAliveCount, "tcp-keepalive-count", 0, "TCP keep-alive count")

	pflag.DurationVar(&config.ResolverTTL, "resolver-ttl", 10*time.Minute, "DNS resolver cache TTL")
	pflag.IntVar(&config.ResolverMaxLen, "resolver-max-len", 64, "DNS resolver cache maximum entries")

	pflag.StringSliceVarP(
		&config.Methods,
		"methods",
		"m",
		[]string{"noauth"},
		"Socks5 methods (comma-separated: noauth)",
	)

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\n", meta.VersionLong)
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		pflag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s --addr :1080 --log-level debug\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -a 127.0.0.1:1081 -l info --tcp-buffer-size 65536\n", os.Args[0])
	}

	pflag.Parse()

	// Print banner
	fmt.Printf("%s\n", meta.VersionLong)

	// Setup logger
	logLevel := parseLogLevel(config.LogLevel)
	logger := log.NewAsyncLogger(log.WithLevel(logLevel))
	defer logger.Close()

	// Validate and normalize address
	addrStr := config.ListenAddr
	if strings.HasPrefix(addrStr, ":") {
		addrStr = "0.0.0.0" + addrStr
	} else if !strings.Contains(addrStr, ":") {
		// If no port specified, add default port
		addrStr = addrStr + ":1080"
	}

	_, err := netip.ParseAddrPort(addrStr)
	if err != nil {
		logger.Errorf("invalid address format: %v\n", err)
		os.Exit(1)
	}
	config.ListenAddr = addrStr

	// Setup socks5 methods
	negotiator := method.New(method.WithLogger(logger))
	for _, authMethod := range config.Methods {
		switch strings.ToLower(authMethod) {
		case "noauth":
			negotiator = method.New(
				method.WithMethod(&method.NoAuthHandler{}),
				method.WithLogger(logger),
			)
		case "username-password":
			// TODO: implement username/password authentication
			fallthrough
		default:
			logger.Warnf("unknown authentication method: %s, using noauth", authMethod)
			negotiator = method.New(
				method.WithMethod(&method.NoAuthHandler{}),
				method.WithLogger(logger),
			)
		}
	}

	// Setup resolver
	res := resolver.NewCachedResolver(
		resolver.WithTTL(config.ResolverTTL),
		resolver.WithMaxLen(config.ResolverMaxLen),
	)

	// Create server options slice
	var serverOpts []func(*server.Server)

	// Add common options
	serverOpts = append(serverOpts,
		server.WithLogger(logger),
		server.WithNegotiator(negotiator),
		server.WithResolver(res),
		server.WithTCPByteBufferSize(config.TCPBufferSize),
		server.WithUDPByteBufferSize(config.UDPBufferSize),
		server.WithUDPNATBehavior(parseUDPNATBehavior(config.UDPNATBehavior)),
	)

	// Configure TCP keep-alive if enabled
	if config.TCPKeepAlive {
		keepAliveConfig := net.KeepAliveConfig{
			Enable:   true,
			Idle:     config.TCPKeepAliveIdle,
			Interval: config.TCPKeepAliveInterval,
			Count:    config.TCPKeepAliveCount,
		}
		serverOpts = append(serverOpts, server.WithTCPKeepAliveConfig(keepAliveConfig))
	}

	srv := server.New(serverOpts...)

	// Log configuration
	logger.Infof("Starting SOCKS5 server with configuration:")
	logger.Infof("%+v", config)
	// Setup signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Parse address and start server
	addr, err := netip.ParseAddrPort(config.ListenAddr)
	if err != nil {
		logger.Errorf("failed to parse address: %v", err)
		os.Exit(1)
	}

	err = srv.ListenAndServe(ctx, addr)
	if err != nil {
		logger.Errorf("failed to listen: %v", err)
		os.Exit(1)
	}
}

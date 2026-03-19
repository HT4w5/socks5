package main

import (
	"context"
	"net/netip"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HT4w5/socks5/pkg/log"
	"github.com/HT4w5/socks5/pkg/method"
	"github.com/HT4w5/socks5/pkg/resolver"
	"github.com/HT4w5/socks5/pkg/server"
)

func main() {
	logger := log.NewStdoutLogger(log.WithLevel(log.Debug))
	srv := server.New(
		server.WithLogger(logger),
		server.WithNegotiator(method.New(method.WithMethod(&method.NoAuthHandler{}), method.WithLogger(logger))),
		server.WithResolver(resolver.NewCachedResolver(resolver.WithTTL(10*time.Minute))),
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := srv.ListenAndServe(ctx, netip.MustParseAddrPort("0.0.0.0:1080"))
	if err != nil {
		logger.Errorf("failed to listen: %v", err)
		os.Exit(1)
	}
}

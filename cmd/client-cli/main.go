package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	api "github.com/gcleroux/projet-A23/api/v1"
	"github.com/gcleroux/projet-A23/src/config"
	// "github.com/gcleroux/projet-A23/src/loadbalance"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
)

var (
	rootCmd = &cobra.Command{
		Use:   "client",
		Short: "gRPC client gateway to send REST requests to the gRPC server",
		Run: func(cmd *cobra.Command, args []string) {
			setupInterruptHandler()
			if err := run(); err != nil {
				grpclog.Fatal(err)
			}
		},
	}

	ctx context.Context
)

func init() {
	if err := config.InitializeConfig(rootCmd); err != nil {
		grpclog.Fatal(err)
	}
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		grpclog.Fatal(err)
	}
}

func run() error {
	conf, err := config.LoadConfig()
	if err != nil {
		return err
	}

	ctx = context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()

	// The leader will always be the first server
	s := conf.Servers[0]

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		KeyFile:       conf.Certs.UserKeyFile,
		CertFile:      conf.Certs.UserCertFile,
		CAFile:        conf.Certs.CAFile,
		Server:        false,
		ServerAddress: s.Address,
	})
	if err != nil {
		return err
	}

	clientCreds := credentials.NewTLS(clientTLSConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(clientCreds)}

	// This is what sets up a gRPC-gateway in order to send REST requests to the server
	// conn := fmt.Sprintf("%s:///%s:%d", loadbalance.Name, conf.Client.)
	err = api.RegisterLogHandlerFromEndpoint(ctx, mux, conf.Client.ConnectedServer, opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", conf.Client.GatewayPort), mux)
}

func setupInterruptHandler() {
	// Set up a channel to receive interrupt signals
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		// Wait for an interrupt signal
		<-interruptChan
		_, cancel := context.WithCancel(ctx)
		cancel()

		os.Exit(0)
	}()
}

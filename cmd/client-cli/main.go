package main

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/gcleroux/projet-ift605/api/v1"
	"github.com/gcleroux/projet-ift605/src/config"
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
			if err := run(); err != nil {
				grpclog.Fatal(err)
			}
		},
	}
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

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()

	clientTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CAFile: conf.Certs.CAFile,
	})
	if err != nil {
		return err
	}

	clientCreds := credentials.NewTLS(clientTLSConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(clientCreds)}

	err = api.RegisterLogHandlerFromEndpoint(ctx, mux, conf.Server.Address, opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", conf.Client.GatewayPort), mux)
}

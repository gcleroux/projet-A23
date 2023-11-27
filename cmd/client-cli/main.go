package main

import (
	"context"
	"fmt"
	"net/http"

	api "github.com/gcleroux/projet-ift605/api/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/grpclog"
)

var (
	serverAddress string
	gatewayPort   int

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
	rootCmd.Flags().StringVarP(&serverAddress, "server-address", "s", "localhost:50051", "Server address")
	rootCmd.Flags().IntVarP(&gatewayPort, "gateway-port", "p", 8080, "Gateway port")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		grpclog.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err := api.RegisterLogHandlerFromEndpoint(ctx, mux, serverAddress, opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(fmt.Sprintf(":%d", gatewayPort), mux)
}

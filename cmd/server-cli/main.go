package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	api "github.com/gcleroux/projet-A23/api/v1"
	"github.com/gcleroux/projet-A23/src/agent"
	"github.com/gcleroux/projet-A23/src/config"
	dlog "github.com/gcleroux/projet-A23/src/distributedLog"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/cobra"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
)

// Cobra command
var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "Simple gRPC server for distributed logs",
	Run: func(cmd *cobra.Command, args []string) {
		if err := run(); err != nil {
			grpclog.Fatal(err)
		}
	},
}

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
	var teardown []func() error

	// Load configuration from file
	conf, err := config.LoadConfig()
	if err != nil {
		return err
	}

	for _, s := range conf.Servers {

		if err := os.MkdirAll(s.LogDirectory, os.ModePerm); err != nil {
			return err
		}

		serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile:      conf.Certs.ServerCertFile,
			KeyFile:       conf.Certs.ServerKeyFile,
			CAFile:        conf.Certs.CAFile,
			ServerAddress: s.Address,
			Server:        true,
		})
		if err != nil {
			return err
		}
		peerTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile:      conf.Certs.UserCertFile,
			KeyFile:       conf.Certs.UserKeyFile,
			CAFile:        conf.Certs.CAFile,
			Server:        false,
			ServerAddress: s.Address,
		})
		if err != nil {
			return err
		}

		c := agent.Config{
			Bootstrap:       s.Bootstrap,
			NodeName:        s.NodeName,
			StartJoinAddrs:  s.JoinAddr,
			BindAddr:        fmt.Sprintf("%s:%d", s.Address, s.SerfPort),
			RPCPort:         s.RPCPort,
			DataDir:         s.LogDirectory,
			ACLModelFile:    conf.Certs.ACLModelFile,
			ACLPolicyFile:   conf.Certs.ACLPolicyFile,
			ServerTLSConfig: serverTLSConfig,
			PeerTLSConfig:   peerTLSConfig,
		}
		if s.Bootstrap {
			serverInfo := make(map[string]dlog.ServerInfo)
			for _, s := range conf.Servers {
				serverInfo[s.NodeName] = dlog.ServerInfo{
					s.Latitude,
					s.Longitude,
					s.GatewayPort,
				}
			}
			c.ServerInfo = serverInfo
		}

		agent, err := agent.New(c)
		if err != nil {
			return err
		}
		mux, fn, err := setupGateway(peerTLSConfig, fmt.Sprintf("%s:%d", s.Address, s.RPCPort))
		if err != nil {
			return err
		}

		go http.ListenAndServe(fmt.Sprintf(":%d", s.GatewayPort), mux)
		teardown = append(teardown, fn, agent.Shutdown)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc
	for _, f := range teardown {
		_ = f()
	}
	return nil
}

func setupGateway(cc *tls.Config, RPCaddr string) (*runtime.ServeMux, func() error, error) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	mux := runtime.NewServeMux()

	clientCreds := credentials.NewTLS(cc)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(clientCreds)}

	// This is what sets up a gRPC-gateway in order to send REST requests to the server
	// conn := fmt.Sprintf("%s:///%s:%d", loadbalance.Name, conf.Client.)
	err := api.RegisterLogHandlerFromEndpoint(ctx, mux, RPCaddr, opts)
	if err != nil {
		return nil, func() error { cancel(); return nil }, err
	}
	return mux, func() error { cancel(); return nil }, nil
}

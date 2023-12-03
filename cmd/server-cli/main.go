package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gcleroux/projet-ift605/src/config"
	"github.com/gcleroux/projet-ift605/src/log"
	"github.com/gcleroux/projet-ift605/src/server"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"
)

var (
	lis  net.Listener
	clog *log.Log
	srv  *grpc.Server

	// Cobra command
	rootCmd = &cobra.Command{
		Use:   "server",
		Short: "Simple gRPC server for distributed logs",
		Run: func(cmd *cobra.Command, args []string) {
			setupInterruptHandler()
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
	// Load configuration from file
	conf, err := config.LoadConfig()
	if err != nil {
		return err
	}

	lis, err = net.Listen("tcp", conf.Server.Address)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(conf.Server.LogDirectory, os.ModePerm); err != nil {
		return err
	}

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      conf.Certs.ServerCertFile,
		KeyFile:       conf.Certs.ServerKeyFile,
		CAFile:        conf.Certs.CAFile,
		ServerAddress: lis.Addr().String(),
	})
	if err != nil {
		return err
	}

	serverCreds := credentials.NewTLS(serverTLSConfig)

	clog, err = log.NewLog(conf.Server.LogDirectory, log.Config{
		Segment: struct {
			MaxStoreBytes uint64
			MaxIndexBytes uint64
			InitialOffset uint64
		}{
			MaxStoreBytes: conf.Server.MaxStoreBytes,
			MaxIndexBytes: conf.Server.MaxIndexBytes,
			InitialOffset: 0,
		},
	})
	if err != nil {
		return err
	}

	cfg := &server.Config{
		CommitLog: clog,
	}

	srv, err = server.NewGRPCServer(cfg, grpc.Creds(serverCreds))
	if err != nil {
		return err
	}

	// Shouldn't happen since the program will never shut down
	defer lis.Close()
	defer srv.Stop()
	defer clog.Close()

	return srv.Serve(lis)
}

func setupInterruptHandler() {
	// Set up a channel to receive interrupt signals
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		// Wait for an interrupt signal
		<-interruptChan
		clog.Close()
		srv.Stop()
		lis.Close()

		os.Exit(0)
	}()
}

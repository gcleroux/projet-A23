package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gcleroux/projet-ift605/pkg/log"
	"github.com/gcleroux/projet-ift605/pkg/server"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
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

	// Cobra flags
	maxStoreBytes uint64
	maxIndexBytes uint64
	logDirectory  string
	serverPort    int
)

func init() {
	rootCmd.Flags().Uint64VarP(&maxStoreBytes, "max-store-bytes", "s", 1024, "Maximum store bytes")
	rootCmd.Flags().Uint64VarP(&maxIndexBytes, "max-index-bytes", "i", 1024, "Maximum index bytes")
	rootCmd.Flags().StringVarP(&logDirectory, "directory", "d", "./log", "Log directory")
	rootCmd.Flags().IntVarP(&serverPort, "port", "p", 50051, "Server port")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		grpclog.Fatal(err)
	}
}

func run() error {
	var err error

	lis, err = net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(logDirectory, os.ModePerm); err != nil {
		return err
	}

	clog, err = log.NewLog(logDirectory, log.Config{
		Segment: struct {
			MaxStoreBytes uint64
			MaxIndexBytes uint64
			InitialOffset uint64
		}{
			MaxStoreBytes: maxStoreBytes,
			MaxIndexBytes: maxIndexBytes,
			InitialOffset: 0,
		},
	})
	if err != nil {
		return err
	}

	cfg := &server.Config{
		CommitLog: clog,
	}

	srv, err = server.NewGRPCServer(cfg)
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

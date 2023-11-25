package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/gcleroux/projet-ift605/pkg/log"
	"github.com/gcleroux/projet-ift605/pkg/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	lis  net.Listener
	clog *log.Log
	srv  *grpc.Server
)

func main() {
	setupInterruptHandler()
	if err := run(); err != nil {
		grpclog.Fatal(err)
	}
}

func run() error {
	var err error

	lis, err = net.Listen("tcp", ":50051")
	if err != nil {
		return err
	}
	if err := os.MkdirAll("./log", os.ModePerm); err != nil {
		return err
	}

	dir := "./log"
	clog, err = log.NewLog(dir, log.Config{})
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

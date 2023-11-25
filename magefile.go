//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/magefile/mage/mg"
)

// Building the application
func Build() {
	mg.Deps(BuildServer)
	mg.Deps(BuildClient)
}

// Build the gRPC server
func BuildServer() error {
	mg.Deps(InstallDeps)
	mg.Deps(Compile)
	mg.Deps(CompileGateway)
	fmt.Println("Building Server...")
	cmd := exec.Command("go", "build", "-o", "server", "./cmd/server-cli")
	return cmd.Run()
}

// Build the gRPC client
func BuildClient() error {
	mg.Deps(InstallDeps)
	mg.Deps(Compile)
	mg.Deps(CompileGateway)
	fmt.Println("Building Client...")
	cmd := exec.Command("go", "build", "-o", "client", "./cmd/client-cli")
	return cmd.Run()
}

// Compiling protobuf objects
func Compile() error {
	fmt.Println("Compiling protobufs...")
	protoFiles, err := filepath.Glob("api/v1/*.proto")
	if err != nil {
		return err
	}
	if len(protoFiles) == 0 {
		return fmt.Errorf("no .proto files found in api/v1")
	}
	args := append([]string{
		"--go_out=.",
		"--go-grpc_out=.",
		"--go_opt=paths=source_relative",
		"--go-grpc_opt=paths=source_relative",
		"--proto_path=.",
	}, protoFiles...)
	cmd := exec.Command("protoc", args...)
	return cmd.Run()
}

// Compiling protobuf gateway
func CompileGateway() error {
	fmt.Println("Compiling gRPC Gateway")
	protoFiles, err := filepath.Glob("api/v1/*.proto")
	if err != nil {
		return err
	}
	if len(protoFiles) == 0 {
		return fmt.Errorf("no .proto files found in api/v1")
	}
	args := append([]string{
		"-I=.",
		"--grpc-gateway_out=.",
		"--grpc-gateway_opt=paths=source_relative",
		"--grpc-gateway_opt=generate_unbound_methods=true",
	}, protoFiles...)
	cmd := exec.Command("protoc", args...)
	return cmd.Run()
}

// Installing package dependencies
func InstallDeps() error {
	fmt.Println("Installing Deps...")
	cmd := exec.Command("go", "mod", "download")
	return cmd.Run()
}

// Running tests
func Test() error {
	fmt.Println("Testing code...")
	cmd := exec.Command("go", "test", "-coverpkg=./pkg/...", "./pkg/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Cleaning up
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("./log")
	os.RemoveAll("./server")
	os.RemoveAll("./client")
}

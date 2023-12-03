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

const (
	CONFIG_PATH string = ".config"
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
	cmd := exec.Command("go", "build", "-o", filepath.FromSlash("./bin/server"), filepath.FromSlash("./cmd/server-cli"))
	return cmd.Run()
}

// Build the gRPC client
func BuildClient() error {
	mg.Deps(InstallDeps)
	mg.Deps(Compile)
	mg.Deps(CompileGateway)
	fmt.Println("Building Client...")
	cmd := exec.Command("go", "build", "-o", filepath.FromSlash("./bin/client"), filepath.FromSlash("./cmd/client-cli"))
	return cmd.Run()
}

// Compiling protobuf objects
func Compile() error {
	fmt.Println("Compiling protobufs...")
	protoFiles, err := filepath.Glob(filepath.FromSlash("api/v1/*.proto"))
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
	protoFiles, err := filepath.Glob(filepath.FromSlash("api/v1/*.proto"))
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

// Generate the SSL certifications
func GenCert() error {
	fmt.Println("Generating Certs...")
	if err := os.MkdirAll(CONFIG_PATH, os.ModePerm); err != nil {
		return err
	}

	if err := genCACert(); err != nil {
		return err
	}
	if err := genServerCert(); err != nil {
		return err
	}

	pemFiles, err := filepath.Glob("*.pem")
	if err != nil {
		return err
	}
	csrFiles, err := filepath.Glob("*.csr")
	if err != nil {
		return err
	}
	files := append(pemFiles, csrFiles...)

	if len(files) == 0 {
		return fmt.Errorf("no *.pem|*.csr files found")
	}
	for _, file := range files {
		os.Rename(file, filepath.Join(CONFIG_PATH, filepath.Base(file)))
	}
	return nil
}

func genCACert() error {
	cfssl := exec.Command(
		"cfssl",
		"gencert",
		"-initca",
		filepath.FromSlash("test/ca-csr.json"),
	)
	cfssljson := exec.Command(
		"cfssljson",
		"-bare",
		"ca",
	)
	cfssljson.Stdin, _ = cfssl.StdoutPipe()

	if err := cfssl.Start(); err != nil {
		return err
	}
	if err := cfssljson.Run(); err != nil {
		return err
	}
	return cfssl.Wait()
}

func genServerCert() error {
	cfssl := exec.Command(
		"cfssl",
		"gencert",
		"-ca=ca.pem",
		"-ca-key=ca-key.pem",
		"-config="+filepath.FromSlash("test/ca-config.json"),
		"-profile=server",
		filepath.FromSlash("test/server-csr.json"),
	)
	cfssljson := exec.Command(
		"cfssljson",
		"-bare",
		"server",
	)

	cfssljson.Stdin, _ = cfssl.StdoutPipe()
	if err := cfssl.Start(); err != nil {
		return err
	}
	if err := cfssljson.Run(); err != nil {
		return err
	}
	return cfssl.Wait()
}

// Installing package dependencies
func InstallDeps() error {
	fmt.Println("Installing Deps...")
	cmd := exec.Command("go", "mod", "tidy")
	return cmd.Run()
}

// Running tests
func Test() error {
	fmt.Println("Testing code...")
	src := filepath.FromSlash("./src/...")
	cmd := exec.Command("go", "test", "-coverpkg=", src, src)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Cleaning up
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("data")
	os.RemoveAll("bin")
}

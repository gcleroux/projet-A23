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
func Build() error {
	mg.Deps(InstallDeps)
	mg.Deps(Compile)
	fmt.Println("Building...")
	cmd := exec.Command("go", "build", "-o", "bin/main", ".")
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
	args := append([]string{"--go_out=.", "--go_opt=paths=source_relative", "--proto_path=."}, protoFiles...)
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
	return cmd.Run()
}

// Cleaning up
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("bin")
}

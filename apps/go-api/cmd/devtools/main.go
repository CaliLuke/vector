package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CaliLuke/go-plane/internal/commandline"

	"github.com/CaliLuke/go-plane/internal/devtools"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "start" && args[0] != "check-login" && args[0] != "check-logs") {
		return commandline.Diagnostic(stderr, 2, "usage: devtools {start|check-login|check-logs}")
	}
	root, err := repositoryRoot()
	if err != nil {
		return commandline.Diagnostic(stderr, 1, fmt.Sprint(err))
	}
	var runErr error
	switch args[0] {
	case "start":
		return devtools.RunLocalDevelopment(context.Background(), root, stdout, stderr)
	case "check-login":
		runErr = devtools.CheckLocalLogin(context.Background(), os.Getenv("PLANE_TEST_EMAIL"), os.Getenv("PLANE_TEST_PASSWORD"), stdout)
	case "check-logs":
		runErr = devtools.CheckLocalLogging(context.Background(), root, stdout)
	}
	if runErr != nil {
		return commandline.Diagnostic(stderr, 1, fmt.Sprintf("devtools %s failed: %v", args[0], runErr))
	}
	return 0
}

func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "docker-compose-local.yml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("run devtools from the repository checkout")
		}
		dir = parent
	}
}

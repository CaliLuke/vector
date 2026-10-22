package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CaliLuke/go-plane/internal/devtools"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	root, err := repositoryRoot()
	if err != nil {
		fail(err)
	}
	var runErr error
	switch os.Args[1] {
	case "start":
		os.Exit(devtools.RunLocalDevelopment(context.Background(), root, os.Stdout, os.Stderr))
	case "check-login":
		runErr = devtools.CheckLocalLogin(context.Background(), os.Getenv("PLANE_TEST_EMAIL"), os.Getenv("PLANE_TEST_PASSWORD"), os.Stdout)
	case "check-logs":
		runErr = devtools.CheckLocalLogging(context.Background(), root, os.Stdout)
	default:
		usage()
	}
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "devtools %s failed: %v\n", os.Args[1], runErr)
		os.Exit(1)
	}
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

func usage() {
	fmt.Fprintln(os.Stderr, "usage: devtools {start|check-login|check-logs}")
	os.Exit(2)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

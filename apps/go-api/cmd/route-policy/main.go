package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/CaliLuke/go-plane/internal/commandline"

	"github.com/CaliLuke/go-plane/internal/routing"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stderr))
}

func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("route-policy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var outputPath string
	flags.StringVar(&outputPath, "output", "", "generated Caddy policy path")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if outputPath == "" {
		return commandline.Diagnostic(stderr, 2, "output is required")
	}
	policy, err := routing.Load()
	if err != nil {
		return commandline.Diagnostic(stderr, 1, fmt.Sprint(err))
	}
	output, err := routing.RenderCaddy(policy)
	if err != nil {
		return commandline.Diagnostic(stderr, 1, fmt.Sprint(err))
	}
	if err := os.WriteFile(outputPath, output, 0o644); err != nil {
		return commandline.Diagnostic(stderr, 1, fmt.Sprintf("write Caddy policy: %v", err))
	}
	return 0
}

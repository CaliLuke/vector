package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/CaliLuke/go-plane/internal/routing"
)

func main() {
	var outputPath string
	flag.StringVar(&outputPath, "output", "", "generated Caddy policy path")
	flag.Parse()
	if outputPath == "" {
		fmt.Fprintln(os.Stderr, "output is required")
		os.Exit(2)
	}
	policy, err := routing.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output, err := routing.RenderCaddy(policy)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputPath, output, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write Caddy policy: %v\n", err)
		os.Exit(1)
	}
}

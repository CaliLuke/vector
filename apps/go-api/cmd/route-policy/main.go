package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/CaliLuke/go-plane/internal/routing"
)

func main() {
	var inventoryPath string
	var outputPath string
	flag.StringVar(&inventoryPath, "inventory", "", "route inventory JSON path")
	flag.StringVar(&outputPath, "output", "", "generated Caddy policy path")
	flag.Parse()
	if inventoryPath == "" || outputPath == "" {
		fmt.Fprintln(os.Stderr, "inventory and output are required")
		os.Exit(2)
	}
	inventory, err := routing.LoadFile(inventoryPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	policy, err := routing.RenderCaddy(inventory)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(outputPath, policy, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write Caddy policy: %v\n", err)
		os.Exit(1)
	}
}

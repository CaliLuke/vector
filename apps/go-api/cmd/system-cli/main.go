package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/CaliLuke/go-plane/internal/commandline"

	loom "github.com/CaliLuke/loom/pkg"
)

func main() { os.Exit(run(os.Stdout, os.Stderr)) }

func run(stdout, stderr io.Writer) int {
	var (
		hostF = flag.String("host", "localhost", "Server host (valid values: localhost)")
		addrF = flag.String("url", "", "URL to service host")

		verboseF = flag.Bool("verbose", false, "Print request and response details")
		vF       = flag.Bool("v", false, "Print request and response details")
		timeoutF = flag.Int("timeout", 30, "Maximum number of seconds to wait for response")
	)
	flag.Usage = usage
	flag.Parse()

	var (
		addr    string
		timeout int
		debug   bool
	)
	{
		addr = *addrF
		if addr == "" {
			switch *hostF {
			case "localhost":
				addr = "http://localhost:8080"
			default:
				return commandline.Diagnostic(stderr, 1, fmt.Sprintf("invalid host argument: %q (valid hosts: localhost)", *hostF))
			}
		}
		timeout = *timeoutF
		debug = *verboseF || *vF
	}

	var (
		scheme string
		host   string
	)
	{
		u, err := url.Parse(addr)
		if err != nil {
			return commandline.Diagnostic(stderr, 1, fmt.Sprintf("invalid URL %#v: %s", addr, err))
		}
		scheme = u.Scheme
		host = u.Host
	}

	var (
		endpoint loom.Endpoint
		payload  any
		err      error
	)
	{
		switch scheme {
		case "http", "https":
			endpoint, payload, err = doHTTP(scheme, host, timeout, debug)
		default:
			return commandline.Diagnostic(stderr, 1, fmt.Sprintf("invalid scheme: %q (valid schemes: http|https)", scheme))
		}
	}
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return commandline.Diagnostic(stderr, 1, err.Error()+"\nrun '"+os.Args[0]+" --help' for detailed usage.")
	}

	data, err := endpoint(context.Background(), payload)
	if err != nil {
		return commandline.Diagnostic(stderr, 1, fmt.Sprint(err.Error()))
	}

	if data != nil {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(data); err != nil {
			return commandline.Diagnostic(stderr, 1, fmt.Sprintf("write response: %v", err))
		}
	}
	return 0
}

func usage() {
	var usageCommands []string
	usageCommands = append(usageCommands, httpUsageCommands()...)
	sort.Strings(usageCommands)
	usageCommands = slices.Compact(usageCommands)
	fmt.Fprintf(os.Stderr, `%s is a command line client for the system API.

Usage:
    %s [-host HOST][-url URL][-timeout SECONDS][-verbose|-v] SERVICE ENDPOINT [flags]

    -host HOST:  server host (localhost). valid values: localhost
    -url URL:    specify service URL overriding host URL (http://localhost:8080)
    -timeout:    maximum number of seconds to wait for response (30)
    -verbose|-v: print request and response details (false)

Commands:
%s
Additional help:
    %s SERVICE [ENDPOINT] --help

Example:
%s
`, os.Args[0], os.Args[0], indent(strings.Join(usageCommands, "\n")), os.Args[0], indent(httpUsageExamples()))
}

func indent(s string) string {
	if s == "" {
		return ""
	}
	return "    " + strings.ReplaceAll(s, "\n", "\n    ")
}

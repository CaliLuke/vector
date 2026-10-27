package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
)

func mcpConfigFromEnvironment() (mcptransport.Config, error) {
	cfg := mcptransport.DefaultConfig()
	if raw := os.Getenv("MCP_ENABLED"); raw != "" {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return cfg, fmt.Errorf("MCP_ENABLED must be a boolean")
		}
		cfg.Enabled = enabled
	}
	cfg.Mode = os.Getenv("MCP_AUTH_MODE")
	cfg.ResourceURL = os.Getenv("MCP_RESOURCE_URL")
	for _, value := range strings.Split(os.Getenv("MCP_TRUSTED_ORIGINS"), ",") {
		if origin := strings.TrimSpace(value); origin != "" {
			cfg.TrustedOrigins = append(cfg.TrustedOrigins, origin)
		}
	}
	if raw := os.Getenv("MCP_MAX_REQUEST_BYTES"); raw != "" {
		limit, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return cfg, fmt.Errorf("MCP_MAX_REQUEST_BYTES must be an integer")
		}
		cfg.MaxRequestBytes = limit
	}
	if raw := os.Getenv("MCP_REQUEST_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil {
			return cfg, fmt.Errorf("MCP_REQUEST_TIMEOUT must be a duration")
		}
		cfg.RequestTimeout = timeout
	}
	return cfg, cfg.Validate()
}

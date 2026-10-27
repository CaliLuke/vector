package mcptransport

import "time"

// RateLimitError reports an exhausted shared request limit without credentials.
type RateLimitError struct{ RetryAfter time.Duration }

func (*RateLimitError) Error() string { return "MCP request rate exceeded" }

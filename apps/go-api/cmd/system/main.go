package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/CaliLuke/go-plane/internal/requestinput"

	"github.com/CaliLuke/go-plane/internal/authentication/browsersession"
	"github.com/CaliLuke/go-plane/internal/backgroundtasks"
	"github.com/CaliLuke/go-plane/internal/mcpoauth"
	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/platform"
	"github.com/CaliLuke/go-plane/internal/telemetry"
	workitemstore "github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/loom/clue/log"
)

const (
	defaultHTTPAddress     = "0.0.0.0:8080"
	defaultShutdownTimeout = 30 * time.Second
	defaultFileSizeLimit   = int64(5 * 1024 * 1024)
)

type config struct {
	mcp                        mcptransport.Config
	mcpOAuth                   mcpoauth.Config
	goBackgroundTasks          bool
	goBackgroundTasksNoWorkers bool
	httpAddress                string
	shutdownTimeout            time.Duration
	debug                      bool
	databaseURL                string
	readReplicaDatabaseURL     string
	redisURL                   string
	apiKeyRate                 string
	corsOrigins                []string
	csrfTrustedOrigins         []string
	sessionCookie              string
	adminSessionCookie         string
	cookieDomain               string
	sessionCookieAge           int64
	adminSessionCookieAge      int64
	sessionCookieSecure        bool
	saveEveryRequest           bool
	secretKey                  string
	fallbackKeys               []string
	fileSizeLimit              int64
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output io.Writer) (runErr error) {
	cfg, err := parseConfig(args)
	if err != nil {
		return err
	}
	if err = validateMCPAvailability(cfg.mcp, cfg.mcpOAuth); err != nil {
		return err
	}

	var mcpOAuthProvider *mcpoauth.Provider
	if cfg.mcp.Enabled && (cfg.mcp.Mode == "oauth" || cfg.mcp.Mode == "both") {
		mcpOAuthProvider, err = mcpoauth.NewProvider(ctx, cfg.mcpOAuth, nil)
		if err != nil {
			return err
		}
	}

	wrapHTTP, otelLogOutput, shutdownTelemetry, err := telemetry.ConfigureHTTPObservability(ctx, "plane-go-api")
	if err != nil {
		return fmt.Errorf("configure HTTP observability: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		runErr = errors.Join(runErr, shutdownTelemetry(shutdownCtx))
	}()
	logCtx := newLogContext(ctx, output, cfg.debug, otelLogOutput)
	defer func() {
		if runErr != nil {
			log.Error(logCtx, runErr, log.KV{K: log.MessageKey, V: "Go API exited with an error"})
		}
	}()
	assembly := &systemAssembly{ctx: ctx, cfg: cfg, logCtx: logCtx, wrapHTTP: wrapHTTP, mcpOAuthProvider: mcpOAuthProvider}
	defer assembly.close(&runErr)
	if err := assembly.configure(); err != nil {
		return err
	}
	return assembly.serve()
}

func issueActivityTaskPublisher(fallback backgroundtasks.TaskFallback, admission backgroundtasks.IssueActivityTaskAdmitter) (workitemstore.Publisher, error) {
	if fallback == nil {
		return nil, errors.New("task publisher is required")
	}
	if admission == nil {
		return fallback, nil
	}
	return backgroundtasks.NewIssueActivityTaskPublisher(fallback, admission)
}

type rejectingTaskPublisher struct{}

func (rejectingTaskPublisher) Task(_ context.Context, name string, _ []any, _ map[string]any) error {
	return fmt.Errorf("native background task %q has no registered Go handler", name)
}

func (rejectingTaskPublisher) SoftDelete(_ context.Context, table, id, field string) error {
	return fmt.Errorf("native soft delete %s.%s by %s has no registered Go handler", table, field, id)
}

func readReplicaDatabaseURL() (string, error) {
	if os.Getenv("ENABLE_READ_REPLICA") != "1" {
		return "", nil
	}
	if configured := os.Getenv("DATABASE_READ_REPLICA_URL"); configured != "" {
		return configured, nil
	}
	databaseName := os.Getenv("POSTGRES_READ_REPLICA_DB")
	username := os.Getenv("POSTGRES_READ_REPLICA_USER")
	host := os.Getenv("POSTGRES_READ_REPLICA_HOST")
	if databaseName == "" || username == "" || host == "" {
		return "", fmt.Errorf("read replica requires DATABASE_READ_REPLICA_URL or POSTGRES_READ_REPLICA_DB, POSTGRES_READ_REPLICA_USER, and POSTGRES_READ_REPLICA_HOST")
	}
	password := os.Getenv("POSTGRES_READ_REPLICA_PASSWORD")
	user := url.User(username)
	if password != "" {
		user = url.UserPassword(username, password)
	}
	return (&url.URL{
		Scheme: "postgresql",
		User:   user,
		Host:   net.JoinHostPort(host, envValueOrDefault("POSTGRES_READ_REPLICA_PORT", "5432")),
		Path:   "/" + databaseName,
	}).String(), nil
}
func parseConfig(args []string) (config, error) {
	mcpConfig, err := mcpConfigFromEnvironment()
	if err != nil {
		return config{}, err
	}
	sessionCookieAge, err := envSeconds("SESSION_COOKIE_AGE", 604800)
	if err != nil {
		return config{}, err
	}
	adminSessionCookieAge, err := envSeconds("ADMIN_SESSION_COOKIE_AGE", 3600)
	if err != nil {
		return config{}, err
	}
	fileSizeLimit, err := envInt64("FILE_SIZE_LIMIT", defaultFileSizeLimit)
	if err != nil {
		return config{}, err
	}
	debugMode, err := envNativeDebug()
	if err != nil {
		return config{}, err
	}

	flags := flag.NewFlagSet("go-plane", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	cfg := config{sessionCookieAge: sessionCookieAge, adminSessionCookieAge: adminSessionCookieAge, mcp: mcpConfig, mcpOAuth: mcpOAuthConfigFromEnvironment(mcpConfig.ResourceURL)}
	flags.StringVar(&cfg.httpAddress, "http-address", defaultHTTPAddress, "HTTP listen address")
	flags.DurationVar(&cfg.shutdownTimeout, "shutdown-timeout", defaultShutdownTimeout, "graceful shutdown timeout")
	flags.BoolVar(&cfg.debug, "debug", debugMode, "enable debug responses and logs")
	flags.BoolVar(&cfg.goBackgroundTasks, "go-background-tasks", false, "run native durable tasks and admit migrated task groups")
	flags.BoolVar(&cfg.goBackgroundTasksNoWorkers, "go-background-tasks-no-workers", false, "initialize native durable admission without starting activity workers")
	flags.StringVar(&cfg.databaseURL, "database-url", os.Getenv("DATABASE_URL"), "PostgreSQL URL")
	flags.StringVar(&cfg.readReplicaDatabaseURL, "read-replica-database-url", "", "PostgreSQL read replica URL")
	flags.StringVar(&cfg.redisURL, "redis-url", os.Getenv("REDIS_URL"), "Redis URL")
	flags.StringVar(&cfg.apiKeyRate, "api-key-rate", envValueOrDefault("API_KEY_RATE_LIMIT", "60/minute"), "API-key throttle rate")
	var corsOrigins string
	flags.StringVar(&corsOrigins, "cors-allowed-origins", os.Getenv("CORS_ALLOWED_ORIGINS"), "comma-separated CORS allowed origins")
	csrfTrustedOriginsRaw, csrfTrustedOriginsSet := os.LookupEnv("CSRF_TRUSTED_ORIGINS")
	if !csrfTrustedOriginsSet {
		csrfTrustedOriginsRaw = os.Getenv("CORS_ALLOWED_ORIGINS")
	}
	flags.StringVar(&csrfTrustedOriginsRaw, "csrf-trusted-origins", csrfTrustedOriginsRaw, "comma-separated trusted origins for account-authentication CSRF checks")
	flags.StringVar(&cfg.sessionCookie, "session-cookie-name", envValueOrDefault("SESSION_COOKIE_NAME", browsersession.DefaultCookieName), "browser session cookie name")
	flags.StringVar(&cfg.adminSessionCookie, "admin-session-cookie-name", "admin-session-id", "admin browser session cookie name")
	flags.StringVar(&cfg.cookieDomain, "session-cookie-domain", os.Getenv("COOKIE_DOMAIN"), "browser session cookie domain")
	flags.Func("session-cookie-age", "browser session cookie age", func(value string) error {
		age, sessionAgeFlagErr := sessionAgeFlag(value)
		if sessionAgeFlagErr == nil {
			cfg.sessionCookieAge = age
		}
		return sessionAgeFlagErr
	})
	flags.Func("admin-session-cookie-age", "admin browser session cookie age", func(value string) error {
		age, sessionAgeFlagErr := sessionAgeFlag(value)
		if sessionAgeFlagErr == nil {
			cfg.adminSessionCookieAge = age
		}
		return sessionAgeFlagErr
	})
	flags.BoolVar(&cfg.saveEveryRequest, "session-refresh-on-request", os.Getenv("SESSION_REFRESH_ON_REQUEST") == "1", "refresh browser sessions after every successful request")
	flags.StringVar(&cfg.secretKey, "secret-key", os.Getenv("SECRET_KEY"), "browser session signing key")
	var fallbackKeys string
	flags.StringVar(&fallbackKeys, "fallback-secret-keys", os.Getenv("SECRET_KEY_FALLBACKS"), "comma-separated fallback signing keys")
	flags.Int64Var(&cfg.fileSizeLimit, "file-size-limit", fileSizeLimit, "request body size limit")

	if parseErr := flags.Parse(args); parseErr != nil {
		return config{}, fmt.Errorf("parse flags: %w", parseErr)
	}
	if flags.NArg() != 0 {
		return config{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if cfg.goBackgroundTasksNoWorkers && !cfg.goBackgroundTasks {
		return config{}, fmt.Errorf("go-background-tasks-no-workers requires go-background-tasks")
	}
	readReplicaFlagSet := false
	flags.Visit(func(parsed *flag.Flag) {
		readReplicaFlagSet = readReplicaFlagSet || parsed.Name == "read-replica-database-url"
	})
	if !readReplicaFlagSet {
		cfg.readReplicaDatabaseURL, err = readReplicaDatabaseURL()
		if err != nil {
			return config{}, err
		}
	}
	if cfg.httpAddress == "" {
		return config{}, fmt.Errorf("http-address must not be empty")
	}
	if cfg.shutdownTimeout <= 0 {
		return config{}, fmt.Errorf("shutdown-timeout must be positive")
	}
	if cfg.secretKey == "" {
		return config{}, fmt.Errorf("secret-key is required")
	}
	if cfg.adminSessionCookie == "" {
		return config{}, fmt.Errorf("admin-session-cookie-name must not be empty")
	}
	for _, key := range strings.Split(fallbackKeys, ",") {
		if key = strings.TrimSpace(key); key != "" {
			cfg.fallbackKeys = append(cfg.fallbackKeys, key)
		}
	}
	for _, origin := range strings.Split(corsOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			cfg.corsOrigins = append(cfg.corsOrigins, origin)
		}
	}
	flags.Visit(func(parsed *flag.Flag) {
		if parsed.Name == "csrf-trusted-origins" {
			csrfTrustedOriginsSet = true
		}
	})
	if csrfTrustedOriginsSet {
		for _, origin := range strings.Split(csrfTrustedOriginsRaw, ",") {
			if origin = strings.TrimSpace(origin); origin != "" {
				cfg.csrfTrustedOrigins = append(cfg.csrfTrustedOrigins, origin)
			}
		}
	} else {
		cfg.csrfTrustedOrigins = append(cfg.csrfTrustedOrigins, cfg.corsOrigins...)
	}
	cfg.sessionCookieSecure = len(cfg.corsOrigins) > 0
	for _, origin := range cfg.corsOrigins {
		if strings.Contains(origin, "http:") {
			cfg.sessionCookieSecure = false
			break
		}
	}
	return cfg, nil
}

func envSeconds(name string, fallback int64) (int64, error) {
	raw, found := os.LookupEnv(name)
	if !found {
		return fallback, nil
	}
	value, err := boundedInt64(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer number of seconds: %w", name, err)
	}
	return value, nil
}

func sessionAgeFlag(value string) (int64, error) {
	if duration, err := time.ParseDuration(value); err == nil {
		return int64(duration / time.Second), nil
	}
	age, err := boundedInt64(value)
	if err != nil {
		return 0, fmt.Errorf("session cookie age must be a duration or integer seconds: %w", err)
	}
	return age, nil
}

func boundedInt64(value string) (int64, error) {
	integer, err := requestinput.IntegerField(value)
	if err != nil {
		return 0, err
	}
	if !integer.IsInt64() {
		return 0, fmt.Errorf("integer exceeds signed 64-bit range")
	}
	return integer.Int64(), nil
}

func envValueOrDefault(name, fallback string) string {
	if value, found := os.LookupEnv(name); found {
		return value
	}
	return fallback
}

func envInt64(name string, fallback int64) (int64, error) {
	raw, found := os.LookupEnv(name)
	if !found {
		return fallback, nil
	}
	value, err := boundedInt64(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return value, nil
}

func envNativeDebug() (bool, error) {
	raw, found := os.LookupEnv("DEBUG")
	if !found {
		return false, nil
	}
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false, fmt.Errorf("DEBUG must be a boolean: %w", err)
	}
	return value, nil
}

func newLogContext(ctx context.Context, output io.Writer, debug bool, additionalOutputs ...log.Output) context.Context {
	outputs := []log.Output{{Writer: output, Format: log.FormatJSON}}
	outputs = append(outputs, additionalOutputs...)
	options := []log.LogOption{
		log.WithOutputs(outputs...),
		log.WithDisableBuffering(func(context.Context) bool { return true }),
		log.WithFunc(log.Span),
		log.WithFunc(func(ctx context.Context) []log.KV {
			var fields []log.KV
			requestID, ok := platform.RequestID(ctx)
			if ok {
				fields = append(fields, log.KV{K: log.RequestIDKey, V: requestID})
			}
			switch ctx.Err() {
			case context.Canceled:
				fields = append(fields, log.KV{K: "request_context_error", V: "context_canceled"})
			case context.DeadlineExceeded:
				fields = append(fields, log.KV{K: "request_context_error", V: "deadline_exceeded"})
			}
			return fields
		}),
	}
	if debug {
		options = append(options, log.WithDebug())
	}
	return log.Context(ctx, options...)
}

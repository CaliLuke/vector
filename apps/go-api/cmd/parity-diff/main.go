package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/clioutput"
	"github.com/CaliLuke/go-plane/internal/parity"
)

type stringList []string

func (values *stringList) String() string {
	return strings.Join(*values, ",")
}

func (values *stringList) Set(value string) error {
	if value == "" {
		return fmt.Errorf("changed table cannot be empty")
	}
	*values = append(*values, value)
	return nil
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(arguments []string, stdin *os.File, stdout io.Writer, stderr io.Writer) int {
	if len(arguments) == 1 && arguments[0] == confirmationReadFlag {
		return readConfirmationInput(stdin, stderr)
	}
	flags := flag.NewFlagSet("parity-diff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var config parity.Config
	var normalizationPath string
	var sourceResponsePath string
	var targetResponsePath string
	var timeout time.Duration
	var expectedTables stringList
	flags.StringVar(&config.SourcePostgresURL, "source-postgres-url", "", "source Django PostgreSQL URL (required)")
	flags.StringVar(&config.TargetPostgresURL, "target-postgres-url", "", "target Go PostgreSQL URL (required)")
	flags.StringVar(&config.Schema, "schema", "public", "PostgreSQL schema")
	flags.StringVar(&normalizationPath, "normalizations", "", "strict JSON normalization file")
	flags.StringVar(&sourceResponsePath, "source-response", "", "recorder-canonicalized source response JSON (required)")
	flags.StringVar(&targetResponsePath, "target-response", "", "recorder-canonicalized target response JSON (required)")
	flags.Var(&expectedTables, "expected-changed-table", "expected application table write (repeatable)")
	flags.DurationVar(&timeout, "timeout", 5*time.Minute, "complete capture and comparison timeout")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		return clioutput.Report(stderr, 2, "error: positional arguments are not supported\n")
	}
	config.ExpectedChangedTables = expectedTables
	if normalizationPath != "" {
		normalizations, err := parity.LoadNormalizations(normalizationPath)
		if err != nil {
			return clioutput.Report(stderr, 2, "error: %v\n", err)
		}
		config.Normalizations = normalizations
	}
	if sourceResponsePath == "" || targetResponsePath == "" {
		return clioutput.Report(stderr, 2, "error: --source-response and --target-response are required\n")
	}
	for role, path := range map[string]string{"source": sourceResponsePath, "target": targetResponsePath} {
		if err := validateResponsePath(path); err != nil {
			return clioutput.Report(stderr, 2, "error: %s response path: %v\n", role, err)
		}
	}
	action, err := responseAction(func(ctx context.Context) error { return readConfirmation(ctx, stdin) }, stdout, sourceResponsePath, targetResponsePath)
	if err != nil {
		return clioutput.Report(stderr, 2, "error: response artifacts: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	diff, err := parity.CaptureAndCompare(ctx, config, action)
	return reportComparison(stdout, stderr, diff, err)
}

func reportComparison(stdout, stderr io.Writer, diff string, comparisonErr error) int {
	var mismatch *parity.MismatchError
	if comparisonErr != nil && !errors.As(comparisonErr, &mismatch) {
		return clioutput.Report(stderr, 1, "error: %v\n", comparisonErr)
	}
	message, exitCode := diff, 1
	if comparisonErr == nil && diff == "" {
		message, exitCode = "parity match: source and target transitions are equal", 0
	}
	if _, err := fmt.Fprintln(stdout, message); err != nil {
		comparisonErr = errors.Join(comparisonErr, fmt.Errorf("write parity result: %w", err))
	}
	if comparisonErr != nil {
		return clioutput.Report(stderr, 1, "error: %v\n", comparisonErr)
	}
	return exitCode
}

func validateResponsePath(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return fmt.Errorf("must be a regular file")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("inspect file: %w", err)
	}
	parent, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("inspect parent directory: %w", err)
	}
	if !parent.IsDir() {
		return fmt.Errorf("parent must be a directory")
	}
	return nil
}

type responseArtifactState struct {
	exists bool
	info   os.FileInfo
	digest [sha256.Size]byte
}

func responseAction(
	confirm func(context.Context) error,
	stdout io.Writer,
	sourcePath string,
	targetPath string,
) (func(context.Context) (parity.ActionResult, error), error) {
	sourceBefore, err := captureResponseArtifact(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("capture source response before snapshots: %w", err)
	}
	targetBefore, err := captureResponseArtifact(targetPath)
	if err != nil {
		return nil, fmt.Errorf("capture target response before snapshots: %w", err)
	}
	if sameResponseArtifact(sourcePath, sourceBefore, targetPath, targetBefore) {
		return nil, fmt.Errorf("source and target response artifacts must be separate files")
	}

	return func(ctx context.Context) (parity.ActionResult, error) {
		sourceAtPrompt, err := captureResponseArtifact(sourcePath)
		if err != nil {
			return parity.ActionResult{}, fmt.Errorf("capture source response before action prompt: %w", err)
		}
		targetAtPrompt, err := captureResponseArtifact(targetPath)
		if err != nil {
			return parity.ActionResult{}, fmt.Errorf("capture target response before action prompt: %w", err)
		}
		if sameResponseArtifact(sourcePath, sourceAtPrompt, targetPath, targetAtPrompt) {
			return parity.ActionResult{}, fmt.Errorf("source and target response artifacts must be separate files")
		}
		if _, promptErr := fmt.Fprintln(stdout, "before snapshots captured; run the equivalent source and target actions, then press Enter"); promptErr != nil {
			return parity.ActionResult{}, fmt.Errorf("write action prompt: %w", promptErr)
		}
		readErr := confirm(ctx)
		if readErr != nil {
			return parity.ActionResult{}, readErr
		}
		sourceAfter, err := captureResponseArtifact(sourcePath)
		if err != nil {
			return parity.ActionResult{}, fmt.Errorf("capture source response after actions: %w", err)
		}
		targetAfter, err := captureResponseArtifact(targetPath)
		if err != nil {
			return parity.ActionResult{}, fmt.Errorf("capture target response after actions: %w", err)
		}
		if !responseArtifactIsFresh(sourceAtPrompt, sourceAfter) {
			return parity.ActionResult{}, fmt.Errorf("source response artifact was not created, replaced, or changed after the action prompt")
		}
		if !responseArtifactIsFresh(targetAtPrompt, targetAfter) {
			return parity.ActionResult{}, fmt.Errorf("target response artifact was not created, replaced, or changed after the action prompt")
		}
		if sameResponseArtifact(sourcePath, sourceAfter, targetPath, targetAfter) {
			return parity.ActionResult{}, fmt.Errorf("source and target response artifacts must be separate files")
		}
		sourceResponse, err := parity.LoadCapturedResponse(sourcePath)
		if err != nil {
			return parity.ActionResult{}, fmt.Errorf("source response: %w", err)
		}
		targetResponse, err := parity.LoadCapturedResponse(targetPath)
		if err != nil {
			return parity.ActionResult{}, fmt.Errorf("target response: %w", err)
		}
		return parity.ActionResult{SourceResponse: sourceResponse, TargetResponse: targetResponse}, nil
	}, nil
}

func captureResponseArtifact(path string) (_ responseArtifactState, resultErr error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return responseArtifactState{}, nil
	}
	if err != nil {
		return responseArtifactState{}, fmt.Errorf("open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close file: %w", closeErr))
		}
	}()
	info, err := file.Stat()
	if err != nil {
		return responseArtifactState{}, fmt.Errorf("inspect file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return responseArtifactState{}, fmt.Errorf("must be a regular file")
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		return responseArtifactState{}, fmt.Errorf("read file: %w", err)
	}
	return responseArtifactState{exists: true, info: info, digest: sha256.Sum256(contents)}, nil
}

func responseArtifactIsFresh(before responseArtifactState, after responseArtifactState) bool {
	if !after.exists {
		return false
	}
	if !before.exists {
		return true
	}
	return !os.SameFile(before.info, after.info) ||
		before.info.Size() != after.info.Size() ||
		!before.info.ModTime().Equal(after.info.ModTime()) ||
		before.digest != after.digest
}

func sameResponseArtifact(
	sourcePath string,
	source responseArtifactState,
	targetPath string,
	target responseArtifactState,
) bool {
	sourceAbsolute, sourceErr := filepath.Abs(sourcePath)
	targetAbsolute, targetErr := filepath.Abs(targetPath)
	if sourceErr == nil && targetErr == nil && sourceAbsolute == targetAbsolute {
		return true
	}
	return source.exists && target.exists && os.SameFile(source.info, target.info)
}

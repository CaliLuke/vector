package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/CaliLuke/go-plane/internal/clioutput"
)

const confirmationReadFlag = "--internal-read-confirmation"

// readConfirmation borrows input. The helper owns the blocking read, so context
// cancellation kills and reaps it without closing stdin or abandoning a goroutine.
func readConfirmation(ctx context.Context, input *os.File) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if input == nil {
		return errors.New("confirmation input is missing")
	}
	if _, err := input.Stat(); err != nil {
		return fmt.Errorf("inspect confirmation input: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate confirmation helper: %w", err)
	}
	command := exec.CommandContext(ctx, executable, confirmationReadFlag)
	// An *os.File is inherited directly; exec does not start an input-copy goroutine.
	command.Stdin = input
	var diagnostic bytes.Buffer
	command.Stderr = &diagnostic
	err = command.Run()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		if message := strings.TrimSpace(diagnostic.String()); message != "" {
			return fmt.Errorf("read confirmation: %w: %s", err, message)
		}
		return fmt.Errorf("read confirmation: %w", err)
	}
	return nil
}

func readConfirmationInput(input *os.File, stderr io.Writer) int {
	if input == nil {
		return clioutput.Report(stderr, 1, "error: confirmation input is missing\n")
	}
	_, err := bufio.NewReader(input).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return clioutput.Report(stderr, 1, "error: read confirmation input: %v\n", err)
	}
	return 0
}

package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

type migrationFixture struct {
	version                                             uint
	dirty                                               bool
	upErr, versionErr, sourceCloseErr, databaseCloseErr error
	upCalls, versionCalls, closeCalls                   int
}

func (m *migrationFixture) Up() error {
	m.upCalls++
	if m.upErr != nil {
		return m.upErr
	}
	m.version = 3
	return nil
}
func (m *migrationFixture) Version() (uint, bool, error) {
	m.versionCalls++
	return m.version, m.dirty, m.versionErr
}
func (m *migrationFixture) Close() (error, error) {
	m.closeCalls++
	return m.sourceCloseErr, m.databaseCloseErr
}

type failingOutput struct{}

func (failingOutput) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestMigrationOutputFailureDoesNotRepeatAppliedMigration(t *testing.T) {
	m := &migrationFixture{}
	err := executeSupportMigration(m, "up", failingOutput{})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("error = %v; want output cause", err)
	}
	if m.version != 3 || m.upCalls != 1 || m.closeCalls != 1 {
		t.Fatalf("migration state = %+v; want applied once and closed once", m)
	}
}
func TestMigrationVersionOutputFailureClosesWithoutWriting(t *testing.T) {
	m := &migrationFixture{version: 7, dirty: true}
	err := executeSupportMigration(m, "version", failingOutput{})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("error = %v; want output cause", err)
	}
	if m.version != 7 || !m.dirty || m.upCalls != 0 || m.versionCalls != 1 || m.closeCalls != 1 {
		t.Fatalf("unexpected migration state: %+v", m)
	}
}
func TestMigrationRetainsOperationAndBothCleanupErrors(t *testing.T) {
	operation := errors.New("migration rejected")
	source := errors.New("source close failed")
	database := errors.New("database close failed")
	m := &migrationFixture{upErr: operation, sourceCloseErr: source, databaseCloseErr: database}
	var output bytes.Buffer
	err := executeSupportMigration(m, "up", &output)
	if !errors.Is(err, operation) || !errors.Is(err, source) || !errors.Is(err, database) {
		t.Fatalf("error = %v; want all three causes", err)
	}
	if output.Len() != 0 || m.closeCalls != 1 || m.upCalls != 1 {
		t.Fatalf("output=%q migration=%+v", output.String(), m)
	}
}
func TestMigrationCleanupFailureIsNotSuccess(t *testing.T) {
	failure := errors.New("database close failed")
	m := &migrationFixture{databaseCloseErr: failure}
	err := executeSupportMigration(m, "up", io.Discard)
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v; want cleanup cause", err)
	}
	if m.version != 3 || m.closeCalls != 1 || m.upCalls != 1 {
		t.Fatalf("unexpected migration state: %+v", m)
	}
}
func TestMigrationVersionReadFailureDoesNotAnnounceSuccess(t *testing.T) {
	failure := errors.New("version unavailable")
	m := &migrationFixture{versionErr: failure}
	var output bytes.Buffer
	err := executeSupportMigration(m, "version", &output)
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v; want version cause", err)
	}
	if output.Len() != 0 || m.closeCalls != 1 || m.upCalls != 0 {
		t.Fatalf("output=%q migration=%+v", output.String(), m)
	}
}

func TestMigrationNoChangeRetainsExistingVersion(t *testing.T) {
	m := &migrationFixture{version: 7, upErr: migrate.ErrNoChange}
	var output bytes.Buffer
	if err := executeSupportMigration(m, "up", &output); err != nil {
		t.Fatal(err)
	}
	if m.version != 7 || m.upCalls != 1 || m.closeCalls != 1 || output.Len() == 0 {
		t.Fatalf("output=%q migration=%+v", output.String(), m)
	}
}

func TestMigrationVersionReportsStoredDirtyState(t *testing.T) {
	m := &migrationFixture{version: 7, dirty: true}
	var output bytes.Buffer
	if err := executeSupportMigration(m, "version", &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "version=7 dirty=true") || m.upCalls != 0 || m.closeCalls != 1 {
		t.Fatalf("output=%q migration=%+v", output.String(), m)
	}
}

func TestMigrationOutputAndCleanupFailuresAreBothRetained(t *testing.T) {
	failure := errors.New("source cleanup failed")
	m := &migrationFixture{sourceCloseErr: failure}
	err := executeSupportMigration(m, "up", failingOutput{})
	if !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, failure) {
		t.Fatalf("error lost a cause: %v", err)
	}
	if m.upCalls != 1 || m.closeCalls != 1 || m.version != 3 {
		t.Fatalf("unexpected migration state: %+v", m)
	}
}

func TestMigrationUsageDiagnosticFailureExitsAsIOFailure(t *testing.T) {
	if code := run(nil, io.Discard, failingOutput{}); code != 1 {
		t.Fatalf("exit code = %d; want I/O failure", code)
	}
}

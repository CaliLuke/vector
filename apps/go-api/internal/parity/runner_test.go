package parity

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSamePhysicalDatabaseUsesPostgresIdentity(t *testing.T) {
	startedAt := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	source := databaseIdentity{name: "plane", oid: 16384, systemIdentifier: "7615923049701060123", postmasterStartedAt: startedAt}

	if !samePhysicalDatabase(source, source) {
		t.Fatal("samePhysicalDatabase() = false for the same PostgreSQL identity")
	}

	tests := []struct {
		name   string
		target databaseIdentity
	}{
		{name: "different database name", target: databaseIdentity{name: "plane_target", oid: 16384, systemIdentifier: source.systemIdentifier, postmasterStartedAt: startedAt}},
		{name: "different database oid", target: databaseIdentity{name: "plane", oid: 16385, systemIdentifier: source.systemIdentifier, postmasterStartedAt: startedAt}},
		{name: "different cluster", target: databaseIdentity{name: "plane", oid: 16384, systemIdentifier: "different", postmasterStartedAt: startedAt}},
		{name: "different server process", target: databaseIdentity{name: "plane", oid: 16384, systemIdentifier: source.systemIdentifier, postmasterStartedAt: startedAt.Add(time.Second)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if samePhysicalDatabase(source, test.target) {
				t.Fatal("samePhysicalDatabase() = true for separate PostgreSQL identities")
			}
		})
	}
}

func TestSafeDatabaseErrorDoesNotExposeConnectionDetails(t *testing.T) {
	err := safeDatabaseError("source", "connect for identity check", errors.New("postgresql://user:secret@database/plane"))
	for _, sensitive := range []string{"user", "secret", "database/plane"} {
		if strings.Contains(err.Error(), sensitive) {
			t.Fatalf("safeDatabaseError() exposed %q: %v", sensitive, err)
		}
	}
}

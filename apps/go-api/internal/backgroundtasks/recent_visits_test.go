package backgroundtasks

import (
	"testing"
)

func TestValidateRecentVisitInputAllowsNullableEntityIdentifier(t *testing.T) {
	input := RecentVisitInput{Slug: "workspace", EntityName: "PROJECT", EntityIdentifier: nil, UserID: "4a1d6291-1698-4c9a-a4c7-010a77dd7288"}
	if err := validateRecentVisitInput(input); err != nil {
		t.Fatalf("nullable entity identifier should be valid: %v", err)
	}
	invalid := "not-a-uuid"
	input.EntityIdentifier = &invalid
	if err := validateRecentVisitInput(input); err == nil {
		t.Fatal("invalid entity identifier must fail validation")
	}
}

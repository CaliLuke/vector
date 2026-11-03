package projectmembershipreads

import "github.com/CaliLuke/go-plane/internal/emailvalidation"

func normalizeInvitationEmail(raw string) string {
	return emailvalidation.Normalize(raw)
}

func validInvitationEmail(raw string) bool {
	return emailvalidation.Valid(raw)
}

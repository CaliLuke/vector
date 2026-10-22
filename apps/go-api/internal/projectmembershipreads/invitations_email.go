package projectmembershipreads

import "github.com/CaliLuke/go-plane/internal/emailvalidation"

func normalizeInvitationEmail(raw string) string {
	return emailvalidation.NormalizePythonEmail(raw)
}

func validInvitationEmail(raw string) bool {
	return emailvalidation.ValidDjangoEmail(raw)
}

// Package emailvalidation owns account and invitation address contracts.
package emailvalidation

import (
	"net/mail"
	"net/netip"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/idna"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Normalize retains the full Unicode lowercase identity used by stored accounts.
// It does not rewrite internationalized domains or quoted local parts: either
// would change an existing account's lookup key.
func Normalize(raw string) string {
	return cases.Lower(language.Und).String(strings.TrimSpace(raw))
}

var mailboxDomain = idna.New(idna.MapForLookup(), idna.VerifyDNSLength(true))

// Valid accepts a single bare mailbox, including internationalized domains and
// local development hosts. Display names, controls and address lists are not
// account identities. The bound matches the persisted account email column.
func Valid(raw string) bool {
	if !utf8.ValidString(raw) || utf8.RuneCountInString(raw) > 255 || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return false
	}
	at := strings.LastIndexByte(raw, '@')
	if at <= 0 {
		return false
	}
	local, domain := raw[:at], raw[at+1:]
	// Validate a local part with the standard mailbox parser independently of
	// domain literals and IDNA. Explicit framing excludes names and comments.
	if local[0] == '"' {
		if len(local) < 2 || local[len(local)-1] != '"' {
			return false
		}
	} else if strings.ContainsAny(local, "() \t") {
		return false
	}
	address, err := mail.ParseAddress(local + "@example.test")
	if err != nil || address.Name != "" || strings.ContainsAny(local, "<>\r\n") {
		return false
	}
	if strings.HasPrefix(domain, "[") && strings.HasSuffix(domain, "]") {
		ip, parseErr := netip.ParseAddr(domain[1 : len(domain)-1])
		return parseErr == nil && ip.Zone() == ""
	}
	ascii, err := mailboxDomain.ToASCII(domain)
	if err != nil || len(ascii) > 253 {
		return false
	}
	if strings.EqualFold(ascii, "localhost") {
		return true
	}
	return strings.Contains(ascii, ".") && !strings.HasSuffix(ascii, ".")
}

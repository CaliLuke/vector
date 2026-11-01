package requestinput

import (
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/idna"
)

var urlHostNames = idna.New(idna.MapForLookup(), idna.VerifyDNSLength(true), idna.BidiRule())

// ValidURL accepts absolute web and file-transfer URLs. Caller-specific scheme
// restrictions and destination authorization belong to the caller.
func ValidURL(raw string) bool {
	if !utf8.ValidString(raw) || utf8.RuneCountInString(raw) > 2048 || strings.ContainsFunc(raw, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
		return false
	}
	parsed, parseErr := url.Parse(raw)
	if parseErr != nil || parsed.Hostname() == "" || parsed.Opaque != "" {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "ftp", "ftps":
	default:
		return false
	}
	if strings.HasSuffix(parsed.Host, ":") {
		return false
	}
	if port := parsed.Port(); port != "" {
		if _, portErr := strconv.ParseUint(port, 10, 16); portErr != nil {
			return false
		}
	}
	if strings.ContainsAny(parsed.Hostname(), "[]:") || strings.HasPrefix(parsed.Host, "[") {
		address, addressErr := netip.ParseAddr(parsed.Hostname())
		return addressErr == nil && address.Is6()
	}
	if address, addressErr := netip.ParseAddr(parsed.Hostname()); addressErr == nil {
		return address.Is4()
	}
	host, err := urlHostNames.ToASCII(strings.TrimSuffix(parsed.Hostname(), "."))
	if err != nil || host == "" {
		return false
	}
	// Numeric names that failed IP parsing are ambiguous address spellings.
	return strings.ContainsFunc(host, func(r rune) bool { return r != '.' && (r < '0' || r > '9') })
}

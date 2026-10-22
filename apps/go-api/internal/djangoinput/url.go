package djangoinput

import (
	"net/netip"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Django 5.2 URLValidator bounds URLs at 2048 characters, DNS labels at 63,
// hostnames at 253, and numeric ports at five digits (without a port range check).
// Match the authority separately so valid literal percent signs in paths survive.
var urlAuthority = regexp.MustCompile(`(?i)^([a-z0-9.+-]*)://((?:[^\s:@/]+(?::[^\s:@/]*)?@)?)(\[[0-9a-f:.]+\]|[^:/?#]+)(?::[0-9]{1,5})?([/?#][^\s]*)?$`)

const dnsCharacter = `a-z\x{00a1}-\x{ffff}`
const dnsLabel = `[` + dnsCharacter + `0-9](?:[` + dnsCharacter + `0-9-]{0,61}[` + dnsCharacter + `0-9])?`

var dnsHost = regexp.MustCompile(`(?i)^` + dnsLabel + `(?:\.` + dnsLabel + `)*\.(?:[` + dnsCharacter + `][` + dnsCharacter + `-]{0,61}[` + dnsCharacter + `]|xn--[a-z0-9]{1,59})\.?$`)

// ValidURL follows the default Django 5.2 URLValidator string contract.
// Caller-specific field limits, coercion, and scheme rules belong to the caller.
func ValidURL(raw string) bool {
	if utf8.RuneCountInString(raw) > 2048 || strings.ContainsAny(raw, "\t\r\n") {
		return false
	}
	scheme, remainder, present := strings.Cut(raw, "://")
	if !present {
		return false
	}
	switch strings.ToLower(scheme) {
	case "http", "https", "ftp", "ftps":
	default:
		return false
	}
	// Python urlsplit ends the authority at the first path/query/fragment delimiter.
	// Its netloc can differ from the authority matched by Django's full-URL regex.
	netloc := remainder
	if end := strings.IndexAny(netloc, "/?#"); end >= 0 {
		netloc = netloc[:end]
	}
	host, valid := urlHostname(netloc)
	if !valid || host == "" || urlHostnameLength(host) > 253 {
		return false
	}
	normalizedInput := strings.NewReplacer("@", "", ":", "", "#", "", "?", "").Replace(netloc)
	normalized := norm.NFKC.String(normalizedInput)
	if normalized != normalizedInput && strings.ContainsAny(normalized, "/?#@:") {
		return false
	}
	parts := urlAuthority.FindStringSubmatch(raw)
	if parts == nil {
		return false
	}
	// Python's \s excludes Unicode whitespace from credentials and resource
	// text. Its broad DNS character class deliberately has different rules.
	if strings.ContainsFunc(parts[2], pythonWhitespace) || strings.ContainsFunc(parts[4], pythonWhitespace) {
		return false
	}
	regexHost := parts[3]
	if strings.HasPrefix(regexHost, "[") {
		// urlsplit has already checked any brackets in its actual netloc.
		// A bracketed spelling after a query/fragment boundary is regex-only.
		return ipv6URLHost.MatchString(regexHost)
	}
	if address, err := netip.ParseAddr(regexHost); err == nil && address.Is4() {
		return true
	}
	return strings.EqualFold(regexHost, "localhost") || dnsHost.MatchString(regexHost)
}

var ipv6URLHost = regexp.MustCompile(`(?i)^\[[0-9a-f:.]+\]$`)

var ipvFutureHost = regexp.MustCompile(`^v[a-fA-F0-9]+\..+$`)

// urlHostname follows Python 3.12 urlsplit's host and bracket validation.
// It deliberately leaves percent escapes untouched, as the source parser does.
func urlHostname(netloc string) (string, bool) {
	hostPort := netloc
	if at := strings.LastIndexByte(hostPort, '@'); at >= 0 {
		hostPort = hostPort[at+1:]
	}
	host, _, _ := strings.Cut(hostPort, ":")
	if strings.ContainsAny(netloc, "[]") {
		if !strings.Contains(netloc, "[") || !strings.Contains(netloc, "]") {
			return "", false
		}
		before, bracketed, hasBracket := strings.Cut(hostPort, "[")
		if hasBracket {
			if before != "" {
				return "", false
			}
			var after string
			var closed bool
			host, after, closed = strings.Cut(bracketed, "]")
			if !closed || after != "" && !strings.HasPrefix(after, ":") {
				return "", false
			}
		}
		if strings.HasPrefix(host, "v") {
			return host, ipvFutureHost.MatchString(host)
		}
		address, err := netip.ParseAddr(host)
		if err != nil || !address.Is6() {
			return "", false
		}
	}
	return host, true
}

// Python lower() expands U+0130 to i and a combining dot before checking
// hostname length. urlsplit preserves the spelling after an IPv6 zone percent.
func urlHostnameLength(host string) int {
	beforePercent, _, _ := strings.Cut(host, "%")
	return utf8.RuneCountInString(host) + strings.Count(beforePercent, "İ")
}

func pythonWhitespace(r rune) bool {
	return unicode.IsSpace(r) || r >= '\x1c' && r <= '\x1f'
}

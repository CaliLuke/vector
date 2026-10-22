package emailvalidation

import (
	"net/netip"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func NormalizePythonEmail(raw string) string {
	// Python str.strip includes these four separators in addition to Unicode
	// whitespace. Full lowercase mapping retains expansions and final sigma.
	trimmed := strings.TrimFunc(raw, func(c rune) bool {
		return unicode.IsSpace(c) || c >= '\u001c' && c <= '\u001f'
	})
	return cases.Lower(language.Und).String(trimmed)
}

// ValidDjangoEmail follows the pinned Django 5.2 EmailValidator, including
// its quoted local part and Unicode domain branches. It does not normalize the
// address; each invitation action retains its source normalization boundary.
func ValidDjangoEmail(raw string) bool {
	if !utf8.ValidString(raw) || utf8.RuneCountInString(raw) > 320 {
		return false
	}
	at := strings.LastIndexByte(raw, '@')
	if at <= 0 || !validEmailLocal(raw[:at]) {
		return false
	}
	domain := raw[at+1:]
	if domain == "localhost" {
		return true
	}
	if strings.HasPrefix(domain, "[") && strings.HasSuffix(domain, "]") {
		literal := domain[1 : len(domain)-1]
		for _, c := range literal {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F' || c == ':' || c == '.') {
				return false
			}
		}
		_, err := netip.ParseAddr(literal)
		return err == nil
	}
	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}
	for _, label := range labels {
		length := utf8.RuneCountInString(label)
		if length == 0 || length > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !emailDomainLetter(c) && !(c >= '0' && c <= '9') && c != '-' {
				return false
			}
		}
	}
	tld := labels[len(labels)-1]
	if utf8.RuneCountInString(tld) < 2 {
		return false
	}
	letters := true
	for _, c := range tld {
		letters = letters && (emailDomainLetter(c) || c == '-')
	}
	if letters {
		return true
	}
	if len(tld) < 5 || !strings.EqualFold(tld[:4], "xn--") {
		return false
	}
	for _, c := range tld[4:] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func emailDomainLetter(c rune) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '\u00a1' && c <= '\uffff'
}

func validEmailLocal(local string) bool {
	if len(local) >= 2 && local[0] == '"' && local[len(local)-1] == '"' {
		for i := 1; i < len(local)-1; i++ {
			c := local[i]
			if c == '\\' {
				i++
				if i >= len(local)-1 || local[i] == 0 || local[i] == '\n' || local[i] == '\r' || local[i] > 127 {
					return false
				}
			} else if !(c >= 1 && c <= 8 || c == 11 || c == 12 || c >= 14 && c <= 31 || c == '!' || c >= '#' && c <= '[' || c >= ']' && c <= 127) {
				return false
			}
		}
		return true
	}
	for _, atom := range strings.Split(local, ".") {
		if atom == "" {
			return false
		}
		for _, c := range atom {
			// Python's Unicode IGNORECASE adds these four letters to A-Z.
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-!#$%&'*+/=?^_`{}|~İıſK", c)) {
				return false
			}
		}
	}
	return true
}

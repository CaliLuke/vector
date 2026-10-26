package logredact

import (
	"regexp"
	"strings"
)

var (
	secretPattern = regexp.MustCompile(`(?i)(password|secret|authorization|cookie|token)([\w-]*["']?\s*[:=]\s*)("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\r\n]*)`)
	headerPattern = regexp.MustCompile(`(?i)((?:proxy-)?authorization|(?:set-)?cookie)(["']?\s*[:=]\s*["']?)[^\r\n,"']+`)
)

// RedactSensitiveText masks credentials in structured or plain log text.
func RedactSensitiveText(text string) string {
	text = headerPattern.ReplaceAllString(text, `${1}${2}[redacted]`)
	return secretPattern.ReplaceAllStringFunc(text, redactSecret)
}

func redactSecret(value string) string {
	match := secretPattern.FindStringSubmatch(value)
	if len(match) != 4 {
		return value
	}
	secret := match[3]
	quote := ""
	if len(secret) >= 2 && (secret[0] == '\'' || secret[0] == '"') && secret[len(secret)-1] == secret[0] {
		quote = secret[:1]
	}
	return strings.Join([]string{match[1], match[2], quote, "[redacted]", quote}, "")
}

package issuequery

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

type pqlToken struct {
	kind, text string
	pos        int
}

// PQLMaxBytes bounds parsing, including blank queries and alternative API modes.
const PQLMaxBytes = 16 * 1024

// PQL errors deliberately contain no user-supplied query text or literals.
func pqlError(pos int, category string) error {
	return &FilterError{Code: "invalid_pql", Message: fmt.Sprintf("Invalid PQL at byte %d: %s.", pos, category)}
}

func pqlTokens(raw string) ([]pqlToken, error) {
	if len(raw) > PQLMaxBytes || !utf8.ValidString(raw) {
		return nil, pqlError(0, "invalid input size or encoding")
	}
	tokens := []pqlToken{}
	for i := 0; i < len(raw); {
		if strings.ContainsRune(" \t\n\r", rune(raw[i])) {
			i++
			continue
		}
		start := i
		switch {
		case raw[i] == '"':
			i++
			for i < len(raw) && raw[i] != '"' {
				if raw[i] == '\\' {
					i++
				}
				i++
			}
			if i >= len(raw) {
				return nil, pqlError(start, "unterminated string")
			}
			i++
			var value string
			if json.Unmarshal([]byte(raw[start:i]), &value) != nil {
				return nil, pqlError(start, "invalid string")
			}
			tokens = append(tokens, pqlToken{"string", value, start})
		case pqlLetter(raw[i]):
			for i < len(raw) && (pqlLetter(raw[i]) || raw[i] >= '0' && raw[i] <= '9') {
				i++
			}
			tokens = append(tokens, pqlToken{"word", raw[start:i], start})
		case raw[i] >= '0' && raw[i] <= '9' || raw[i] == '-':
			i++
			for i < len(raw) && (raw[i] >= '0' && raw[i] <= '9' || raw[i] == '.') {
				i++
			}
			tokens = append(tokens, pqlToken{"number", raw[start:i], start})
		case strings.ContainsRune("()[],=~<>!", rune(raw[i])):
			i++
			if strings.ContainsRune("<>!", rune(raw[start])) && i < len(raw) && raw[i] == '=' {
				i++
			}
			tokens = append(tokens, pqlToken{"symbol", raw[start:i], start})
		default:
			return nil, pqlError(start, "unexpected character")
		}
	}
	return append(tokens, pqlToken{"end", "", len(raw)}), nil
}

func pqlLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' }

type pqlParser struct {
	tokens            []pqlToken
	index, conditions int
}

func (p *pqlParser) peek() pqlToken { return p.tokens[p.index] }
func (p *pqlParser) take() pqlToken {
	t := p.peek()
	if t.kind != "end" {
		p.index++
	}
	return t
}
func (p *pqlParser) match(s string) bool {
	t := p.peek()
	if t.kind != "word" && t.kind != "symbol" {
		return false
	}
	if !strings.EqualFold(t.text, s) {
		return false
	}
	p.take()
	return true
}
func (p *pqlParser) expect(s string) error {
	if !p.match(s) {
		return pqlError(p.peek().pos, "unexpected token")
	}
	return nil
}

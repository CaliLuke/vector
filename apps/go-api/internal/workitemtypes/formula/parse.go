package formula

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type token struct{ kind, text string }
type parser struct {
	tokens []token
	index  int
	fields map[string]Field
}

// Compile resolves references to stable IDs and checks every branch's types,
// including branches that may be skipped by IF at evaluation time.
func Compile(source string, result Kind, fields map[string]Field) (*Expression, error) {
	if !validKind(result) {
		return nil, fmt.Errorf("unsupported formula result kind %q", result)
	}
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	refs := make(map[string]Field, len(fields))
	for name, field := range fields {
		key := ReferenceKey(name)
		if key == "" || field.ID == "" || !validKind(field.Kind) {
			return nil, fmt.Errorf("invalid field reference %q", name)
		}
		if _, exists := refs[key]; exists {
			return nil, fmt.Errorf("ambiguous field reference %q", name)
		}
		refs[key] = field
	}
	p := parser{tokens: tokens, fields: refs}
	root, err := p.expression(0, 0)
	if err != nil {
		return nil, err
	}
	if p.peek().kind != "end" {
		return nil, fmt.Errorf("unexpected token %q", p.peek().text)
	}
	if root.kind != result {
		return nil, fmt.Errorf("expression returns %s, expected %s", root.kind, result)
	}
	return &Expression{root: root}, nil
}

func lex(source string) ([]token, error) {
	if len(source) == 0 || len(source) > 4096 || !utf8.ValidString(source) {
		return nil, fmt.Errorf("enter a valid expression of at most 4096 bytes")
	}
	var out []token
	for i := 0; i < len(source); {
		if len(out) >= 1024 {
			return nil, fmt.Errorf("expression exceeds 1024 tokens")
		}
		r, size := utf8.DecodeRuneInString(source[i:])
		if unicode.IsSpace(r) {
			i += size
			continue
		}
		start := i
		if strings.HasPrefix(source[i:], "{{") {
			end := strings.Index(source[i+2:], "}}")
			if end < 0 {
				return nil, fmt.Errorf("unclosed field reference")
			}
			out = append(out, token{"field", source[i+2 : i+2+end]})
			i += end + 4
			continue
		}
		if source[i] == '"' {
			i++
			for i < len(source) && source[i] != '"' {
				if source[i] == '\\' {
					i++
				}
				i++
			}
			if i >= len(source) {
				return nil, fmt.Errorf("unclosed text literal")
			}
			i++
			var decoded string
			if err := json.Unmarshal([]byte(source[start:i]), &decoded); err != nil {
				return nil, fmt.Errorf("invalid text literal")
			}
			out = append(out, token{"text", decoded})
			continue
		}
		if source[i] >= '0' && source[i] <= '9' || source[i] == '.' {
			for i < len(source) && (source[i] >= '0' && source[i] <= '9' || source[i] == '.') {
				i++
			}
			out = append(out, token{"number", source[start:i]})
			continue
		}
		if unicode.IsLetter(r) {
			for i < len(source) {
				r, size = utf8.DecodeRuneInString(source[i:])
				if !unicode.IsLetter(r) {
					break
				}
				i += size
			}
			out = append(out, token{"word", strings.ToUpper(source[start:i])})
			continue
		}
		i++
		if strings.ContainsRune("<>=!", r) && i < len(source) && source[i] == '=' {
			i++
		}
		op := source[start:i]
		if !strings.Contains(" +-*/&=!=<<=>>=(), ", op) || op == "!" {
			return nil, fmt.Errorf("unexpected character %q", r)
		}
		out = append(out, token{op, op})
	}
	return append(out, token{"end", ""}), nil
}

func (p *parser) peek() token { return p.tokens[p.index] }
func (p *parser) take() token {
	t := p.peek()
	if t.kind != "end" {
		p.index++
	}
	return t
}
func (p *parser) expect(kind string) error {
	if p.take().kind != kind {
		return fmt.Errorf("expected %s", kind)
	}
	return nil
}
func precedence(op string) int {
	switch op {
	case "=", "!=", "<", "<=", ">", ">=":
		return 1
	case "&":
		return 2
	case "+", "-":
		return 3
	case "*", "/":
		return 4
	default:
		return 0
	}
}
func (p *parser) expression(minimum, depth int) (*node, error) {
	if depth > 32 {
		return nil, fmt.Errorf("expression exceeds 32 nesting levels")
	}
	left, err := p.primary(depth + 1)
	if err != nil {
		return nil, err
	}
	compared := false
	for {
		op := p.peek().kind
		rank := precedence(op)
		if rank == 0 || rank < minimum {
			break
		}
		if rank == 1 && compared {
			return nil, fmt.Errorf("chained comparisons are unsupported")
		}
		p.take()
		right, parseErr := p.expression(rank+1, depth+1)
		if parseErr != nil {
			return nil, parseErr
		}
		kind, typeErr := binaryKind(op, left.kind, right.kind)
		if typeErr != nil {
			return nil, typeErr
		}
		left = &node{op: op, kind: kind, args: []*node{left, right}}
		compared = rank == 1
	}
	return left, nil
}
func (p *parser) primary(depth int) (*node, error) {
	t := p.take()
	switch t.kind {
	case "number":
		n, err := strconv.ParseFloat(t.text, 64)
		value := Value{Kind: Number, Number: n}
		if err != nil || value.validate() != nil {
			return nil, fmt.Errorf("invalid finite number %q", t.text)
		}
		return &node{op: "literal", kind: Number, value: value}, nil
	case "text":
		return &node{op: "literal", kind: Text, value: Value{Kind: Text, Text: t.text}}, nil
	case "field":
		f, ok := p.fields[ReferenceKey(t.text)]
		if !ok {
			return nil, fmt.Errorf("unknown or ineligible field %q", t.text)
		}
		return &node{op: "field", kind: f.Kind, field: f.ID}, nil
	case "word":
		if t.text == "TRUE" || t.text == "FALSE" {
			return &node{op: "literal", kind: Boolean, value: Value{Kind: Boolean, Boolean: t.text == "TRUE"}}, nil
		}
		return p.call(t.text, depth)
	case "(":
		n, err := p.expression(0, depth)
		if err != nil {
			return nil, err
		}
		return n, p.expect(")")
	case "+", "-":
		n, err := p.expression(5, depth)
		if err != nil {
			return nil, err
		}
		if n.kind != Number {
			return nil, fmt.Errorf("unary signs require a number")
		}
		return &node{op: "unary" + t.kind, kind: Number, args: []*node{n}}, nil
	default:
		return nil, fmt.Errorf("expected a value, got %q", t.text)
	}
}
func (p *parser) call(name string, depth int) (*node, error) {
	if err := p.expect("("); err != nil {
		return nil, err
	}
	var args []*node
	if p.peek().kind != ")" {
		for {
			n, err := p.expression(0, depth)
			if err != nil {
				return nil, err
			}
			args = append(args, n)
			if p.peek().kind != "," {
				break
			}
			p.take()
		}
	}
	if err := p.expect(")"); err != nil {
		return nil, err
	}
	kind, err := functionKind(name, args)
	if err != nil {
		return nil, err
	}
	return &node{op: name, kind: kind, args: args}, nil
}

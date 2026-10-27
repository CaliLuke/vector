package issuequery

import "strings"

type pqlExpression struct {
	op, field    string
	propertyID   string
	pos          int
	left, right  *pqlExpression
	values       []pqlValue
	listFunction *pqlValue
}

type pqlValue struct {
	pqlToken
	arguments []pqlValue
}

func parsePQL(raw string) (*pqlExpression, error) {
	tokens, err := pqlTokens(raw)
	if err != nil {
		return nil, err
	}
	p := pqlParser{tokens: tokens}
	if p.peek().kind == "end" {
		return nil, nil
	}
	expression, err := p.logic(0, 0)
	if err != nil {
		return nil, err
	}
	if p.peek().kind != "end" {
		return nil, pqlError(p.peek().pos, "trailing input")
	}
	return expression, nil
}

func (p *pqlParser) logic(level, depth int) (*pqlExpression, error) {
	if level == 2 {
		return p.unary(depth)
	}
	left, err := p.logic(level+1, depth)
	if err != nil {
		return nil, err
	}
	op := []string{"OR", "AND"}[level]
	for p.match(op) {
		right, e := p.logic(level+1, depth)
		if e != nil {
			return nil, e
		}
		left = &pqlExpression{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *pqlParser) unary(depth int) (*pqlExpression, error) {
	if depth >= 64 {
		return nil, pqlError(p.peek().pos, "nesting limit exceeded")
	}
	if p.match("NOT") {
		child, err := p.unary(depth + 1)
		return &pqlExpression{op: "NOT", left: child}, err
	}
	if p.match("(") {
		child, err := p.logic(0, depth+1)
		if err != nil {
			return nil, err
		}
		return child, p.expect(")")
	}
	p.conditions++
	if p.conditions > 5 {
		return nil, pqlError(p.peek().pos, "condition limit exceeded")
	}
	return p.condition(depth)
}

func (p *pqlParser) condition(depth int) (*pqlExpression, error) {
	field := p.take()
	if field.kind != "word" {
		return nil, pqlError(field.pos, "expected field or predicate")
	}
	node := &pqlExpression{field: field.text, pos: field.pos}
	if p.match("(") {
		values, err := p.values(depth + 1)
		node.op, node.values = "predicate", values
		return node, err
	}
	if field.text == "cf" && p.match("[") {
		property := p.take()
		if property.kind != "string" {
			return nil, pqlError(property.pos, "expected property identifier")
		}
		if err := p.expect("]"); err != nil {
			return nil, err
		}
		// Preserve the field category without putting a caller's ID in errors.
		node.field = "cf"
		node.propertyID = property.text
	}
	op := p.take()
	node.op = op.text
	switch {
	case op.kind == "word":
		return p.wordOperator(node, op, depth)
	case op.kind == "symbol" && (op.text == "=" || op.text == "!=" || op.text == "~" || op.text == ">" || op.text == ">=" || op.text == "<" || op.text == "<="):
		value, err := p.value(depth)
		node.values = []pqlValue{value}
		return node, err
	default:
		return nil, pqlError(op.pos, "expected comparison operator")
	}
}

func (p *pqlParser) wordOperator(node *pqlExpression, op pqlToken, depth int) (*pqlExpression, error) {
	// Keywords are case insensitive; identifiers and function names are not.
	switch {
	case strings.EqualFold(op.text, "IS"):
		node.op = "IS"
		if p.match("NOT") {
			node.op += " NOT"
		}
		if p.match("NULL") {
			node.op += " NULL"
		} else if p.match("EMPTY") {
			node.op += " EMPTY"
		} else {
			return nil, pqlError(p.peek().pos, "expected NULL or EMPTY")
		}
		return node, nil
	case strings.EqualFold(op.text, "IN"), strings.EqualFold(op.text, "NOT"):
		node.op = "IN"
		if strings.EqualFold(op.text, "NOT") {
			if err := p.expect("IN"); err != nil {
				return nil, err
			}
			node.op = "NOT IN"
		}
		if p.match("(") {
			values, err := p.values(depth + 1)
			node.values = values
			if err == nil && len(values) == 0 {
				err = pqlError(op.pos, "empty list")
			}
			return node, err
		}
		value, err := p.value(depth)
		if err != nil {
			return nil, err
		}
		if value.kind != "call" {
			return nil, pqlError(value.pos, "expected list or list function")
		}
		node.listFunction = &value
		return node, nil
	case strings.EqualFold(op.text, "BETWEEN"):
		node.op = "BETWEEN"
		parenthesized := p.match("(")
		first, err := p.value(depth)
		if err != nil {
			return nil, err
		}
		separator := "AND"
		if parenthesized {
			separator = ","
		}
		if err = p.expect(separator); err != nil {
			return nil, err
		}
		last, err := p.value(depth)
		if err != nil {
			return nil, err
		}
		if parenthesized {
			err = p.expect(")")
		}
		node.values = []pqlValue{first, last}
		return node, err
	default:
		return nil, pqlError(op.pos, "unknown operator")
	}
}

func (p *pqlParser) value(depth int) (pqlValue, error) {
	token := p.take()
	v := pqlValue{pqlToken: token}
	switch token.kind {
	case "string", "number":
		return v, nil
	case "word":
		if token.text == "true" || token.text == "false" {
			v.kind = "boolean"
			return v, nil
		}
		if p.match("(") {
			v.kind = "call"
			var err error
			v.arguments, err = p.values(depth + 1)
			return v, err
		}
	}
	return v, pqlError(token.pos, "expected literal or value function")
}

func (p *pqlParser) values(depth int) ([]pqlValue, error) {
	if depth >= 64 {
		return nil, pqlError(p.peek().pos, "nesting limit exceeded")
	}
	values := []pqlValue{}
	if p.match(")") {
		return values, nil
	}
	for {
		if len(values) == 256 {
			return nil, pqlError(p.peek().pos, "list limit exceeded")
		}
		value, err := p.value(depth)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		if p.match(")") {
			return values, nil
		}
		if err = p.expect(","); err != nil {
			return nil, err
		}
	}
}

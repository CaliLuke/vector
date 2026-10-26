package formula

import (
	"cmp"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func (n *node) evaluate(values map[string]Value, instant time.Time) (Value, error) {
	switch n.op {
	case "literal":
		return n.value, nil
	case "field":
		v, exists := values[n.field]
		if !exists {
			return Value{Kind: n.kind, Empty: true}, nil
		}
		if v.Kind != n.kind {
			return Value{}, fmt.Errorf("field %s has an unexpected value kind", n.field)
		}
		return v, v.validate()
	case "IF":
		condition, err := n.args[0].evaluate(values, instant)
		if err != nil {
			return Value{}, err
		}
		if condition.Empty {
			return Value{Kind: n.kind, Empty: true}, nil
		}
		branch := 2
		if condition.Boolean {
			branch = 1
		}
		return n.args[branch].evaluate(values, instant)
	}
	args := make([]Value, len(n.args))
	var firstError error
	empty := false
	for i, arg := range n.args {
		value, err := arg.evaluate(values, instant)
		if err != nil && firstError == nil {
			firstError = err
		}
		empty = empty || value.Empty
		args[i] = value
	}
	if firstError != nil {
		return Value{}, firstError
	}
	if empty {
		return Value{Kind: n.kind, Empty: true}, nil
	}
	value, err := n.apply(args, instant)
	if err != nil {
		return Value{}, err
	}
	return value, value.validate()
}

func (n *node) apply(args []Value, instant time.Time) (Value, error) {
	out := Value{Kind: n.kind}
	switch n.op {
	case "TODAY":
		out.Date = time.Date(instant.Year(), instant.Month(), instant.Day(), 0, 0, 0, 0, time.UTC)
		out.DateOnly = true
	case "NOW":
		out.Date = instant
	case "unary+":
		out.Number = args[0].Number
	case "unary-":
		out.Number = -args[0].Number
	case "ABS":
		out.Number = math.Abs(args[0].Number)
	case "ROUND":
		digits := 0.0
		if len(args) == 2 {
			digits = args[1].Number
		}
		if digits != math.Trunc(digits) || math.Abs(digits) > 15 {
			return out, fmt.Errorf("ROUND precision must be an integer from -15 to 15")
		}
		out.Number = roundDecimal(args[0].Number, int(digits))
	case "UPPER":
		out.Text = strings.ToUpper(args[0].Text)
	case "LOWER":
		out.Text = strings.ToLower(args[0].Text)
	case "LEN":
		out.Number = float64(utf8.RuneCountInString(args[0].Text))
	case "CONCAT", "&":
		var b strings.Builder
		for _, v := range args {
			text := v.scalarText()
			if b.Len()+len(text) > maxText {
				return out, fmt.Errorf("text result exceeds %d bytes", maxText)
			}
			b.WriteString(text)
		}
		out.Text = b.String()
	default:
		return binary(n.op, args[0], args[1])
	}
	return out, nil
}

// Round the shortest decimal representation, rather than a binary product that
// can move decimal ties such as 1.005 below the rounding boundary.
func roundDecimal(value float64, digits int) float64 {
	rational, _ := new(big.Rat).SetString(strconv.FormatFloat(value, 'f', -1, 64))
	exponent := digits
	if exponent < 0 {
		exponent = -exponent
	}
	factor := new(big.Rat).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil))
	if digits < 0 {
		factor.Inv(factor)
	}
	rational.Mul(rational, factor)
	numerator := new(big.Int).Abs(rational.Num())
	integer, remainder := new(big.Int), new(big.Int)
	integer.QuoRem(numerator, rational.Denom(), remainder)
	if remainder.Lsh(remainder, 1).Cmp(rational.Denom()) >= 0 {
		integer.Add(integer, big.NewInt(1))
	}
	if value < 0 {
		integer.Neg(integer)
	}
	rounded, _ := new(big.Rat).Quo(new(big.Rat).SetInt(integer), factor).Float64()
	return rounded
}

func binary(op string, left, right Value) (Value, error) {
	if op == "=" || op == "!=" || op == "<" || op == "<=" || op == ">" || op == ">=" {
		comparison := compare(left, right)
		result := false
		switch op {
		case "=":
			result = comparison == 0
		case "!=":
			result = comparison != 0
		case "<":
			result = comparison < 0
		case "<=":
			result = comparison <= 0
		case ">":
			result = comparison > 0
		case ">=":
			result = comparison >= 0
		}
		return Value{Kind: Boolean, Boolean: result}, nil
	}
	if left.Kind == Date {
		return dateArithmetic(op, left, right)
	}
	out := Value{Kind: Number}
	switch op {
	case "+":
		out.Number = left.Number + right.Number
	case "-":
		out.Number = left.Number - right.Number
	case "*":
		out.Number = left.Number * right.Number
	case "/":
		if right.Number == 0 {
			return out, fmt.Errorf("division by zero")
		}
		out.Number = left.Number / right.Number
	}
	return out, nil
}

func compare(left, right Value) int {
	switch left.Kind {
	case Number:
		return cmp.Compare(left.Number, right.Number)
	case Text:
		return strings.Compare(left.Text, right.Text)
	case Date:
		return left.Date.Compare(right.Date)
	default:
		if left.Boolean == right.Boolean {
			return 0
		}
		if left.Boolean {
			return 1
		}
		return -1
	}
}

func dateArithmetic(op string, left, right Value) (Value, error) {
	if right.Kind == Date {
		// time.Duration saturates beyond about 292 years; Unix seconds keep
		// supported calendar dates comparable across the entire allowed range.
		seconds := float64(left.Date.Unix()-right.Date.Unix()) + float64(left.Date.Nanosecond()-right.Date.Nanosecond())/1e9
		return Value{Kind: Number, Number: seconds / 86400}, nil
	}
	days := right.Number
	if op == "-" {
		days = -days
	}
	if math.Abs(days) > 3652059 {
		return Value{}, fmt.Errorf("date arithmetic exceeds the supported range")
	}
	if left.DateOnly && days != math.Trunc(days) {
		return Value{}, fmt.Errorf("date-only arithmetic requires whole days")
	}
	whole, fraction := math.Modf(days)
	date := left.Date.UTC().AddDate(0, 0, int(whole)).Add(time.Duration(fraction * 86400 * 1e9))
	return Value{Kind: Date, Date: date, DateOnly: left.DateOnly}, nil
}

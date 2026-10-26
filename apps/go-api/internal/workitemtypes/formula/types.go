package formula

import "fmt"

func binaryKind(op string, left, right Kind) (Kind, error) {
	switch op {
	case "&":
		return Text, nil
	case "+", "-":
		if left == Number && right == Number {
			return Number, nil
		}
		if left == Date && right == Number {
			return Date, nil
		}
		if op == "-" && left == Date && right == Date {
			return Number, nil
		}
	case "*", "/":
		if left == Number && right == Number {
			return Number, nil
		}
	case "=", "!=":
		if left == right {
			return Boolean, nil
		}
	case "<", "<=", ">", ">=":
		if left == right && left != Boolean {
			return Boolean, nil
		}
	}
	return "", fmt.Errorf("operator %s does not accept %s and %s", op, left, right)
}

func functionKind(name string, args []*node) (Kind, error) {
	switch name {
	case "NOW", "TODAY":
		if len(args) == 0 {
			return Date, nil
		}
	case "IF":
		if len(args) == 3 && args[0].kind == Boolean && args[1].kind == args[2].kind {
			return args[1].kind, nil
		}
	case "ABS":
		if len(args) == 1 && args[0].kind == Number {
			return Number, nil
		}
	case "ROUND":
		if (len(args) == 1 || len(args) == 2 && args[1].kind == Number) && args[0].kind == Number {
			return Number, nil
		}
	case "UPPER", "LOWER", "LEN":
		if len(args) == 1 && args[0].kind == Text {
			if name == "LEN" {
				return Number, nil
			}
			return Text, nil
		}
	case "CONCAT":
		if len(args) > 0 {
			return Text, nil
		}
	default:
		return "", fmt.Errorf("unknown function %s", name)
	}
	return "", fmt.Errorf("invalid arguments for %s", name)
}

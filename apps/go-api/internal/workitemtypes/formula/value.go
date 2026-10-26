// Package formula implements the bounded, typed work-item formula language.
// Expressions never execute host code or resolve fields outside the supplied catalog.
package formula

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Kind string

const (
	Number  Kind = "Number"
	Text    Kind = "Text"
	Date    Kind = "Date"
	Boolean Kind = "Boolean"
	maxText      = 65536
)

// Value distinguishes an absent input from false, zero, and a runtime error.
type Value struct {
	Kind     Kind
	Empty    bool
	Number   float64
	Text     string
	Boolean  bool
	Date     time.Time
	DateOnly bool
}

type Field struct {
	ID   string
	Kind Kind
}

// ReferenceKey is shared by field suggestions and compilation. Persisted field
// references remain immutable when the visible property title changes.
func ReferenceKey(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.ReplaceAll(name, "_", " ")), "_"))
}

func validKind(kind Kind) bool {
	return kind == Number || kind == Text || kind == Date || kind == Boolean
}

func (v Value) validate() error {
	if !validKind(v.Kind) {
		return fmt.Errorf("unsupported result kind %q", v.Kind)
	}
	if v.Empty {
		return nil
	}
	switch v.Kind {
	case Number:
		if math.IsNaN(v.Number) || math.IsInf(v.Number, 0) {
			return fmt.Errorf("numeric result is not finite")
		}
	case Text:
		if len(v.Text) > maxText {
			return fmt.Errorf("text result exceeds %d bytes", maxText)
		}
	case Date:
		if v.Date.Year() < 1 || v.Date.Year() > 9999 {
			return fmt.Errorf("date is outside the supported range")
		}
	}
	return nil
}

func (v Value) scalarText() string {
	switch v.Kind {
	case Number:
		return strconv.FormatFloat(v.Number, 'f', -1, 64)
	case Boolean:
		return strconv.FormatBool(v.Boolean)
	case Date:
		if v.DateOnly {
			return v.Date.UTC().Format(time.DateOnly)
		}
		return v.Date.UTC().Format(time.RFC3339Nano)
	default:
		return v.Text
	}
}

type node struct {
	op    string
	kind  Kind
	value Value
	field string
	args  []*node
}

type Expression struct{ root *node }

// Evaluate uses one caller-supplied instant for NOW/TODAY across every row in a
// page/filter operation. Missing fields produce empty results, not zero values.
func (e *Expression) Evaluate(values map[string]Value, instant time.Time) (Value, error) {
	return e.root.evaluate(values, instant.UTC())
}

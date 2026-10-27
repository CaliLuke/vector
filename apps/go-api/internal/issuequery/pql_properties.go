package issuequery

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"uuid"
)

func (c *pqlCompiler) property(e *pqlExpression) (string, error) {
	id, err := uuid.Parse(e.propertyID)
	if err != nil {
		return "", pqlError(e.pos, "invalid property identifier")
	}
	definition, err := c.propertyDefinition(id.String(), e.pos)
	if err != nil {
		return "", err
	}
	if e.listFunction != nil || !pqlPropertyOperator(definition.Kind, e.op) {
		return "", pqlError(e.pos, "operator or function is not valid for property type")
	}
	parameter := c.propertyParameter(id.String())
	value := "(" + parameter + " -> issue.id::text)"
	column := "(" + value + " #>> '{}')::" + string(definition.Kind)
	op := e.op
	switch op {
	case "IS EMPTY":
		op = "IS NULL"
	case "IS NOT EMPTY":
		op = "IS NOT NULL"
	}
	var comparison string
	if op == "IS NULL" || op == "IS NOT NULL" {
		comparison = "(" + value + " #>> '{}') " + op
	} else {
		operands, operandErr := c.propertyOperands(e, definition)
		if operandErr != nil {
			return "", operandErr
		}
		comparison = pqlComparison(column, op, operands)
		if definition.Kind == PQLPropertySelection {
			comparison = `EXISTS(SELECT 1 FROM jsonb_array_elements_text(` + value + `) selected(value) WHERE selected.value::uuid IN (` + strings.Join(operands, ",") + `))`
			if op == "!=" || op == "NOT IN" {
				comparison = "NOT " + comparison
			}
			comparison = `(CASE WHEN ` + value + `='null'::jsonb THEN NULL ELSE ` + comparison + ` END)`
		}
	}
	return `(CASE WHEN ` + parameter + ` ? issue.id::text THEN ` + comparison + ` ELSE NULL END)`, nil
}

func (c *pqlCompiler) propertyDefinition(id string, pos int) (PQLPropertyDefinition, error) {
	if definition, exists := c.properties[id]; exists {
		return definition, nil
	}
	if c.resolveProperty == nil || !c.preparingProperties {
		return PQLPropertyDefinition{}, pqlError(pos, "property is unavailable")
	}
	definition, found, err := c.resolveProperty(id)
	if err != nil {
		return definition, err
	}
	if !found {
		return definition, pqlError(pos, "property is unavailable")
	}
	c.properties[id] = definition
	return definition, nil
}

func (c *pqlCompiler) propertyParameter(id string) string {
	if c.propertyParameters == nil {
		c.propertyParameters = map[string]string{}
	}
	if parameter, exists := c.propertyParameters[id]; exists {
		return parameter
	}
	value := "{}"
	if !c.preparingProperties {
		value = c.propertyValues[id]
	}
	c.args = append(c.args, value)
	parameter := fmt.Sprintf("$%d::jsonb", len(c.args))
	c.propertyParameters[id] = parameter
	return parameter
}

func pqlPropertyOperator(kind PQLPropertyKind, op string) bool {
	switch kind {
	case PQLPropertyText:
		return op != "IN" && op != "NOT IN" && pqlOperatorAllowed("text", op)
	case PQLPropertyNumber, PQLPropertyDate:
		return pqlOperatorAllowed("date", op)
	case PQLPropertySelection:
		return pqlOperatorAllowed("uuid", op)
	case PQLPropertyBoolean:
		return op == "=" || op == "IS NULL" || op == "IS NOT NULL"
	}
	return false
}

func (c *pqlCompiler) propertyOperands(e *pqlExpression, definition PQLPropertyDefinition) ([]string, error) {
	if definition.Kind == PQLPropertyNumber {
		operands := make([]string, len(e.values))
		for index, value := range e.values {
			number, err := strconv.ParseFloat(value.text, 64)
			if value.kind != "number" || !json.Valid([]byte(value.text)) || err != nil || math.IsInf(number, 0) || math.IsNaN(number) {
				return nil, pqlError(value.pos, "invalid numeric property operand")
			}
			c.args = append(c.args, value.text)
			operands[index] = fmt.Sprintf("$%d::numeric", len(c.args))
		}
		return operands, nil
	}
	kind := string(definition.Kind)
	if definition.Kind == PQLPropertySelection {
		kind = "uuid"
		if c.preparingProperties {
			if err := c.validatePropertySelections(e, definition); err != nil {
				return nil, err
			}
		}
	}
	return c.operands(kind, e)
}

func (c *pqlCompiler) validatePropertySelections(e *pqlExpression, definition PQLPropertyDefinition) error {
	ids := make([]string, len(e.values))
	for index, value := range e.values {
		id, err := c.operand("uuid", "cf", value)
		if err != nil {
			return err
		}
		ids[index] = id.(string)
	}
	if definition.ValidateSelections == nil {
		return pqlError(e.pos, "property selections are unavailable")
	}
	valid, err := definition.ValidateSelections(ids)
	if err != nil {
		return err
	}
	if !valid {
		return pqlError(e.pos, "property selections are unavailable")
	}
	return nil
}

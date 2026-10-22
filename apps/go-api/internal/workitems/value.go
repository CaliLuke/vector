package workitems

import (
	"encoding/json"
	"math/big"
	"reflect"
)

func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case string:
		return v != ""
	case json.Number:
		n, ok := new(big.Rat).SetString(string(v))
		return !ok || n.Sign() != 0
	default:
		reflected := reflect.ValueOf(value)
		switch reflected.Kind() {
		case reflect.Slice, reflect.Map, reflect.Array:
			return reflected.Len() != 0
		case reflect.Int, reflect.Int64:
			return reflected.Int() != 0
		case reflect.Float32, reflect.Float64:
			return reflected.Float() != 0
		}
		return true
	}
}

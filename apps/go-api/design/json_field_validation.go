package design

import . "github.com/CaliLuke/loom/dsl"

// JSONFieldValidation preserves DRF's field errors for non-finite parsed numbers.
var jsonFieldValidation = Type("JSONFieldValidation", MapOf(String, ArrayOf(String)))

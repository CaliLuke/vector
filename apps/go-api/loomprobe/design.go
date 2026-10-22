package design

import . "github.com/CaliLuke/loom/dsl"

var nestedObject = Type("ProbeNestedObject", func() {
	Attribute("label", String)
	Required("label")
})
var nestedChoice = Type("ProbeNestedChoice", OneOf(String, nestedObject))
var scalarArray = Type("ProbeScalarArray", ArrayOf(Int))
var scalarOrArray = Type("ProbeScalarOrArray", OneOf(String, scalarArray))
var record = Type("ProbeRecord", func() {
	Attribute("id", String)
	Attribute("display", String)
	Attribute("choice", nestedChoice)
	Attribute("children", ArrayOf(nestedChoice))
	Attribute("value", scalarOrArray)
	Required("id", "display", "choice", "children", "value")
})
var recordList = Type("ProbeRecordList", ArrayOf(record))
var page = Type("ProbePage", func() {
	Attribute("results", ArrayOf(record))
	Attribute("next", String, func() { Nullable() })
	Required("results", "next")
})
var scalarOrObject = Type("ProbeScalarOrObject", OneOf(String, record))
var rawMetadata = Type("ProbeRawMetadata", func() {
	Attribute("content_type", String)
	Required("content_type")
})
var _ = Service("probe", func() {
	Method("list", func() {
		Result(rawMetadata)
		HTTP(func() {
			GET("/probe/")
			SkipResponseBodyEncodeDecode()
			Response(200, func() {
				Header("content_type:Content-Type")
				ContentType("application/json")
				OpenAPIBody(OneOf(recordList, page), Untagged)
			})
		})
	})
	Method("scalar_or_object", func() {
		Result(rawMetadata)
		HTTP(func() {
			GET("/probe/union/")
			SkipResponseBodyEncodeDecode()
			Response(200, func() {
				Header("content_type:Content-Type")
				ContentType("application/json")
				OpenAPIBody(scalarOrObject, Untagged)
			})
		})
	})
})

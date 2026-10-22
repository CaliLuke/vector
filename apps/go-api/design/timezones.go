package design

import . "github.com/CaliLuke/loom/dsl"

var timezone = Type("Timezone", func() {
	Attribute("gmt_offset", String)
	Attribute("label", String)
	Attribute("utc_offset", String)
	Attribute("value", String)
	Required("gmt_offset", "label", "utc_offset", "value")
})

var timezoneList = Type("TimezoneList", func() {
	Attribute("timezones", ArrayOf(timezone))
	Required("timezones")
})

var _ = Service("timezones", func() {
	Description("Lists the timezones that the Plane user interface supports.")
	HTTP(func() {
		RuntimeCORS()
	})

	Method("list", func() {
		Result(timezoneList)
		HTTP(func() {
			GET("/api/timezones/")
			Response(StatusOK)
		})
	})
})

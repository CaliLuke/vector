package djangoinput

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSerializerFormFieldsRetainLastValuePartialCheckboxAndJSONSemantics(t *testing.T) {
	input, err := Parse(strings.NewReader(`name=first&name=last&icon_prop=%7B%22type%22%3A%22icon%22%7D&archive_in=&project_lead=&cover_image=`), "application/x-www-form-urlencoded")
	if err != nil {
		t.Fatal(err)
	}
	if value, present := input.FieldValue("name", true, false, false); !present || value != "last" {
		t.Fatalf("repeated name=%v,%v", value, present)
	}
	if value, present := input.BooleanFieldValue("page_view", false); !present || value != false {
		t.Fatalf("full checkbox=%v,%v", value, present)
	}
	if _, present := input.BooleanFieldValue("page_view", true); present {
		t.Fatal("partial omitted checkbox was supplied")
	}
	if _, present := input.FieldValue("archive_in", false, false, false); present {
		t.Fatal("optional empty integer was supplied")
	}
	if value, present := input.FieldValue("project_lead", false, false, true); !present || value != nil {
		t.Fatalf("empty relation=%v,%v", value, present)
	}
	if value, present := input.FieldValue("cover_image", false, true, true); !present || value != "" {
		t.Fatalf("blank text=%v,%v", value, present)
	}
	value, present, err := input.JSONFieldValue("icon_prop")
	if err != nil || !present || value.(map[string]any)["type"] != "icon" {
		t.Fatalf("form JSON=%v,%v,%v", value, present, err)
	}
	if input.Fields["icon_prop"] != `{"type":"icon"}` || len(input.FormValues["name"]) != 2 {
		t.Fatal("serializer coercion changed task/request input")
	}
}

func TestSerializerJSONFieldKeepsJSONStringsAndRejectsMalformedFormJSON(t *testing.T) {
	input, err := Parse(strings.NewReader(`{"icon_prop":"{bad}","description_html":{"z":1,"a":2}}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	value, _, err := input.JSONFieldValue("icon_prop")
	if err != nil || value != "{bad}" {
		t.Fatalf("JSON string=%v,%v", value, err)
	}
	if string(input.JSONFieldSource("description_html")) != `{"z":1,"a":2}` {
		t.Fatal("original JSON field order was lost")
	}
	input = Input{Form: true, Fields: map[string]any{"icon_prop": "{bad}"}}
	if _, _, err = input.JSONFieldValue("icon_prop"); err == nil {
		t.Fatal("malformed form JSON accepted")
	}
	input.Fields["icon_prop"] = `1.00`
	value, _, err = input.JSONFieldValue("icon_prop")
	if err != nil || value != json.Number("1.00") {
		t.Fatalf("form JSON number=%v,%v", value, err)
	}
}

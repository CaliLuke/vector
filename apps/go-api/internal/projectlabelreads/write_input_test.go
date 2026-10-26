package projectlabelreads

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPostFieldValidationMatchesRecordedSerializerCases(t *testing.T) {
	for _, scenario := range []struct {
		name, body string
		want       map[string]any
	}{
		{"missing name", `{}`, map[string]any{"name": []string{"This field is required."}}},
		{"null name", `{"name":null}`, map[string]any{"name": []string{"This field may not be null."}}},
		{"boolean name", `{"name":true}`, map[string]any{"name": []string{"Not a valid string."}}},
		{"blank name", `{"name":"  \t "}`, map[string]any{"name": []string{"This field may not be blank."}}},
		{"null character", `{"name":"nul\u0000name"}`, map[string]any{"name": []string{"Null characters are not allowed."}}},
		{"long name", `{"name":"` + strings.Repeat("x", 256) + `"}`, map[string]any{"name": []string{"Ensure this field has no more than 255 characters."}}},
		{"null color", `{"name":"Color null","color":null}`, map[string]any{"color": []string{"This field may not be null."}}},
		{"long color", `{"name":"Color limit","color":"` + strings.Repeat("x", 256) + `"}`, map[string]any{"color": []string{"Ensure this field has no more than 255 characters."}}},
		{"null sort order", `{"name":"Null order","sort_order":null}`, map[string]any{"sort_order": []string{"This field may not be null."}}},
		{"bad sort order", `{"name":"Bad order","sort_order":"invalid"}`, map[string]any{"sort_order": []string{"A valid number is required."}}},
		{"malformed parent", `{"name":"Child","parent":"not-a-uuid"}`, map[string]any{"parent": []string{"“not-a-uuid” is not a valid UUID."}}},
		{"unpaired surrogate", `{"name":"surrogate\ud800"}`, map[string]any{"name": []string{"Surrogate characters are not allowed: U+D800."}}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := &parentWriteDatabase{label: existingWriteLabel()}
			repository := newWriteRepository(t, db)
			_, err := repository.Write(context.Background(), "user", "workspace", writeProject, "", http.MethodPost, strings.NewReader(scenario.body), "application/json")
			assertWriteHTTPError(t, err, 400, scenario.want)
			if db.saved {
				t.Fatal("invalid input changed label state")
			}
		})
	}
}

func TestPostCollectsIndependentFieldErrors(t *testing.T) {
	db := &parentWriteDatabase{label: existingWriteLabel()}
	db.row = func(query string, _ []any) pgx.Row {
		if strings.Contains(query, "FOR KEY SHARE") {
			return rowFunc(func(...any) error { return pgx.ErrNoRows })
		}
		if strings.Contains(query, "FOR UPDATE OF project") {
			return stringRow("workspace")
		}
		return writeLabelRow(db.label)
	}
	repository := newWriteRepository(t, db)

	_, err := repository.Write(context.Background(), "user", "workspace", writeProject, "", http.MethodPost, strings.NewReader(`{"color":null,"name":null,"parent":"`+writeParent+`","sort_order":"invalid"}`), "application/json")

	assertWriteHTTPError(t, err, 400, map[string]any{
		"color":      []string{"This field may not be null."},
		"name":       []string{"This field may not be null."},
		"parent":     []string{`Invalid pk "` + writeParent + `" - object does not exist.`},
		"sort_order": []string{"A valid number is required."},
	})
	if db.saved {
		t.Fatal("invalid input changed label state")
	}
}

func TestPostNameCollisionUsesGlobalExactThenProjectCasefoldValidation(t *testing.T) {
	for _, scenario := range []struct {
		name            string
		global, project bool
		want            string
	}{
		{"global exact", true, false, "Label with this name already exists."},
		{"project exact", false, true, "LABEL_NAME_ALREADY_EXISTS"},
		{"project casefold", false, true, "LABEL_NAME_ALREADY_EXISTS"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			db := &parentWriteDatabase{}
			db.row = func(query string, args []any) pgx.Row {
				if strings.Contains(query, "FOR UPDATE OF project") {
					return stringRow("workspace")
				}
				if !strings.Contains(query, "SELECT EXISTS") {
					t.Fatalf("unexpected query: %s", query)
				}
				if args[0].(*string) == nil {
					return boolRow(scenario.global)
				}
				return boolRow(scenario.project)
			}
			repository := newWriteRepository(t, db)
			name := "Alpha"
			if scenario.name == "project casefold" {
				name = "alpha"
			}
			_, err := repository.Write(context.Background(), "user", "workspace", writeProject, "", http.MethodPost, strings.NewReader(`{"name":"`+name+`"}`), "application/json")
			assertWriteHTTPError(t, err, 400, map[string]any{"name": []string{scenario.want}})
			if len(db.queries) != 3 && !scenario.global {
				t.Fatalf("queries = %v", db.queries)
			}
			if len(db.queries) != 2 && scenario.global {
				t.Fatalf("queries = %v", db.queries)
			}
		})
	}
}

func TestPostNumericNameAndBooleanSortOrderAreAccepted(t *testing.T) {
	db := &parentWriteDatabase{label: existingWriteLabel()}
	repository := newWriteRepository(t, db)

	label, err := repository.Write(context.Background(), "user", "workspace", writeProject, "", http.MethodPost, strings.NewReader(`{"name":123,"sort_order":true}`), "application/json")

	if err != nil {
		t.Fatal(err)
	}
	if !db.saved || label.Name != "123" {
		t.Fatalf("saved=%v label=%+v", db.saved, label)
	}
	if db.args[len(db.args)-1][1] != "123" {
		t.Fatalf("inserted name = %v", db.args[len(db.args)-1][1])
	}
	// The model's save method overrides client sort_order on create.
	if label.SortOrder != 40010 || db.args[len(db.args)-1][5] != float64(1) || !strings.Contains(db.queries[len(db.queries)-1], "MAX(sort_order)+10000") {
		t.Fatalf("created sort_order=%v SQL=%s args=%v", label.SortOrder, db.queries[len(db.queries)-1], db.args[len(db.args)-1])
	}
}

func TestPostValidSurrogatePairIsAccepted(t *testing.T) {
	db := &parentWriteDatabase{label: existingWriteLabel()}
	repository := newWriteRepository(t, db)

	label, err := repository.Write(context.Background(), "user", "workspace", writeProject, "", http.MethodPost, strings.NewReader(`{"name":"rocket \ud83d\ude80"}`), "application/json")

	if err != nil {
		t.Fatal(err)
	}
	if label.Name != "rocket 🚀" {
		t.Fatalf("name = %q", label.Name)
	}
}

func TestParseLabelInputSupportsFormAndMultipartLastValue(t *testing.T) {
	for _, scenario := range []struct{ name, contentType, body string }{
		{"form", "application/x-www-form-urlencoded", "name=First&name=Form+label"},
		{"multipart", "multipart/form-data; boundary=plane", "--plane\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nFirst\r\n--plane\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nMultipart label\r\n--plane--\r\n"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			input, err := parseLabelInput(strings.NewReader(scenario.body), scenario.contentType)
			if err != nil {
				t.Fatal(err)
			}
			want := "Form label"
			if scenario.name == "multipart" {
				want = "Multipart label"
			}
			if input.fields["name"] != want {
				t.Fatalf("name = %v", input.fields["name"])
			}
		})
	}
}

func TestParseLabelInputRejectsMalformedUnsupportedAndNonObjectJSON(t *testing.T) {
	for _, scenario := range []struct {
		name, contentType, body string
		status                  int
		want                    map[string]any
	}{
		{"malformed", "application/json", `{"name":`, 400, map[string]any{"detail": "JSON parse error - Expecting value: line 1 column 9 (char 8)"}},
		{"trailing data", "application/json", `{"name":"A"}junk`, 400, map[string]any{"detail": "JSON parse error - Extra data: line 1 column 13 (char 12)"}},
		{"missing object delimiter", "application/json", `{"name":"A"`, 400, map[string]any{"detail": "JSON parse error - Expecting ',' delimiter: line 1 column 12 (char 11)"}},
		{"invalid escape", "application/json", `{"name":"\q"}`, 400, map[string]any{"detail": `JSON parse error - Invalid \escape: line 1 column 10 (char 9)`}},
		{"unsupported", "text/plain", "ignored", 415, map[string]any{"detail": `Unsupported media type "text/plain" in request.`}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			_, err := parseLabelInput(strings.NewReader(scenario.body), scenario.contentType)
			assertWriteHTTPError(t, err, scenario.status, scenario.want)
		})
	}
	input, err := parseLabelInput(strings.NewReader(`["value"]`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if input.kind != "list" || !reflect.DeepEqual(input.fields, map[string]any{}) {
		t.Fatalf("input = %+v", input)
	}
}

func TestPostAndPutNullJSONReturnNoDataProvided(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		t.Run(method, func(t *testing.T) {
			db := &parentWriteDatabase{label: existingWriteLabel()}
			repository := newWriteRepository(t, db)
			_, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, method, strings.NewReader(`null`), "application/json")
			assertWriteHTTPError(t, err, 400, map[string]any{"non_field_errors": []string{"No data provided"}})
			if db.saved {
				t.Fatal("null input changed label state")
			}
		})
	}
}

func TestPostMultipartFileValueIsNotAString(t *testing.T) {
	db := &parentWriteDatabase{label: existingWriteLabel()}
	repository := newWriteRepository(t, db)
	body := "--plane\r\nContent-Disposition: form-data; name=\"name\"; filename=\"label.txt\"\r\nContent-Type: text/plain\r\n\r\nFile contents\r\n--plane--\r\n"

	_, err := repository.Write(context.Background(), "user", "workspace", writeProject, "", http.MethodPost, strings.NewReader(body), "multipart/form-data; boundary=plane")

	assertWriteHTTPError(t, err, 400, map[string]any{"name": []string{"Not a valid string."}})
	if db.saved {
		t.Fatal("uploaded file was saved as a name")
	}
}

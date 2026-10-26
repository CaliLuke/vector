package projectlabelreads

import (
	"context"
	"math"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestLabelWriteOptionalFormSortOrder(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch} {
		for _, input := range []struct{ media, body string }{
			{"application/x-www-form-urlencoded", "name=Updated&sort_order="},
			{"multipart/form-data; boundary=plane", "--plane\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nUpdated\r\n--plane\r\nContent-Disposition: form-data; name=\"sort_order\"\r\n\r\n\r\n--plane--\r\n"},
		} {
			t.Run(method+"/"+input.media, func(t *testing.T) {
				db := &parentWriteDatabase{label: existingWriteLabel()}
				repository := newWriteRepository(t, db)
				label, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, method, strings.NewReader(input.body), input.media)
				if err != nil {
					t.Fatal(err)
				}
				if !db.saved {
					t.Fatal("valid form did not save")
				}
				if method == http.MethodPost {
					if got := db.args[len(db.args)-1][5]; got != float64(65535) {
						t.Fatalf("first-label sort order = %v, want default 65535", got)
					}
				} else if label.SortOrder != 10 {
					t.Fatalf("empty form changed order to %v", label.SortOrder)
				}
			})
		}
	}
}

func TestLabelWriteFloatFieldGuards(t *testing.T) {
	for _, test := range []struct {
		name, value, message string
		wantInfinity         bool
	}{
		{"JSON empty stays invalid", `""`, "A valid number is required.", false},
		{"string size limit", `"` + strings.Repeat("0", 1001) + `"`, "String value too large.", false},
		{"size checked before trimming", `"` + strings.Repeat(" ", 1000) + `1"`, "String value too large.", false},
		{"string boundary", `"` + strings.Repeat("0", 1000) + `"`, "", false},
		{"integer overflow", "1" + strings.Repeat("0", 309), "Integer value too large to convert to float", false},
		{"negative integer overflow", "-1" + strings.Repeat("0", 309), "Integer value too large to convert to float", false},
		{"float overflow retains source persistence", "1e309", "", true},
		{"string overflow retains source persistence", `"1e309"`, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := &parentWriteDatabase{label: existingWriteLabel()}
			repository := newWriteRepository(t, db)
			label, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(`{"sort_order":`+test.value+`}`), "application/json")
			if test.message != "" {
				assertWriteHTTPError(t, err, 400, map[string]any{"sort_order": []string{test.message}})
				if db.saved {
					t.Fatal("invalid order was persisted")
				}
				return
			}
			if err != nil || !db.saved {
				t.Fatalf("saved=%v err=%v", db.saved, err)
			}
			if test.wantInfinity != math.IsInf(label.SortOrder, 1) || (!test.wantInfinity && label.SortOrder != 0) {
				t.Fatalf("order = %v, want infinity=%v", label.SortOrder, test.wantInfinity)
			}
		})
	}
}

func TestLabelWriteParentUUIDRepresentations(t *testing.T) {
	compact := strings.ReplaceAll(writeParent, "-", "")
	unicodeDigits := strings.Map(func(r rune) rune { return '\u0660' + r - '0' }, compact)
	for _, encoded := range []string{"urn:uuid:" + compact, "urn:uuid:{" + compact + "}", "{{" + compact[:4] + "--" + compact[4:] + "}}", unicodeDigits} {
		t.Run(encoded, func(t *testing.T) {
			db := &parentWriteDatabase{label: existingWriteLabel()}
			db.row = func(query string, args []any) pgx.Row {
				if strings.Contains(query, "FOR UPDATE OF project") {
					return stringRow("workspace")
				}
				if strings.Contains(query, "FOR KEY SHARE") {
					if args[0] != writeParent {
						t.Fatalf("parent lookup = %v, want %s", args[0], writeParent)
					}
					return stringRow(writeParent)
				}
				if strings.Contains(query, "WITH RECURSIVE ancestry") {
					return boolPairRow(false, false)
				}
				if strings.Contains(query, "UPDATE labels SET") {
					db.saved = true
					db.label.Parent = args[3].(*string)
				}
				return writeLabelRow(db.label)
			}
			repository := newWriteRepository(t, db)
			label, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(`{"parent":"`+encoded+`"}`), "application/json")
			if err != nil || !db.saved || label.Parent == nil || *label.Parent != writeParent {
				t.Fatalf("saved=%v label=%+v err=%v", db.saved, label, err)
			}
		})
	}
}

func TestLabelWriteNumericNameUsesPythonScalarCoercion(t *testing.T) {
	for _, test := range []struct{ value, want string }{{"1e309", "inf"}, {"-1e309", "-inf"}, {"-0", "0"}} {
		t.Run(test.value, func(t *testing.T) {
			db := &parentWriteDatabase{label: existingWriteLabel()}
			repository := newWriteRepository(t, db)
			label, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPost, strings.NewReader(`{"name":`+test.value+`}`), "application/json")
			if err != nil || !db.saved || label.Name != test.want {
				t.Fatalf("saved=%v label=%+v err=%v, want name=%q", db.saved, label, err, test.want)
			}
		})
	}
}

func TestLabelWriteMultipartFileCannotSupplySortOrder(t *testing.T) {
	db := &parentWriteDatabase{label: existingWriteLabel()}
	repository := newWriteRepository(t, db)
	body := "--plane\r\nContent-Disposition: form-data; name=\"sort_order\"; filename=\"123\"\r\nContent-Type: text/plain\r\n\r\n123\r\n--plane--\r\n"
	_, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, http.MethodPatch, strings.NewReader(body), "multipart/form-data; boundary=plane")
	assertWriteHTTPError(t, err, 400, map[string]any{"sort_order": []string{"A valid number is required."}})
	if db.saved {
		t.Fatal("uploaded file changed sort order")
	}
}

func TestLabelWriteReportsFirstSurrogate(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			db := &parentWriteDatabase{label: existingWriteLabel()}
			repository := newWriteRepository(t, db)
			_, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, method, strings.NewReader(`{"name":"bad\ud800\ud801","color":"\udfff\udffe"}`), "application/json")
			assertWriteHTTPError(t, err, 400, map[string]any{
				"name":  []string{"Surrogate characters are not allowed: U+D800."},
				"color": []string{"Surrogate characters are not allowed: U+DFFF."},
			})
			if db.saved {
				t.Fatal("invalid surrogate was persisted")
			}
		})
	}
}

package projectlabelreads

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestLabelWritesTrimPythonControlWhitespace(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch} {
		for _, scenario := range []struct {
			name, color, wantName, wantColor string
			blank                            bool
		}{
			{name: "\u001c\u001d\u001e\u001f", blank: true},
			{name: "\u001cTrimmed\u001f", color: "\u001d#ffffff\u001e", wantName: "Trimmed", wantColor: "#ffffff"},
			{name: "Inner\u001fspace", color: "\u001c\u001f", wantName: "Inner\u001fspace", wantColor: ""},
		} {
			t.Run(method+"/"+scenario.name, func(t *testing.T) {
				db := &writeDatabase{label: existingWriteLabel()}
				repository := newWriteRepository(t, db, &writePublisher{db: db})
				body, err := json.Marshal(map[string]string{"name": scenario.name, "color": scenario.color})
				if err != nil {
					t.Fatal(err)
				}
				label, err := repository.Write(context.Background(), "user", "workspace", writeProject, writeID, method, strings.NewReader(string(body)), "application/json")
				if scenario.blank {
					assertWriteHTTPError(t, err, 400, map[string]any{"name": []string{"This field may not be blank."}})
					if db.saved {
						t.Fatal("blank name was persisted")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if !db.saved || label.Name != scenario.wantName || label.Color != scenario.wantColor {
					t.Fatalf("saved=%v name=%q color=%q, want %q %q", db.saved, label.Name, label.Color, scenario.wantName, scenario.wantColor)
				}
			})
		}
	}
}

package backgroundtasks

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestFormatIssueExportSupportsSourceFormatsAndSanitizesSpreadsheetCells(t *testing.T) {
	records := []map[string]any{{
		"project_name": "Demo", "project_identifier": "DM", "parent": "", "identifier": "DM-3",
		"sequence_id": float64(3), "name": "=1+1", "state_name": "Started", "priority": "high",
		"assignees": []any{"A User", "B User"}, "subscribers": []any{}, "created_by_name": "A User",
		"labels": []any{"Bug", "Urgent"}, "links": []any{map[string]any{"url": "https://example.test", "title": "Example"}},
		"is_draft": false,
	}}
	t.Run("csv", func(t *testing.T) {
		name, data, err := formatIssueExport("demo", "csv", records)
		if err != nil {
			t.Fatal(err)
		}
		if name != "demo.csv" {
			t.Fatalf("filename=%q", name)
		}
		rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 || rows[0][0] != "Project Name" || rows[1][5] != "'=1+1" || rows[1][8] != "[\"A User\",\"B User\"]" {
			t.Fatalf("CSV rows=%#v", rows)
		}
	})
	t.Run("json", func(t *testing.T) {
		name, data, err := formatIssueExport("demo", "json", records)
		if err != nil {
			t.Fatal(err)
		}
		var decoded []map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if name != "demo.json" || decoded[0]["name"] != "=1+1" {
			t.Fatalf("JSON export=%q %s", name, data)
		}
	})
	t.Run("xlsx", func(t *testing.T) {
		name, data, err := formatIssueExport("demo", "xlsx", records)
		if err != nil {
			t.Fatal(err)
		}
		book, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		defer book.Close()
		rows, err := book.GetRows(book.GetSheetName(0))
		if err != nil {
			t.Fatal(err)
		}
		if name != "demo.xlsx" || rows[0][0] != "Project Name" || rows[1][5] != "'=1+1" || rows[1][8] != "A User, B User" {
			t.Fatalf("XLSX rows=%#v", rows)
		}
	})
}

func TestIssueExportZipContainsEveryEncodedFile(t *testing.T) {
	archive, err := zipIssueExport([]struct {
		name string
		data []byte
	}{{name: "demo.csv", data: []byte("a,b\n1,2\n")}, {name: "project.json", data: []byte(`[]`)}})
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("ZIP file count=%d", len(reader.File))
	}
	for index, expected := range []string{"demo.csv", "project.json"} {
		if reader.File[index].Name != expected {
			t.Fatalf("file %d name=%q", index, reader.File[index].Name)
		}
		file, err := reader.File[index].Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(file)
		_ = file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if expected == "demo.csv" && string(got) != "a,b\n1,2\n" || expected == "project.json" && !strings.EqualFold(string(got), "[]") {
			t.Fatalf("file %q content=%q", expected, got)
		}
	}
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	systemapi "github.com/CaliLuke/go-plane"
	publicprojectmetadatareadsapi "github.com/CaliLuke/go-plane/gen/public_project_metadata_reads"
	system "github.com/CaliLuke/go-plane/gen/system"
	publicprojectmetadatareadsstore "github.com/CaliLuke/go-plane/internal/publicprojectmetadatareads"
	"github.com/CaliLuke/loom/clue/log"
)

func TestPublicMetadataQueryNormalizationPreservesOriginalAuditAndRequestLog(t *testing.T) {
	const rawQuery = "search=first&search=last&se%61rch=encoded%2Bvalue&ignored=%ZZ&&bare&empty=&array=one&array=two"
	const path = "/api/public/anchor/example/labels/"
	var logs bytes.Buffer
	activity := &publicProjectMetadataActivityStub{records: make(chan publicprojectmetadatareadsstore.APIActivityLog, 1)}
	handler := newHTTPHandlerWithPendingReplacements(
		newLogContext(context.Background(), &logs, false), system.NewEndpoints(systemapi.NewSystem()), false,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		pendingReplacementEndpoints{
			publicProjectMetadataReads:    publicprojectmetadatareadsapi.NewEndpoints(publicProjectMetadataReadsHTTPStub{}),
			publicProjectMetadataThrottle: &publicProjectMetadataThrottleStub{allowed: true},
			publicProjectMetadataActivity: activity,
			publicProjectMetadataSecret:   "audit-secret",
		},
	)
	request := httptest.NewRequest(http.MethodGet, path+"?"+rawQuery, nil)
	request.Header.Set("X-Api-Key", "audit-key")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	select {
	case record := <-activity.records:
		if record.Path != path || record.QueryParams != rawQuery || record.ResponseCode != http.StatusOK {
			t.Errorf("durable activity record = %#v; want original path and raw query %q", record, rawQuery)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for API activity record")
	}
	// The URL is the audit contract, not incidental log wording. Inspect the
	// structured Django-position request entry rather than unrelated outer logs.
	decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
	found := false
	for {
		var entry map[string]any
		if err := decoder.Decode(&entry); err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if entry["path"] == path && entry["status_code"] != nil {
			found = true
			if entry[log.MessageKey] != "GET "+path+"?"+rawQuery+" 200" {
				t.Errorf("request log URL lost original query: %v", entry[log.MessageKey])
			}
		}
	}
	if !found {
		t.Fatalf("missing Django-position request log: %s", logs.String())
	}
}

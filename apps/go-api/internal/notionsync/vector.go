package notionsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Object map[string]any

type Vector struct {
	api       *apiClient
	prefix    string
	Origin    string
	Workspace string
}

// SetRequestInterval configures pacing for a server with a known rate limit.
func (v *Vector) SetRequestInterval(interval time.Duration) error {
	if interval <= 0 {
		return fmt.Errorf("request interval must be positive")
	}
	v.api.interval = interval
	return nil
}

func NewVector(origin, workspace, key string) (*Vector, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" {
		return nil, fmt.Errorf("Vector origin must be a bare HTTP(S) origin")
	}
	if u.Scheme != "https" && (u.Scheme != "http" || u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") {
		return nil, fmt.Errorf("Vector requires HTTPS except on loopback")
	}
	if workspace == "" || strings.ContainsAny(workspace, "/?#\\") || workspace == "." || workspace == ".." {
		return nil, fmt.Errorf("a valid workspace slug is required")
	}
	if key == "" {
		return nil, fmt.Errorf("VECTOR_API_KEY is required")
	}
	origin = strings.TrimRight(origin, "/")
	return &Vector{Origin: origin, Workspace: workspace, prefix: "/api/v1/workspaces/" + url.PathEscape(workspace) + "/projects/",
		api: newAPI(origin, http.Header{"X-Api-Key": {key}}, 1100*time.Millisecond)}, nil
}

func (v *Vector) collection(kind, project string) string {
	if kind == "project" {
		return v.prefix
	}
	return v.prefix + project + "/" + kind + "/"
}

func (v *Vector) list(ctx context.Context, path string) ([]Object, error) {
	var rows []Object
	cursor := ""
	seen := map[string]bool{}
	for {
		var raw json.RawMessage
		query := "?per_page=100"
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		if err := v.api.request(ctx, http.MethodGet, path+query, nil, &raw); err != nil {
			return nil, err
		}
		if len(raw) > 0 && raw[0] == '[' {
			var values []Object
			err := json.Unmarshal(raw, &values)
			return append(rows, values...), err
		}
		var envelope struct {
			Results []Object `json:"results"`
			HasMore bool     `json:"next_page_results"`
			Next    string   `json:"next_cursor"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, err
		}
		rows = append(rows, envelope.Results...)
		if !envelope.HasMore {
			return rows, nil
		}
		if envelope.Next == "" || seen[envelope.Next] {
			return nil, fmt.Errorf("Vector pagination did not advance")
		}
		seen[envelope.Next] = true
		cursor = envelope.Next
	}
}

func (v *Vector) find(ctx context.Context, kind, project, sourceID string, entry *Entry) (Object, error) {
	path := v.collection(kind, project)
	if entry != nil && entry.ID != "" {
		var object Object
		if err := v.api.request(ctx, http.MethodGet, path+entry.ID+"/", nil, &object); err != nil {
			return nil, err
		}
		if object["external_source"] != "notion" || object["external_id"] != sourceID {
			return nil, fmt.Errorf("destination identity changed for %s", sourceID)
		}
		return object, nil
	}
	rows, err := v.list(ctx, path)
	if err != nil {
		return nil, err
	}
	var found Object
	for _, row := range rows {
		if row["external_source"] == "notion" && row["external_id"] == sourceID {
			if found != nil {
				return nil, fmt.Errorf("multiple destination records claim Notion page %s", sourceID)
			}
			found = row
		}
	}
	return found, nil
}

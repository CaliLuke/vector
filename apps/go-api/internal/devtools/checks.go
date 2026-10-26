package devtools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

const localOrigin = "http://localhost:8000"

// CheckLocalLogin verifies login and the authenticated profile endpoints without printing secrets.
func CheckLocalLogin(ctx context.Context, email, password string, output io.Writer) error {
	return checkLocalLoginAt(ctx, email, password, localOrigin, output)
}

func checkLocalLoginAt(ctx context.Context, email, password, origin string, output io.Writer) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || strings.TrimSpace(password) == "" {
		return errors.New("set PLANE_TEST_EMAIL and PLANE_TEST_PASSWORD before running the local login check")
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("create session cookie jar: %w", err)
	}
	base, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("parse local origin: %w", err)
	}
	client := &http.Client{
		Jar: jar, Timeout: 15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	request := func(path string, body io.Reader) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, origin+path, body)
		if err != nil {
			return nil, err
		}
		if body != nil {
			req.Method = http.MethodPost
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		req.Header.Set("Origin", origin)
		req.Header.Set("Referer", origin+"/")
		return client.Do(req)
	}
	resp, err := request("/auth/get-csrf-token/", nil)
	if err != nil {
		return fmt.Errorf("CSRF endpoint request failed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("CSRF endpoint returned HTTP %d", resp.StatusCode)
	}
	var csrf struct {
		Token string `json:"csrf_token"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&csrf); err != nil {
		resp.Body.Close()
		return fmt.Errorf("decode CSRF response: %w", err)
	}
	resp.Body.Close()
	if csrf.Token == "" {
		return errors.New("CSRF endpoint returned no token")
	}
	form := url.Values{
		"email": {email}, "password": {password}, "csrfmiddlewaretoken": {csrf.Token},
	}
	resp, err = request("/auth/sign-in/", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("sign-in request failed: %w", err)
	}
	location, _ := url.Parse(resp.Header.Get("Location"))
	if resp.StatusCode != http.StatusFound {
		resp.Body.Close()
		return fmt.Errorf("sign-in endpoint returned HTTP %d", resp.StatusCode)
	}
	if location == nil || location.Scheme != base.Scheme || location.Host != base.Host || strings.Contains(location.RawQuery, "error_code=") {
		resp.Body.Close()
		return errors.New("sign-in failed or redirected outside the local proxy origin")
	}
	resp.Body.Close()
	if !hasCookie(jar.Cookies(base), "session-id") {
		return errors.New("sign-in did not issue a session cookie")
	}
	for _, path := range []string{"/api/users/me/", "/api/users/me/profile/", "/api/users/me/settings/", "/api/users/me/workspaces/"} {
		resp, err := request(path, nil)
		if err != nil {
			return fmt.Errorf("%s request failed: %w", path, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("%s returned HTTP %d", path, resp.StatusCode)
		}
		if path == "/api/users/me/" {
			var profile struct {
				Email string `json:"email"`
			}
			if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&profile); err != nil {
				resp.Body.Close()
				return fmt.Errorf("decode authenticated account: %w", err)
			}
			if profile.Email != email {
				resp.Body.Close()
				return errors.New("the authenticated account does not match the login")
			}
		} else {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		}
		resp.Body.Close()
		fmt.Fprintf(output, "PASS %s\n", path)
	}
	fmt.Fprintf(output, "Login and session verified through %s/\n", origin)
	return nil
}

func hasCookie(cookies []*http.Cookie, name string) bool {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return true
		}
	}
	return false
}

// CheckLocalLogging verifies proxy ingestion and correlated Go error logs and spans.
func CheckLocalLogging(ctx context.Context, root string, output io.Writer) error {
	marker := "local-logging-check-" + strings.ReplaceAll(uuid.NewString(), "-", "")
	cutoff := time.Now().UnixNano()
	payload := map[string]any{"resourceLogs": []any{map[string]any{
		"resource": map[string]any{"attributes": []any{map[string]any{
			"key": "service.name", "value": map[string]any{"stringValue": "plane-logging-check"},
		}}},
		"scopeLogs": []any{map[string]any{
			"scope": map[string]any{"name": "plane.local.check"},
			"logRecords": []any{map[string]any{
				"timeUnixNano": fmt.Sprint(cutoff), "severityNumber": 9,
				"severityText": "INFO", "body": map[string]any{"stringValue": marker},
			}},
		}},
	}}}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if postErr := localPost(ctx, localOrigin+"/__local_telemetry/v1/logs", encoded, 5*time.Second); postErr != nil {
		return fmt.Errorf("log collector ingestion request failed: %w", postErr)
	}
	// A malformed renderer charset exercises an existing read-only error path
	// before project lookup. It requires neither credentials nor seeded data.
	traceID := strings.ReplaceAll(uuid.NewString(), "-", "")
	parentSpanID := strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, localOrigin+"/api/public/anchor/observability-probe/labels/", nil)
	if err != nil {
		return err
	}
	apiReq.Header.Set("Accept", "application/json; indent*=bogus''%34")
	apiReq.Header.Set("traceparent", "00-"+traceID+"-"+parentSpanID+"-01")
	apiClient := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	apiResp, err := apiClient.Do(apiReq)
	if err != nil {
		return fmt.Errorf("go error logging probe failed: %w", err)
	}
	if closeErr := apiResp.Body.Close(); closeErr != nil {
		return fmt.Errorf("close logging probe response: %w", closeErr)
	}
	if apiResp.StatusCode != http.StatusInternalServerError {
		return fmt.Errorf("go error logging probe returned HTTP %d, want 500", apiResp.StatusCode)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		query := fmt.Sprintf("SELECT COUNT(*) AS count FROM otel_logs WHERE body_json LIKE '%%%s%%'", marker)
		cmd := exec.CommandContext(ctx, "bash", "scripts/logal-query.sh", "sql", "--query", query, "--json")
		cmd.Dir = root
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("query Logal records: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		var result struct {
			Rows []struct {
				Count int64 `json:"count"`
			} `json:"rows"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			return fmt.Errorf("decode Logal query: %w", err)
		}
		found := len(result.Rows) == 1 && result.Rows[0].Count > 0
		correlated, err := hasLocalGoErrorTrace(ctx, root, traceID)
		if err != nil {
			return err
		}
		if found && correlated {
			_, writeErr := fmt.Fprintln(output, "PASS proxy -> Logal -> SQLite; Go HTTP 500 error log and matching request trace persisted")
			return writeErr
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("log collector verification failed: persisted marker=%t, correlated Go error=%t", found, correlated)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// hasLocalGoErrorTrace requires native OTLP correlation fields, so a trace alone
// or an uncorrelated stdout copy cannot make the logging check pass.
func hasLocalGoErrorTrace(ctx context.Context, root, traceID string) (bool, error) {
	type record struct {
		SpanID     string `json:"span_id"`
		StatusCode int    `json:"status_code"`
	}
	rows := make(map[string][]record, 2)
	for _, signal := range []string{"logs", "spans"} {
		args := []string{"scripts/logal-query.sh", signal, "--service", "plane-go-api", "--trace-id", traceID, "--json"}
		if signal == "logs" {
			args = append(args, "--level", "error")
		}
		command := exec.CommandContext(ctx, "bash", args...)
		command.Dir = root
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		if err := command.Run(); err != nil {
			return false, fmt.Errorf("query Go %s: %w: %s", signal, err, strings.TrimSpace(stderr.String()))
		}
		var result struct {
			Rows []record `json:"rows"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			return false, fmt.Errorf("decode Go %s: %w", signal, err)
		}
		rows[signal] = result.Rows
	}
	for _, entry := range rows["logs"] {
		for _, span := range rows["spans"] {
			if entry.SpanID != "" && entry.SpanID == span.SpanID && span.StatusCode == 2 {
				return true, nil
			}
		}
	}
	return false, nil
}

func localPost(ctx context.Context, endpoint string, body []byte, timeout time.Duration) error {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("returned HTTP %d", resp.StatusCode)
	}
	return nil
}

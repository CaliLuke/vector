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
	"sort"
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

// CheckLocalLogging verifies proxy ingestion and a persisted Go request span.
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
	if err := localPost(ctx, localOrigin+"/__local_telemetry/v1/logs", encoded, 5*time.Second); err != nil {
		return fmt.Errorf("Logal ingestion request failed: %w", err)
	}
	// Exercise the shipping API through the browser proxy. Anonymous Go requests create spans.
	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, localOrigin+"/api/users/me/", nil)
	if err != nil {
		return err
	}
	apiClient := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	apiResp, err := apiClient.Do(apiReq)
	if err != nil {
		return fmt.Errorf("authenticated API probe failed: %w", err)
	}
	apiResp.Body.Close()
	if apiResp.StatusCode != http.StatusUnauthorized && apiResp.StatusCode != http.StatusForbidden && (apiResp.StatusCode < 200 || apiResp.StatusCode >= 300) {
		return fmt.Errorf("anonymous API probe returned HTTP %d", apiResp.StatusCode)
	}
	deadline := time.Now().Add(15 * time.Second)
	for {
		query := fmt.Sprintf("SELECT 'log' AS kind, COUNT(*) AS count, '' AS service_name FROM otel_logs WHERE body_json LIKE '%%%s%%' UNION ALL SELECT 'span' AS kind, COUNT(*) AS count, service_name FROM otel_spans WHERE received_at_unix_nano >= %d GROUP BY service_name", marker, cutoff)
		cmd := exec.CommandContext(ctx, "bash", "scripts/logal-query.sh", "sql", "--query", query, "--json")
		cmd.Dir = root
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("query Logal records: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		var result struct {
			Rows []struct {
				Kind        string `json:"kind"`
				Count       int64  `json:"count"`
				ServiceName string `json:"service_name"`
			} `json:"rows"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			return fmt.Errorf("decode Logal query: %w", err)
		}
		found := false
		services := map[string]bool{}
		for _, row := range result.Rows {
			if row.Kind == "log" && row.Count > 0 {
				found = true
			} else if row.Kind == "span" && row.Count > 0 {
				services[row.ServiceName] = true
			}
		}
		if found && services["plane-go-api"] {
			fmt.Fprintln(output, "PASS proxy -> Logal -> SQLite; Go request trace persisted")
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Logal verification failed: persisted marker=%t, traced services=%v", found, sortedKeys(services))
		}
		time.Sleep(250 * time.Millisecond)
	}
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

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

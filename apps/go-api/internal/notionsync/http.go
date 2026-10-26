package notionsync

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

type apiClient struct {
	base     string
	headers  http.Header
	client   *http.Client
	interval time.Duration
	next     time.Time
}

type responseError struct {
	Status int
	Path   string
}

func (e *responseError) Error() string { return fmt.Sprintf("%s returned HTTP %d", e.Path, e.Status) }

func newAPI(base string, headers http.Header, interval time.Duration) *apiClient {
	if headers == nil {
		headers = make(http.Header)
	}
	return &apiClient{base: base, headers: headers, interval: interval, client: &http.Client{
		Timeout:       45 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

func waitUntil(ctx context.Context, when time.Time) error {
	timer := time.NewTimer(time.Until(when))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *apiClient) request(ctx context.Context, method, path string, body, result any) error {
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 5; attempt++ {
		if err := waitUntil(ctx, c.next); err != nil {
			return err
		}
		request, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(encoded))
		if err != nil {
			return err
		}
		request.Header = c.headers.Clone()
		request.Header.Set("Content-Type", "application/json")
		c.next = time.Now().Add(c.interval)
		response, err := c.client.Do(request)
		if err != nil {
			return fmt.Errorf("%s request failed; retry the sync to reconcile its outcome: %w", path, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, 16<<20))
		_ = response.Body.Close()
		if readErr != nil {
			return readErr
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			if result != nil && len(data) > 0 {
				return json.Unmarshal(data, result)
			}
			return nil
		}
		// Writes with uncertain outcomes are reconciled by the journal, never blindly replayed.
		retry := response.StatusCode == 429 || method == http.MethodGet && (response.StatusCode >= 500 && response.StatusCode <= 504 || response.StatusCode == 529)
		if !retry || attempt == 4 {
			return &responseError{Status: response.StatusCode, Path: path}
		}
		delay := time.Second * time.Duration(1<<attempt)
		if seconds, parseErr := strconv.Atoi(response.Header.Get("Retry-After")); parseErr == nil && seconds > 0 {
			delay = time.Duration(seconds) * time.Second
		}
		c.next = time.Now().Add(delay)
	}
	return fmt.Errorf("request retry limit reached")
}

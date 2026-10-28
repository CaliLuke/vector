package backgroundtasks

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

// FetchPublicFile shares the preview/webhook pinned transport. Each redirect
// gets a fresh validation and pinned socket; no proxy, credentials or allowlist
// can turn an import into access to the application's private network.
func FetchPublicFile(ctx context.Context, raw string, limit int64) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 70*time.Second)
	defer cancel()
	for redirects := 0; redirects <= 5; redirects++ {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.User != nil || parsed.Fragment != "" {
			return nil, "", errors.New("invalid public file URL")
		}
		response, err := pinnedTaskRequest(ctx, http.MethodGet, raw, nil, nil, 70*time.Second, true, nil, nil)
		if err != nil {
			return nil, "", errors.New("public file could not be fetched")
		}
		if response.StatusCode >= 300 && response.StatusCode < 400 {
			location, locationErr := response.Location()
			_ = response.Body.Close()
			if locationErr != nil {
				return nil, "", errors.New("invalid file redirect")
			}
			raw = location.String()
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			_ = response.Body.Close()
			return nil, "", errors.New("file server returned an unsuccessful response")
		}
		if response.ContentLength > limit {
			_ = response.Body.Close()
			return nil, "", errors.New("file exceeds upload limit")
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
		closeErr := response.Body.Close()
		if err != nil || closeErr != nil {
			return nil, "", errors.New("file download was interrupted")
		}
		if int64(len(body)) > limit {
			return nil, "", errors.New("file exceeds upload limit")
		}
		return body, response.Header.Get("Content-Type"), nil
	}
	return nil, "", errors.New("too many file redirects")
}

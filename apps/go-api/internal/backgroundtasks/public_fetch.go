package backgroundtasks

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/CaliLuke/go-plane/internal/workspacewebhooks"
)

// FetchPublicFile shares the preview/webhook pinned transport. Each redirect
// gets a fresh validation and pinned socket; no proxy, credentials or allowlist
// can turn an import into access to the application's private network.
func FetchPublicFile(ctx context.Context, raw string, limit int64) ([]byte, string, error) {
	return fetchPublicFile(ctx, raw, limit, nil, nil)
}

func fetchPublicFile(ctx context.Context, raw string, limit int64, resolver workspacewebhooks.IPResolver, dial func(context.Context, string, string) (net.Conn, error)) ([]byte, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 70*time.Second)
	defer cancel()
	for redirects := 0; redirects <= 5; redirects++ {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.User != nil || parsed.Fragment != "" {
			return nil, "", errors.New("invalid public file URL")
		}
		response, err := pinnedTaskRequest(ctx, http.MethodGet, raw, nil, nil, 70*time.Second, true, resolver, dial)
		if err != nil {
			return nil, "", errors.New("public file could not be fetched")
		}
		if response.StatusCode >= 300 && response.StatusCode < 400 {
			location, locationErr := response.Location()
			closeErr := response.Body.Close()
			if locationErr != nil {
				return nil, "", errors.Join(errors.New("invalid file redirect"), closeErr)
			}
			if closeErr != nil {
				return nil, "", errors.Join(errors.New("file download was interrupted"), closeErr)
			}
			raw = location.String()
			continue
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return nil, "", errors.Join(errors.New("file server returned an unsuccessful response"), response.Body.Close())
		}
		if response.ContentLength > limit {
			return nil, "", errors.Join(errors.New("file exceeds upload limit"), response.Body.Close())
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

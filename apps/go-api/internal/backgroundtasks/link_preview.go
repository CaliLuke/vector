package backgroundtasks

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/CaliLuke/go-plane/internal/workspacewebhooks"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const linkPreviewUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

// The source fallback icon, kept as static product content.
const linkPreviewDefaultIcon = "PHN2ZyB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciIHdpZHRoPSIyNCIgaGVpZ2h0PSIyNCIgdmlld0JveD0iMCAwIDI0IDI0IiBmaWxsPSJub25lIiBzdHJva2U9ImN1cnJlbnRDb2xvciIgc3Ryb2tlLXdpZHRoPSIyIiBzdHJva2UtbGluZWNhcD0icm91bmQiIHN0cm9rZS1saW5lam9pbj0icm91bmQiIGNsYXNzPSJsdWNpZGUgbHVjaWRlLWxpbmstaWNvbiBsdWNpZGUtbGluayI+PHBhdGggZD0iTTEwIDEzYTUgNSAwIDAgMCA3LjU0LjU0bDMtM2E1IDUgMCAwIDAtNy4wNy03LjA3bC0xLjcyIDEuNzEiLz48cGF0aCBkPSJNMTQgMTFhNSA1IDAgMCAwLTcuNTQtLjU0bC0zIDNhNSA1IDAgMCAwIDcuMDcgNy4wN2wxLjcxLTEuNzEiLz48L3N2Zz4="

type LinkPreviewInput struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
type LinkPreviewMetadata struct {
	Title      *string `json:"title"`
	Favicon    string  `json:"favicon"`
	URL        string  `json:"url"`
	FaviconURL *string `json:"favicon_url"`
}
type PostgresLinkPreviewStore struct {
	pool     *pgxpool.Pool
	resolver workspacewebhooks.IPResolver
	dial     func(context.Context, string, string) (net.Conn, error)
}

func NewPostgresLinkPreviewStore(pool *pgxpool.Pool) (*PostgresLinkPreviewStore, error) {
	if pool == nil {
		return nil, errors.New("link preview database required")
	}
	return &PostgresLinkPreviewStore{pool: pool}, nil
}
func (s *PostgresLinkPreviewStore) Crawl(ctx context.Context, in LinkPreviewInput) (struct{}, error) {
	metadata := s.preview(ctx, in.URL)
	raw, err := json.Marshal(metadata)
	if err != nil {
		return struct{}{}, err
	}
	// Do not apply stale metadata if the user edited the link during the crawl.
	_, err = s.pool.Exec(ctx, `UPDATE issue_links SET metadata=$1::jsonb,updated_at=NOW() WHERE id=$2 AND url=$3 AND deleted_at IS NULL`, raw, in.ID, in.URL)
	return struct{}{}, err
}
func (s *PostgresLinkPreviewStore) fetch(ctx context.Context, method, rawURL string, limit int64) ([]byte, http.Header, string, int, error) {
	current := rawURL
	for hop := 0; hop <= 5; hop++ {
		timeout := time.Second
		if method == http.MethodHead {
			timeout = 2 * time.Second
		}
		response, err := pinnedTaskRequest(ctx, method, current, http.Header{"User-Agent": []string{linkPreviewUserAgent}}, nil, timeout, true, s.resolver, s.dial)
		if err != nil {
			return nil, nil, current, 0, err
		}
		redirect := response.StatusCode == 301 || response.StatusCode == 302 || response.StatusCode == 303 || response.StatusCode == 307 || response.StatusCode == 308
		if method != http.MethodHead && redirect && response.Header.Get("Location") != "" {
			response.Body.Close()
			if hop == 5 {
				return nil, nil, current, 0, errors.New("link preview redirect limit")
			}
			base, e := url.Parse(current)
			if e != nil {
				return nil, nil, current, 0, e
			}
			next, e := base.Parse(response.Header.Get("Location"))
			if e != nil {
				return nil, nil, current, 0, e
			}
			current = next.String()
			continue
		}
		data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
		response.Body.Close()
		if int64(len(data)) > limit {
			return nil, nil, current, 0, errors.New("link preview response exceeds byte limit")
		}
		return data, response.Header, current, response.StatusCode, err
	}
	return nil, nil, current, 0, errors.New("link preview redirect limit")
}
func (s *PostgresLinkPreviewStore) preview(ctx context.Context, rawURL string) LinkPreviewMetadata {
	result := LinkPreviewMetadata{URL: rawURL, Favicon: "data:image/svg+xml;base64," + linkPreviewDefaultIcon}
	data, headers, final, _, err := s.fetch(ctx, http.MethodGet, rawURL, 2<<20)
	var root *html.Node
	if err == nil {
		reader, e := charset.NewReader(bytes.NewReader(data), headers.Get("Content-Type"))
		if e == nil {
			root, _ = html.Parse(reader)
		}
	}
	if root != nil {
		var title *html.Node
		for node := range root.Descendants() {
			if node.Type == html.ElementNode && node.Data == "title" {
				title = node
				break
			}
		}
		if title != nil {
			var text strings.Builder
			for n := range title.Descendants() {
				if n.Type == html.TextNode {
					text.WriteString(n.Data)
				}
			}
			value := strings.TrimSpace(text.String())
			result.Title = &value
		}
	}
	icon := ""
	if root != nil {
		for _, rel := range []string{"icon", "shortcut icon", "apple-touch-icon", "apple-touch-icon-precomposed"} {
			for node := range root.Descendants() {
				if node.Type != html.ElementNode || node.Data != "link" {
					continue
				}
				var relation, href string
				for _, attr := range node.Attr {
					switch attr.Key {
					case "rel":
						relation = strings.Join(strings.Fields(attr.Val), " ")
					case "href":
						href = attr.Val
					}
				}
				if relation == rel && href != "" {
					base, e := url.Parse(final)
					if e == nil {
						target, e := base.Parse(href)
						if e == nil {
							icon = target.String()
						}
					}
					break
				}
			}
			if icon != "" {
				break
			}
		}
	}
	if icon == "" {
		base, e := url.Parse(final)
		if e == nil && base.Host != "" {
			base.Path = "/favicon.ico"
			base.RawPath = ""
			base.RawQuery = ""
			base.Fragment = ""
			candidate := base.String()
			if _, _, _, status, e := s.fetch(ctx, http.MethodHead, candidate, 0); e == nil && status == 200 {
				icon = candidate
			}
		}
	}
	if icon != "" {
		content, headers, _, _, e := s.fetch(ctx, http.MethodGet, icon, 1<<20)
		if e == nil {
			kind := headers.Get("Content-Type")
			if kind == "" {
				kind = "image/x-icon"
			}
			result.Favicon = fmt.Sprintf("data:%s;base64,%s", kind, base64.StdEncoding.EncodeToString(content))
			result.FaviconURL = &icon
		}
	}
	return result
}

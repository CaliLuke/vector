package mcpauth

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/CaliLuke/go-plane/internal/mcptransport"
	"github.com/CaliLuke/go-plane/internal/workspacevalidation"
)

func credentialHeaders(header http.Header) (token, slug string, err error) {
	authorization := header.Values("Authorization")
	workspaces := header.Values("X-Workspace-Slug")
	if len(header.Values("X-Api-Key")) > 0 || len(authorization) > 1 || len(workspaces) > 1 {
		return "", "", mcptransport.ErrInvalidRequest
	}
	if len(workspaces) == 1 {
		slug = workspaces[0]
		if len(slug) > 255 || !workspacevalidation.ValidSlug(slug) {
			return "", "", mcptransport.ErrInvalidRequest
		}
	}
	if len(authorization) == 0 {
		return "", "", mcptransport.ErrInvalidCredential
	}
	if strings.Contains(authorization[0], ",") {
		return "", "", mcptransport.ErrInvalidRequest
	}
	scheme, token, found := strings.Cut(authorization[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", "", mcptransport.ErrInvalidCredential
	}
	if strings.ContainsFunc(token, unicode.IsSpace) {
		return "", "", mcptransport.ErrInvalidRequest
	}
	return token, slug, nil
}

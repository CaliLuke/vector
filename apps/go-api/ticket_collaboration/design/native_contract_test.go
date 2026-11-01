package design

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestNativePublicCommentReactionsRequireBrowserSession(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	service := expr.Root.Service("TicketCollaboration")
	if service == nil {
		t.Fatal("collaboration service is missing")
	}
	method := service.Method("GetAPIPublicAnchorByAnchorCommentsByCommentIDReactions")
	if method == nil || len(method.SessionAuths) != 1 || len(method.SessionAuths[0].Transports) != 1 {
		t.Fatalf("public comment reactions must require a browser session: %+v", method)
	}
	transport := method.SessionAuths[0].Transports[0]
	if transport.Kind != expr.SessionCookieTransportKind || transport.HTTPName != "session-id" || transport.Scheme == nil || transport.Scheme.SchemeName != "PlaneSessionCookie" {
		t.Fatalf("browser session transport=%+v", transport)
	}
	for _, scheme := range expr.Root.Schemes {
		if strings.Contains(scheme.SchemeName, "Django") {
			t.Errorf("obsolete framework scheme remains registered: %s", scheme.SchemeName)
		}
	}
}

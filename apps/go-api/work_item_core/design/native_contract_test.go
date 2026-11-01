package design

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestNativeWorkItemSearchRequiresAPIToken(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	service := expr.Root.Service("WorkItemCore")
	if service == nil {
		t.Fatal("work-item service is missing")
	}
	method := service.Method("GetAPIV1WorkspacesBySlugIssuesSearch")
	if method == nil || len(method.Requirements) != 1 || len(method.Requirements[0].Schemes) != 1 {
		t.Fatalf("search must require one API-token scheme: %+v", method)
	}
	scheme := method.Requirements[0].Schemes[0]
	if scheme.Kind != expr.APIKeyKind || scheme.SchemeName != "ApiKeyAuth" || len(method.SessionAuths) != 0 {
		t.Fatalf("search authentication=%+v sessions=%+v", scheme, method.SessionAuths)
	}
	for _, registered := range expr.Root.Schemes {
		if strings.Contains(registered.SchemeName, "Django") {
			t.Errorf("obsolete framework scheme remains registered: %s", registered.SchemeName)
		}
	}
}

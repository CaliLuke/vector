package issuequery

import (
	"strings"
	"testing"
)

func TestModuleNameOrderingRequiresScopeOnlyForModuleLists(t *testing.T) {
	options := Options{OrderBy: "issue_module__module__name", OrderBySet: true}
	args := []any{"actor", "workspace", "project"}
	order, err := Order(options, "", "", "", &args)
	if err != nil || strings.Contains(order, "$") || !strings.Contains(order, "order_relation.issue_id = issue.id") {
		t.Fatalf("unscoped issue ordering=%s err=%v", order, err)
	}
	args = append(args, "module")
	order, err = ModuleOrder(options, "", "", "", &args)
	if err != nil || !strings.Contains(order, "modules.id = $4") {
		t.Fatalf("scoped module ordering=%s err=%v", order, err)
	}
}

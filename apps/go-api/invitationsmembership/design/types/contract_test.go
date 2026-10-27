package types

import (
	"reflect"
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestMemberDirectoryContract(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}

	t.Run("pages retain complete cursor envelope and typed results", func(t *testing.T) {
		for _, page := range []struct {
			typ, entry expr.UserType
		}{
			{WorkspaceMembersPage, WorkspaceMember},
			{RolesPage, RoleDefinition},
		} {
			want := []string{"count", "extra_stats", "grouped_by", "next_cursor", "next_page_results", "prev_cursor", "prev_page_results", "results", "sub_grouped_by", "total_count", "total_pages", "total_results"}
			if !reflect.DeepEqual(page.typ.Attribute().Validation.Required, want) {
				t.Fatalf("%s required fields = %v", page.typ.Name(), page.typ.Attribute().Validation.Required)
			}
			results := expr.AsArray(page.typ.Attribute().Find("results").Type)
			if results == nil || results.ElemType.Type != page.entry {
				t.Fatalf("%s lost its typed results", page.typ.Name())
			}
			for _, field := range []string{"extra_stats", "grouped_by", "sub_grouped_by"} {
				if !page.typ.Attribute().Find(field).Nullable {
					t.Errorf("%s.%s must accept null", page.typ.Name(), field)
				}
			}
		}
	})
	t.Run("member identity preserves required strings and nullable fields", func(t *testing.T) {
		want := []string{"role_slug", "display_name", "email", "first_name", "id", "last_name"}
		if !reflect.DeepEqual(WorkspaceMember.Attribute().Validation.Required, want) {
			t.Fatalf("member required fields = %v", WorkspaceMember.Attribute().Validation.Required)
		}
		for _, field := range []string{"display_name", "email", "first_name", "id", "last_name"} {
			attribute := WorkspaceMember.Attribute().Find(field)
			if attribute.Type != expr.String || attribute.Nullable {
				t.Errorf("member.%s must be a non-null string", field)
			}
		}
		for _, field := range []string{"avatar", "avatar_url", "role_slug"} {
			if !WorkspaceMember.Attribute().Find(field).Nullable {
				t.Errorf("member.%s must accept null", field)
			}
		}
	})
	t.Run("role identities retain namespace and UUID constraints", func(t *testing.T) {
		if !reflect.DeepEqual(RoleDefinition.Attribute().Validation.Required, []string{"id", "name", "slug", "namespace"}) {
			t.Fatalf("role required fields = %v", RoleDefinition.Attribute().Validation.Required)
		}
		if RoleDefinition.Attribute().Find("id").Validation.Format != expr.FormatUUID {
			t.Error("role identity must be a UUID")
		}
		if !reflect.DeepEqual(RoleDefinition.Attribute().Find("namespace").Validation.Values, []any{"workspace", "project"}) {
			t.Error("role namespace must distinguish workspace and project")
		}
	})
}

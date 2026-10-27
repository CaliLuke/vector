package projectstatereads

import (
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
)

func TestBoundStateReadRejectsInvalidAdmissionWithoutQuerying(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*APIV1Request, *string)
	}{
		{"unsupported operation", func(r *APIV1Request, _ *string) { r.Operation = "create" }},
		{"missing actor", func(_ *APIV1Request, actor *string) { *actor = "" }},
		{"invalid actor", func(_ *APIV1Request, actor *string) { *actor = "bad" }},
		{"zero actor", func(_ *APIV1Request, actor *string) { *actor = uuid.Nil().String() }},
		{"zero workspace", func(r *APIV1Request, _ *string) { *r.ExpectedWorkspaceID = uuid.Nil() }},
		{"empty slug", func(r *APIV1Request, _ *string) { r.Params["slug"] = "" }},
		{"invalid project", func(r *APIV1Request, _ *string) { r.Params["project_id"] = "bad" }},
		{"zero project", func(r *APIV1Request, _ *string) { r.Params["project_id"] = uuid.Nil().String() }},
		{"missing detail ID", func(r *APIV1Request, _ *string) { r.Operation = "detail" }},
		{"invalid detail ID", func(r *APIV1Request, _ *string) { r.Operation = "detail"; r.Params["state_id"] = "bad" }},
		{"zero detail ID", func(r *APIV1Request, _ *string) { r.Operation = "detail"; r.Params["state_id"] = uuid.Nil().String() }},
		{"fields", func(r *APIV1Request, _ *string) { r.Params["fields"] = "name" }},
		{"expand", func(r *APIV1Request, _ *string) { r.Params["expand"] = "project" }},
		{"empty fields present", func(r *APIV1Request, _ *string) { r.Present["fields"] = true }},
		{"empty expand present", func(r *APIV1Request, _ *string) { r.Present["expand"] = true }},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace, actor := uuid.New(), uuid.New().String()
			request := APIV1Request{Operation: "list", ExpectedWorkspaceID: &workspace,
				Params: map[string]string{"slug": "states"}, Present: map[string]bool{}}
			test.change(&request, &actor)
			db := &databaseStub{}
			repository, err := NewRepository(db)
			require.NoError(t, err)
			_, err = repository.ReadAPIV1(t.Context(), actor, request)
			require.ErrorIs(t, err, ErrForbidden)
			require.Empty(t, db.queries)
		})
	}
}

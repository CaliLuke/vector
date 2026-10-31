//go:build integration

package mcpactions_test

import (
	"encoding/json"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectmutations"
	"github.com/CaliLuke/go-plane/internal/templates"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func templateRevision(t *testing.T, repository *templates.Repository, query templates.Query) string {
	t.Helper()
	query.IncludeRevision = true
	raw, err := repository.List(t.Context(), query)
	require.NoError(t, err)
	var records []struct{ Revision string }
	require.NoError(t, json.Unmarshal(raw, &records))
	require.NotEmpty(t, records)
	require.NotEmpty(t, records[0].Revision)
	return records[0].Revision
}

func templateRevisionAnchor(t *testing.T, repository *templates.Repository, workspace, slug string) templates.Command {
	t.Helper()
	name := "Revision anchor"
	command := templates.Command{Query: templates.Query{ActorID: uuid.MustParse(testsupport.LabelActor), WorkspaceID: uuid.MustParse(workspace), WorkspaceSlug: slug, Kind: "project"}, Action: "create", Name: &name, Data: map[string]any{}}
	result, err := repository.Mutate(t.Context(), command)
	require.NoError(t, err)
	command.ID, command.Action, command.Data = uuid.MustParse(result.ID), "update", nil
	name = "Accepted after native writer"
	key := uuid.New()
	command.RequestID = &key
	return command
}

func assertTemplateNativeRevisionRejectsStaleWrite(t *testing.T, f milestoneFixture, repository *templates.Repository, command templates.Command) {
	t.Helper()
	before := templateStorageSnapshot(t, f)
	rejected, err := repository.Mutate(t.Context(), command)
	require.ErrorIs(t, err, projectmutations.ErrRevision)
	require.Equal(t, projectmutations.Result{}, rejected)
	require.Equal(t, before, templateStorageSnapshot(t, f))
	current := templateRevision(t, repository, command.Query)
	require.NotEqual(t, *command.ExpectedRevision, current)
	command.ExpectedRevision = &current
	accepted, err := repository.Mutate(t.Context(), command)
	require.NoError(t, err)
	require.NotNil(t, accepted.Replayed)
	require.False(t, *accepted.Replayed, "stale revision refusal must leave the retry key usable")
	saved := templateStorageSnapshot(t, f)
	replay, err := repository.Mutate(t.Context(), command)
	require.NoError(t, err)
	require.NotNil(t, replay.Replayed)
	require.True(t, *replay.Replayed)
	accepted.Replayed = replay.Replayed
	require.Equal(t, accepted, replay)
	require.Equal(t, saved, templateStorageSnapshot(t, f))
}

func TestTemplateRevisionsCoverNativeRowWritersInEveryScope(t *testing.T) {
	for _, scope := range templateScopes() {
		for _, write := range []struct{ name, sql string }{
			{"insert", `INSERT INTO resource_templates(workspace_id,project_id,kind,name) VALUES(@workspace,@project,@kind,'Native addition')`},
			{"update", `UPDATE resource_templates SET summary='Native summary' WHERE id=@subject`},
			{"soft delete", `UPDATE resource_templates SET deleted_at=now() WHERE id=@subject`},
			{"hard delete", `DELETE FROM resource_templates WHERE id=@subject`},
		} {
			t.Run(scope.kind+"/"+scope.project+"/"+write.name, func(t *testing.T) {
				f := templateFixture(t)
				repository := templates.NewRepository(f.pool)
				command := templateRevisionAnchor(t, repository, testsupport.LabelWorkspace, "workspace-one")
				subject := uuid.New()
				var project any
				if scope.project != "" {
					project = uuid.MustParse(scope.project)
				}
				args := pgx.NamedArgs{"workspace": command.WorkspaceID, "project": project, "kind": scope.kind, "subject": subject}
				_, err := f.pool.Exec(t.Context(), `INSERT INTO resource_templates(id,workspace_id,project_id,kind,name) VALUES(@subject,@workspace,@project,@kind,'Native subject')`, args)
				require.NoError(t, err)
				revision := templateRevision(t, repository, command.Query)
				command.ExpectedRevision = &revision
				changed, err := f.pool.Exec(t.Context(), write.sql, args)
				require.NoError(t, err)
				require.EqualValues(t, 1, changed.RowsAffected())
				assertTemplateNativeRevisionRejectsStaleWrite(t, f, repository, command)
			})
		}
	}
}

func TestTemplateMovesAndGovernanceInvalidateWorkspaceRevisions(t *testing.T) {
	f := templateFixture(t)
	otherWorkspace, _ := addPrivateProjectInOtherWorkspace(t, f)
	repository := templates.NewRepository(f.pool)
	first := templateRevisionAnchor(t, repository, testsupport.LabelWorkspace, "workspace-one")
	second := templateRevisionAnchor(t, repository, otherWorkspace, "other")
	subject := uuid.New()
	_, err := f.pool.Exec(t.Context(), `INSERT INTO resource_templates(id,workspace_id,kind,name) VALUES($1,$2,'project','Moving template')`, subject, first.WorkspaceID)
	require.NoError(t, err)
	oldFirst, oldSecond := templateRevision(t, repository, first.Query), templateRevision(t, repository, second.Query)
	first.ExpectedRevision, second.ExpectedRevision = &oldFirst, &oldSecond
	_, err = f.pool.Exec(t.Context(), `UPDATE resource_templates SET workspace_id=$2 WHERE id=$1`, subject, second.WorkspaceID)
	require.NoError(t, err)
	assertTemplateNativeRevisionRejectsStaleWrite(t, f, repository, first)
	assertTemplateNativeRevisionRejectsStaleWrite(t, f, repository, second)
	// Workspace templates remain writable under governance, but the change still
	// invalidates the catalogue token used before governance was enabled.
	revision := templateRevision(t, repository, first.Query)
	first.ExpectedRevision = &revision
	key := uuid.New()
	first.RequestID = &key
	_, err = f.pool.Exec(t.Context(), `UPDATE workspace_template_settings SET owned_by_workspace=true WHERE workspace_id=$1`, first.WorkspaceID)
	require.NoError(t, err)
	assertTemplateNativeRevisionRejectsStaleWrite(t, f, repository, first)
}

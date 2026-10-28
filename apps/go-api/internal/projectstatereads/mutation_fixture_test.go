//go:build integration

package projectstatereads

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	googleuuid "github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type stateMutationFixture struct {
	db         *pgxpool.Pool
	repository *WriteRepository
	input      StateWriteInput
}

func (f stateMutationFixture) command(action, state string) MutationCommand {
	cmd := MutationCommand{ActorID: googleuuid.MustParse(f.input.UserID), WorkspaceID: googleuuid.MustParse(testsupport.LabelWorkspace),
		ProjectID: googleuuid.MustParse(f.input.ProjectID), WorkspaceSlug: f.input.Slug, Action: action}
	if state != "" {
		cmd.StateID = googleuuid.MustParse(state)
	}
	return cmd
}

func (f stateMutationFixture) detail(t *testing.T, state string, revision bool) map[string]any {
	t.Helper()
	workspace := uuid.MustParse(testsupport.LabelWorkspace)
	request := APIV1Request{ExpectedWorkspaceID: &workspace, Operation: "detail", Params: map[string]string{
		"slug": f.input.Slug, "project_id": f.input.ProjectID, "state_id": state,
	}, Present: map[string]bool{}}
	if revision {
		request.Params["include_revision"] = "true"
	}
	value, err := f.repository.ReadAPIV1(t.Context(), f.input.UserID, request)
	require.NoError(t, err)
	return value.(map[string]any)
}

func stateValue[T any](value T) *T { return &value }

func newStateMutationFixture(t *testing.T) stateMutationFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pair.Close()) })
	config, err := pgxpool.ParseConfig(pair.SeedURL())
	require.NoError(t, err)
	config.MaxConns = 4
	config.ConnConfig.RuntimeParams["application_name"] = "state-mutation-tests"
	db, err := pgxpool.NewWithConfig(ctx, config)
	require.NoError(t, err)
	t.Cleanup(db.Close)
	_, err = db.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	repository, err := NewWriteRepository(db)
	require.NoError(t, err)
	return stateMutationFixture{db: db, repository: repository, input: StateWriteInput{
		UserID: testsupport.LabelActor, Slug: "milestones", ProjectID: testsupport.LabelProject,
	}}
}

func (f stateMutationFixture) create(t *testing.T, name string) string {
	t.Helper()
	input := f.input
	input.Fields = map[string]any{"name": name, "color": "#abcdef"}
	raw, err := f.repository.Create(t.Context(), input)
	require.NoError(t, err)
	var record struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(raw, &record))
	require.NotEmpty(t, record.ID)
	return record.ID
}

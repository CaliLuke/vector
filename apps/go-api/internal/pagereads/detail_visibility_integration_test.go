//go:build integration

package pagereads_test

import (
	"context"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/accountauthentication"
	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/stretchr/testify/require"
)

type pageDetailVisits struct{ calls int }

func (visits *pageDetailVisits) AdmitRecentVisit(context.Context, string, string, string, string, string, string) error {
	visits.calls++
	return nil
}

func TestPageDetailVisibilityDoesNotDependOnHiddenName(t *testing.T) {
	fixture := newPageCreateFixture(t)
	root := createUpdateTestPage(t, fixture, map[string]any{"name": "Secret title"})
	child := createUpdateTestPage(t, fixture, map[string]any{"name": "Secret child", "parent": root.String()})
	private := createUpdateTestPage(t, fixture, map[string]any{"name": "Secret private", "access": 1})
	accounts, err := accountauthentication.NewRepository(fixture.pool, nil, false)
	require.NoError(t, err)
	owner, err := accounts.CreateUser(t.Context(), "page-owner@example.test", "page-owner", "!", true, true)
	require.NoError(t, err)
	_, err = fixture.pool.Exec(t.Context(), `UPDATE pages SET owned_by_id=$1 WHERE id=ANY($2::uuid[])`, owner.ID, []uuid.UUID{root, child, private})
	require.NoError(t, err)
	visits := &pageDetailVisits{}
	repository, err := pagereads.NewRepositoryWithNativeTasks(fixture.pool, fixture, nil, visits)
	require.NoError(t, err)

	for _, sample := range []struct {
		name, search      string
		page              uuid.UUID
		archived, revoked bool
		role              int
		want              error
	}{
		{name: "restricted empty search", page: root, role: 5, want: pagereads.ErrRestrictedGuest},
		{name: "restricted matching search", page: root, search: "Secret", role: 5, want: pagereads.ErrRestrictedGuest},
		{name: "restricted nonmatching search", page: root, search: "missing", role: 5, want: pagereads.ErrRestrictedGuest},
		{name: "restricted literal wildcards", page: root, search: `%_`, role: 5, want: pagereads.ErrRestrictedGuest},
		{name: "guest child", page: child, role: 5, want: pagereads.ErrNotFound},
		{name: "member child", page: child, role: 15, want: pagereads.ErrNotFound},
		{name: "guest archived project", page: root, role: 5, archived: true, want: pagereads.ErrNotFound},
		{name: "member archived project", page: root, role: 15, archived: true, want: pagereads.ErrNotFound},
		{name: "private nonowner", page: private, role: 5, want: pagereads.ErrForbidden},
		{name: "private before archived project", page: private, role: 5, archived: true, want: pagereads.ErrForbidden},
		{name: "revoked membership", page: root, role: 5, revoked: true, want: pagereads.ErrForbidden},
		{name: "missing page", page: uuid.New(), role: 5, want: pagereads.ErrForbidden},
	} {
		t.Run(sample.name, func(t *testing.T) {
			_, setupErr := fixture.pool.Exec(t.Context(), `UPDATE projects SET archived_at=CASE WHEN $2 THEN NOW() ELSE NULL END WHERE id=$1`, fixture.project, sample.archived)
			require.NoError(t, setupErr)
			_, setupErr = fixture.pool.Exec(t.Context(), `UPDATE project_members SET role=$3,is_active=$4 WHERE project_id=$1 AND member_id=$2`, fixture.project, fixture.actor, sample.role, !sample.revoked)
			require.NoError(t, setupErr)
			data, readErr := repository.Detail(t.Context(), fixture.actor, "milestones", fixture.project, sample.page, sample.search, "-created_at", "UTC")
			require.ErrorIs(t, readErr, sample.want)
			require.Nil(t, data)
			require.Zero(t, visits.calls)
		})
	}

	// The owner and explicitly enabled guest retain access after the denial fixes.
	_, err = fixture.pool.Exec(t.Context(), `UPDATE pages SET owned_by_id=$1 WHERE id=$2`, fixture.actor, root)
	require.NoError(t, err)
	data, err := repository.Detail(t.Context(), fixture.actor, "milestones", fixture.project, root, "Secret", "-created_at", "UTC")
	require.NoError(t, err)
	require.Contains(t, string(data), "Secret title")
	require.Equal(t, 1, visits.calls)
	_, err = fixture.pool.Exec(t.Context(), `UPDATE pages SET owned_by_id=$1 WHERE id=$2`, owner.ID, root)
	require.NoError(t, err)
	_, err = fixture.pool.Exec(t.Context(), `UPDATE projects SET guest_view_all_features=true WHERE id=$1`, fixture.project)
	require.NoError(t, err)
	data, err = repository.Detail(t.Context(), fixture.actor, "milestones", fixture.project, root, "Secret", "-created_at", "UTC")
	require.NoError(t, err)
	require.Contains(t, string(data), "Secret title")
	require.Equal(t, 2, visits.calls)
}

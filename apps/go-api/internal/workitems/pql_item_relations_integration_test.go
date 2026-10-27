//go:build integration

package workitems

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type pqlIdentityFixture struct {
	ctx     context.Context
	db      *pgx.Conn
	repo    *Repository
	request Request
}

func newPQLIdentityFixture(t *testing.T) pqlIdentityFixture {
	t.Helper()
	ctx, db := postgresfixture.OpenSQL(t, testsupport.LabelReadSchema+testsupport.LabelReadSeed+pqlReadSeed+pqlIdentitySeed)
	repo, err := NewRepository(db, modelPublisher{}, "https://example.test")
	require.NoError(t, err)
	repo.now = func() time.Time { return time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC) }
	q := Request{API: true, Method: "GET", Operation: "get_api_v1_workspaces_by_slug_projects_by_project_id_issues", ActorID: testsupport.LabelActor, Slug: "demo", ProjectID: testsupport.LabelProject, Timezone: "UTC"}
	require.NoError(t, repo.AuthorizeInitial(ctx, q))
	return pqlIdentityFixture{ctx, db, repo, q}
}

func (f pqlIdentityFixture) examples(t *testing.T, cases map[string][]int) {
	t.Helper()
	for query, want := range cases {
		t.Run(query, func(t *testing.T) { assertPQLSequences(t, f.ctx, f.repo, f.request, query, want) })
	}
}

func TestAPIPQLItemRelationsRespectDirectionAndVisibility(t *testing.T) {
	f := newPQLIdentityFixture(t)
	f.examples(t, map[string][]int{
		`isTopLevel()`: {3, 4}, `isSubWorkItem()`: {1, 2}, `hasChildren()`: {1, 2}, `NOT hasChildren()`: {3, 4},
		`childOf("FOR-5")`: {1}, `childOf("FOR-0005")`: {1}, `childOf("90000000-0000-4000-8000-000000000005")`: {1},
		`childOf("CUR-6")`: {2}, `childOf("CUR-7")`: {}, `childOf("OTHER-24")`: {},
		`parentOf("FOR-21")`: {1}, `parentOf("FOR-22")`: {2}, `parentOf("FOR-23")`: {}, `parentOf("OTHER-24")`: {},
		`childOf("MISSING-1")`: {}, `NOT childOf("MISSING-1")`: {1, 2, 3, 4}, `childOf("for-5")`: {},
		`childOf("FOR-9999999999999999999999999999999999999999999999999")`: {},
		`linkedTo("FOR-5")`: {1, 2}, `NOT linkedTo("FOR-5")`: {3, 4},
		`blockedBy("FOR-5")`: {1}, `blocks("FOR-5")`: {2}, `duplicateOf("FOR-5")`: {3},
		`NOT blockedBy("FOR-5")`: {2, 3, 4}, `NOT blocks("FOR-5")`: {1, 3, 4}, `NOT duplicateOf("FOR-5")`: {1, 2, 4},
		`linkedTo("CUR-8")`: {}, `NOT linkedTo("CUR-8")`: {1, 2, 3, 4},
		`blockedBy("CUR-7")`: {}, `blockedBy("OTHER-24")`: {},
		`isSubWorkItem() AND hasChildren() AND linkedTo("FOR-5") AND NOT duplicateOf("FOR-5") AND NOT isDraft()`: {1, 2},
	})
	t.Run("duplicate links do not inflate pages", func(t *testing.T) {
		assertPQLFirstTwoPages(t, f.ctx, f.repo, f.request, `linkedTo("FOR-5")`, 2, "1", "2")
	})
}

func TestAPIPQLItemRelationsRecheckGuestAndRevokedTargetMembership(t *testing.T) {
	f := newPQLIdentityFixture(t)
	_, err := f.db.Exec(f.ctx, `UPDATE project_members SET role=5 WHERE project_id='00000000-0000-4000-8000-000000000099' AND member_id='00000000-0000-4000-8000-000000000003'; UPDATE issues SET created_by_id=NULL WHERE sequence_id=5`)
	require.NoError(t, err)
	f.examples(t, map[string][]int{`childOf("FOR-5")`: {}, `NOT childOf("FOR-5")`: {1, 2, 3, 4}, `isTopLevel()`: {1, 3, 4}, `hasChildren()`: {}, `linkedTo("FOR-5")`: {}})
	_, err = f.db.Exec(f.ctx, `UPDATE projects SET guest_view_all_features=true,archived_at=now() WHERE id='00000000-0000-4000-8000-000000000099'`)
	require.NoError(t, err)
	f.examples(t, map[string][]int{`childOf("FOR-5")`: {1}, `linkedTo("FOR-5")`: {1, 2}, `hasChildren()`: {1, 2}})
	_, err = f.db.Exec(f.ctx, `UPDATE project_members SET deleted_at=now() WHERE project_id='00000000-0000-4000-8000-000000000099' AND member_id='00000000-0000-4000-8000-000000000003'`)
	require.NoError(t, err)
	f.examples(t, map[string][]int{`childOf("FOR-5")`: {}, `isTopLevel()`: {1, 3, 4}, `NOT linkedTo("FOR-5")`: {1, 2, 3, 4}, `hasChildren()`: {}})
}

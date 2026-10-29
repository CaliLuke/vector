//go:build integration

package pagereads_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/bootstrap"
	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

var pageCreateSchema = postgresfixture.NewTemplate(bootstrap.MigrateSchema)

func TestMain(m *testing.M) {
	status := m.Run()
	if err := pageCreateSchema.Close(); err != nil {
		fmt.Fprintln(os.Stderr, "close page creation test schema:", err)
		status = 1
	}
	os.Exit(status)
}

type pageCreateFixture struct {
	pool       *pgxpool.Pool
	repository *pagereads.Repository
	actor      uuid.UUID
	project    uuid.UUID
	tasks      []map[string]any
}

func newPageCreateFixture(t *testing.T) *pageCreateFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	database, err := pageCreateSchema.CloneDatabase(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 30*time.Second)
		defer stop()
		require.NoError(t, database.Close(cleanup))
	})
	pool, err := pgxpool.New(ctx, database.URL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, testsupport.MilestoneWriteSeed)
	require.NoError(t, err)
	actor, err := uuid.Parse(testsupport.LabelActor)
	require.NoError(t, err)
	project, err := uuid.Parse(testsupport.LabelProject)
	require.NoError(t, err)
	fixture := &pageCreateFixture{pool: pool, actor: actor, project: project}
	fixture.repository, err = pagereads.NewRepository(pool, fixture)
	require.NoError(t, err)
	return fixture
}

func (fixture *pageCreateFixture) Task(ctx context.Context, name string, _ []any, fields map[string]any) error {
	if name != "plane.bgtasks.page_transaction_task.page_transaction" {
		return fmt.Errorf("unexpected task %q", name)
	}
	var linked bool
	if err := fixture.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pages p JOIN project_pages pp ON pp.page_id=p.id WHERE p.id=$1)`, fields["page_id"]).Scan(&linked); err != nil {
		return err
	}
	if !linked {
		return errors.New("page task was published before the page and project link committed")
	}
	fixture.tasks = append(fixture.tasks, fields)
	return nil
}

func (*pageCreateFixture) SoftDelete(context.Context, string, string, string) error {
	return errors.New("unexpected soft-delete task")
}

func (fixture *pageCreateFixture) create(t *testing.T, fields map[string]any) (int, json.RawMessage, error) {
	t.Helper()
	return fixture.repository.Lifecycle(t.Context(), "create", fixture.actor, "milestones", fixture.project, uuid.UUID{}, "UTC", djangoinput.Input{Fields: fields})
}

func (fixture *pageCreateFixture) counts(t *testing.T) []int {
	t.Helper()
	counts := make([]int, 3)
	require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM pages), (SELECT count(*) FROM project_pages), (SELECT count(*) FROM page_labels)`).Scan(&counts[0], &counts[1], &counts[2]))
	return counts
}

func TestPageCreateStoresRawDescriptionAndPublishesAfterCommit(t *testing.T) {
	fixture := newPageCreateFixture(t)
	for _, sample := range []struct {
		name   string
		fields map[string]any
		html   string
		json   string
		binary []byte
	}{
		{name: "defaults", fields: map[string]any{}, html: "<p></p>", json: "{}"},
		{name: "raw whitespace and bytes", fields: map[string]any{"description_html": "  <p>Body</p>  ", "description_json": map[string]any{"type": "doc"}, "description_binary": []byte{0, 1, 255}}, html: "  <p>Body</p>  ", json: `{"type":"doc"}`, binary: []byte{0, 1, 255}},
		{name: "nullable binary", fields: map[string]any{"description_binary": nil}, html: "<p></p>", json: "{}"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			status, raw, err := fixture.create(t, sample.fields)
			require.NoError(t, err)
			require.Equal(t, 201, status)
			var response map[string]any
			require.NoError(t, json.Unmarshal(raw, &response))
			require.Equal(t, sample.html, response["description_html"])
			task := fixture.tasks[len(fixture.tasks)-1]
			require.Equal(t, response["id"], task["page_id"])
			require.Equal(t, sample.html, task["new_description_html"])
			require.Nil(t, task["old_description_html"])
			var html string
			var encoded, binary []byte
			require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT description_html,description_json,description_binary FROM pages WHERE id=$1`, response["id"]).Scan(&html, &encoded, &binary))
			require.Equal(t, sample.html, html)
			require.JSONEq(t, sample.json, string(encoded))
			require.Equal(t, sample.binary, binary)
		})
	}
	require.Equal(t, []int{3, 3, 0}, fixture.counts(t))
	require.Len(t, fixture.tasks, 3)
}

func TestPageCreateRejectsInvalidInputWithoutDurableEffects(t *testing.T) {
	fixture := newPageCreateFixture(t)
	for _, sample := range []struct {
		name   string
		fields map[string]any
		status int
		body   any
	}{
		{"invalid parent precedes fields", map[string]any{"parent": "bad", "access": nil}, 400, map[string]any{"error": "Please provide valid detail"}},
		{"missing parent precedes fields", map[string]any{"parent": uuid.New().String(), "access": nil}, 404, map[string]any{"error": "The required object does not exist."}},
		{"serializer errors precede raw description", map[string]any{"access": nil, "description_html": false}, 400, map[string]any{"access": []string{"This field may not be null."}}},
		{"null HTML cannot be persisted", map[string]any{"description_html": nil}, 400, map[string]any{"error": "The payload is not valid"}},
		{"null JSON cannot be persisted", map[string]any{"description_json": nil}, 400, map[string]any{"error": "The payload is not valid"}},
		{"nontext raw description", map[string]any{"description_html": false}, 500, map[string]any{"error": "Something went wrong please try again later"}},
		{"raw binary cannot be a request string", map[string]any{"description_binary": "AAE="}, 500, map[string]any{"error": "Something went wrong please try again later"}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			_, _, err := fixture.create(t, sample.fields)
			var lifecycle *pagereads.LifecycleError
			var httpError *djangoapi.HTTPError
			switch {
			case errors.As(err, &lifecycle):
				require.Equal(t, sample.status, lifecycle.Status)
				require.Equal(t, sample.body, lifecycle.Body)
			case errors.As(err, &httpError):
				require.Equal(t, sample.status, httpError.Status)
				require.Equal(t, sample.body, httpError.Body)
			default:
				t.Fatalf("unexpected creation error: %v", err)
			}
			require.Equal(t, []int{0, 0, 0}, fixture.counts(t))
			require.Empty(t, fixture.tasks)
		})
	}
}

func TestPageCreateChecksMembershipBeforeParentOrFields(t *testing.T) {
	fixture := newPageCreateFixture(t)
	_, err := fixture.pool.Exec(t.Context(), `UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, fixture.project, fixture.actor)
	require.NoError(t, err)
	_, _, err = fixture.create(t, map[string]any{"parent": "bad", "access": nil})
	var denied *djangoapi.HTTPError
	require.ErrorAs(t, err, &denied)
	require.Equal(t, 403, denied.Status)
	require.Equal(t, []int{0, 0, 0}, fixture.counts(t))
	require.Empty(t, fixture.tasks)
}

func TestPageCreatePersistsMetadataAndScopedRelations(t *testing.T) {
	fixture := newPageCreateFixture(t)
	_, raw, err := fixture.create(t, map[string]any{"name": "Parent"})
	require.NoError(t, err)
	var parent map[string]any
	require.NoError(t, json.Unmarshal(raw, &parent))
	label := uuid.New().String()
	_, err = fixture.pool.Exec(t.Context(), `INSERT INTO labels(id,workspace_id,project_id,name,description,color,sort_order,created_at,updated_at) VALUES($1,$2,$3,'Label','','#123456',0,now(),now())`, label, testsupport.LabelWorkspace, fixture.project)
	require.NoError(t, err)
	_, raw, err = fixture.create(t, map[string]any{
		"name": "  Child  ", "color": "#654321", "access": json.Number("1"), "parent": parent["id"], "labels": []any{label},
		"view_props": map[string]any{"full_width": true}, "logo_props": map[string]any{"emoji": "map"}, "is_locked": true, "archived_at": "2026-10-29",
	})
	require.NoError(t, err)
	var child map[string]any
	require.NoError(t, json.Unmarshal(raw, &child))
	require.Equal(t, "Child", child["name"])
	require.Equal(t, "#654321", child["color"])
	require.Equal(t, float64(1), child["access"])
	require.Equal(t, parent["id"], child["parent"])
	require.Equal(t, true, child["is_locked"])
	require.Equal(t, map[string]any{"full_width": true}, child["view_props"])
	require.Equal(t, map[string]any{"emoji": "map"}, child["logo_props"])
	require.Equal(t, []any{label}, child["label_ids"])
	var archived string
	require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT to_char(archived_at,'YYYY-MM-DD') FROM pages WHERE id=$1`, child["id"]).Scan(&archived))
	require.Equal(t, "2026-10-29", archived)
	require.Equal(t, []int{2, 2, 1}, fixture.counts(t))
	require.Len(t, fixture.tasks, 2)
}

func TestPageCreateRollsBackWhenProjectLinkCannotBeWritten(t *testing.T) {
	fixture := newPageCreateFixture(t)
	_, err := fixture.pool.Exec(t.Context(), `ALTER TABLE project_pages ADD CONSTRAINT reject_new_page_links CHECK (false) NOT VALID`)
	require.NoError(t, err)
	_, _, err = fixture.create(t, map[string]any{"name": "Rejected"})
	var failure *djangoapi.HTTPError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, 400, failure.Status)
	require.Equal(t, []int{0, 0, 0}, fixture.counts(t))
	require.Empty(t, fixture.tasks)
}

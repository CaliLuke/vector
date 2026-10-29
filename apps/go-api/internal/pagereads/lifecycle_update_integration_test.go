//go:build integration

package pagereads_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/CaliLuke/go-plane/internal/pagereads"
	"github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/stretchr/testify/require"
)

func createUpdateTestPage(t *testing.T, fixture *pageCreateFixture, fields map[string]any) uuid.UUID {
	t.Helper()
	_, raw, err := fixture.create(t, fields)
	require.NoError(t, err)
	var page struct {
		ID uuid.UUID `json:"id"`
	}
	require.NoError(t, json.Unmarshal(raw, &page))
	return page.ID
}

func createUpdateTestLabel(t *testing.T, fixture *pageCreateFixture) string {
	t.Helper()
	label := uuid.New().String()
	_, err := fixture.pool.Exec(t.Context(), `INSERT INTO labels(id,workspace_id,project_id,name,description,color,sort_order,created_at,updated_at) VALUES($1,$2,$3,$4,'','#123456',0,now(),now())`, label, testsupport.LabelWorkspace, fixture.project, label)
	require.NoError(t, err)
	return label
}

func TestPageUpdateValidationOrderLeavesPageAndTasksUnchanged(t *testing.T) {
	for _, sample := range []struct {
		name                             string
		fields                           map[string]any
		locked, selfParent, noMembership bool
		status                           int
		body                             any
	}{
		{name: "locked before parent", fields: map[string]any{"parent": "bad", "access": nil}, locked: true, status: 400, body: map[string]any{"error": "This page is locked"}},
		{name: "membership before parent", fields: map[string]any{"parent": "bad"}, noMembership: true, status: 403, body: map[string]any{"detail": "You do not have permission to perform this action."}},
		{name: "parent syntax before fields", fields: map[string]any{"parent": "bad", "access": nil}, status: 400, body: map[string]any{"error": "Please provide valid detail"}},
		{name: "missing parent before fields", fields: map[string]any{"parent": uuid.New().String(), "access": nil}, status: 400, body: map[string]any{"error": "Access cannot be updated since this page is owned by someone else"}},
		{name: "cycle before fields", fields: map[string]any{"access": nil}, selfParent: true, status: 400, body: map[string]any{"parent": []string{"Page hierarchy cannot contain a cycle."}}},
		{name: "serializer aggregates failures", fields: map[string]any{"description_html": "", "access": nil}, status: 400, body: map[string]any{"description_html": []string{"This field may not be blank."}, "access": []string{"This field may not be null."}}},
		{name: "fields before collaboration revision", fields: map[string]any{"access": nil, "collaboration_revision": 1}, status: 400, body: map[string]any{"access": []string{"This field may not be null."}}},
		{name: "collaboration revision before write", fields: map[string]any{"name": "After", "collaboration_revision": 1}, status: 409, body: map[string]any{"error": "The page was replaced. Reload it before editing."}},
	} {
		t.Run(sample.name, func(t *testing.T) {
			fixture := newPageCreateFixture(t)
			page := createUpdateTestPage(t, fixture, map[string]any{"name": "Before", "description_html": "<p>Before</p>", "is_locked": sample.locked})
			if sample.selfParent {
				sample.fields["parent"] = page.String()
			}
			if sample.noMembership {
				_, err := fixture.pool.Exec(t.Context(), `UPDATE workspace_members SET is_active=false WHERE member_id=$1`, fixture.actor)
				require.NoError(t, err)
			}

			_, _, err := fixture.repository.Lifecycle(t.Context(), "update", fixture.actor, "milestones", fixture.project, page, "UTC", djangoinput.Input{Fields: sample.fields})

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
				t.Fatalf("unexpected update error: %v", err)
			}
			var name, html string
			require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT name,description_html FROM pages WHERE id=$1`, page).Scan(&name, &html))
			require.Equal(t, "Before", name)
			require.Equal(t, "<p>Before</p>", html)
			require.Equal(t, []int{1, 1, 0}, fixture.counts(t))
			require.Len(t, fixture.tasks, 1)
		})
	}
}

type pageUpdateCommitObserver struct{ *pageCreateFixture }

func (observer *pageUpdateCommitObserver) Task(ctx context.Context, name string, args []any, fields map[string]any) error {
	var html string
	if err := observer.pool.QueryRow(ctx, `SELECT description_html FROM pages WHERE id=$1`, fields["page_id"]).Scan(&html); err != nil {
		return err
	}
	if html != "<p>After</p>" {
		return fmt.Errorf("page update task observed uncommitted HTML %q", html)
	}
	return observer.pageCreateFixture.Task(ctx, name, args, fields)
}

func TestPageUpdatePersistsMetadataRelationsAndRawTaskAfterCommit(t *testing.T) {
	fixture := newPageCreateFixture(t)
	parent := createUpdateTestPage(t, fixture, map[string]any{"name": "Parent"})
	page := createUpdateTestPage(t, fixture, map[string]any{"name": "Before", "description_html": "<p>Before</p>"})
	label := createUpdateTestLabel(t, fixture)
	repository, err := pagereads.NewRepository(fixture.pool, &pageUpdateCommitObserver{fixture})
	require.NoError(t, err)
	fields := map[string]any{"name": "  After  ", "description_html": "  <p>After</p>  ", "color": "#abcdef", "access": json.Number("1"), "parent": parent.String(), "labels": []any{label}, "view_props": map[string]any{"full_width": true}, "logo_props": map[string]any{"emoji": "map"}, "is_locked": true, "archived_at": "2026-10-29"}

	status, raw, err := repository.Lifecycle(t.Context(), "update", fixture.actor, "milestones", fixture.project, page, "UTC", djangoinput.Input{Fields: fields})

	require.NoError(t, err)
	require.Equal(t, 200, status)
	var response map[string]any
	require.NoError(t, json.Unmarshal(raw, &response))
	require.Equal(t, "After", response["name"])
	require.Equal(t, "<p>After</p>", response["description_html"])
	require.Equal(t, "#abcdef", response["color"])
	require.Equal(t, float64(1), response["access"])
	require.Equal(t, parent.String(), response["parent"])
	require.Equal(t, map[string]any{"full_width": true}, response["view_props"])
	require.Equal(t, map[string]any{"emoji": "map"}, response["logo_props"])
	require.Equal(t, true, response["is_locked"])
	require.NotContains(t, response, "label_ids")
	var stripped, storedLabel, archived string
	require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT p.description_stripped,pl.label_id::text,to_char(p.archived_at,'YYYY-MM-DD') FROM pages p JOIN page_labels pl ON pl.page_id=p.id AND pl.deleted_at IS NULL WHERE p.id=$1`, page).Scan(&stripped, &storedLabel, &archived))
	require.Equal(t, "After", stripped)
	require.Equal(t, label, storedLabel)
	require.Equal(t, "2026-10-29", archived)
	require.Len(t, fixture.tasks, 3)
	require.Equal(t, "  <p>After</p>  ", fixture.tasks[2]["new_description_html"])
	require.Equal(t, "<p>Before</p>", fixture.tasks[2]["old_description_html"])
}

func TestPageUpdateLabelFailureRollsBackContentAndExistingRelations(t *testing.T) {
	fixture := newPageCreateFixture(t)
	oldLabel := createUpdateTestLabel(t, fixture)
	newLabel := createUpdateTestLabel(t, fixture)
	page := createUpdateTestPage(t, fixture, map[string]any{"name": "Before", "description_html": "<p>Before</p>", "labels": []any{oldLabel}})
	_, err := fixture.pool.Exec(t.Context(), fmt.Sprintf(`ALTER TABLE page_labels ADD CONSTRAINT reject_new_label CHECK (label_id <> '%s'::uuid)`, newLabel))
	require.NoError(t, err)

	_, _, err = fixture.repository.Lifecycle(t.Context(), "update", fixture.actor, "milestones", fixture.project, page, "UTC", djangoinput.Input{Fields: map[string]any{"name": "After", "description_html": "<p>After</p>", "labels": []any{newLabel}}})

	require.Error(t, err)
	var name, html, label string
	require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT p.name,p.description_html,pl.label_id::text FROM pages p JOIN page_labels pl ON pl.page_id=p.id AND pl.deleted_at IS NULL WHERE p.id=$1`, page).Scan(&name, &html, &label))
	require.Equal(t, "Before", name)
	require.Equal(t, "<p>Before</p>", html)
	require.Equal(t, oldLabel, label)
	require.Equal(t, []int{1, 1, 1}, fixture.counts(t))
	require.Len(t, fixture.tasks, 1)
}

func TestPageUpdateClearsParentAndLabelsWithoutChangingDescription(t *testing.T) {
	fixture := newPageCreateFixture(t)
	parent := createUpdateTestPage(t, fixture, map[string]any{"name": "Parent"})
	label := createUpdateTestLabel(t, fixture)
	page := createUpdateTestPage(t, fixture, map[string]any{"name": "Before", "description_html": "<p>Before</p>", "parent": parent.String(), "labels": []any{label}})

	status, _, err := fixture.repository.Lifecycle(t.Context(), "update", fixture.actor, "milestones", fixture.project, page, "UTC", djangoinput.Input{Fields: map[string]any{"parent": nil, "labels": []any{}}})

	require.NoError(t, err)
	require.Equal(t, 200, status)
	var parentID *uuid.UUID
	var html, stripped string
	var activeLabels int
	require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT parent_id,description_html,description_stripped,(SELECT count(*) FROM page_labels WHERE page_id=p.id AND deleted_at IS NULL) FROM pages p WHERE p.id=$1`, page).Scan(&parentID, &html, &stripped, &activeLabels))
	require.Nil(t, parentID)
	require.Equal(t, "<p>Before</p>", html)
	require.Equal(t, "Before", stripped)
	require.Zero(t, activeLabels)
	require.Len(t, fixture.tasks, 2)
}

func TestPageUpdateNonOwnerAccessCheckUsesRawScalarBeforeValidation(t *testing.T) {
	fixture := newPageCreateFixture(t)
	page := createUpdateTestPage(t, fixture, map[string]any{"name": "Before"})
	owner := uuid.New()
	_, err := fixture.pool.Exec(t.Context(), `INSERT INTO users SELECT (jsonb_populate_record(NULL::users,to_jsonb(source)||jsonb_build_object('id',$1::uuid,'email','owner@example.test','username','page-owner'))).* FROM users source WHERE id=$2`, owner, fixture.actor)
	require.NoError(t, err)
	_, err = fixture.pool.Exec(t.Context(), `UPDATE pages SET owned_by_id=$1 WHERE id=$2`, owner, page)
	require.NoError(t, err)

	_, _, err = fixture.repository.Lifecycle(t.Context(), "update", fixture.actor, "milestones", fixture.project, page, "UTC", djangoinput.Input{Fields: map[string]any{"access": "0", "description_html": nil}})

	var rejected *djangoapi.HTTPError
	require.ErrorAs(t, err, &rejected)
	require.Equal(t, 400, rejected.Status)
	require.Equal(t, map[string]any{"error": "Access cannot be updated since this page is owned by someone else"}, rejected.Body)
	var storedName string
	require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT name FROM pages WHERE id=$1`, page).Scan(&storedName))
	require.Equal(t, "Before", storedName)

	status, _, err := fixture.repository.Lifecycle(t.Context(), "update", fixture.actor, "milestones", fixture.project, page, "UTC", djangoinput.Input{Fields: map[string]any{"access": json.Number("0"), "name": "After"}})

	require.NoError(t, err)
	require.Equal(t, 200, status)
	require.NoError(t, fixture.pool.QueryRow(t.Context(), `SELECT name FROM pages WHERE id=$1`, page).Scan(&storedName))
	require.Equal(t, "After", storedName)
	require.Len(t, fixture.tasks, 1)
}

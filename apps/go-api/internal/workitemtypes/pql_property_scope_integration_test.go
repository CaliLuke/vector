//go:build integration

package workitemtypes_test

import (
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/workitems"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func requirePQLPropertyError(t *testing.T, f projectTypeFixture, repo *workitems.Repository, actor, query string, status int) map[string]any {
	t.Helper()
	_, err := repo.Execute(f.ctx, workitems.Request{Operation: pqlPropertyOperation, Method: "GET", ActorID: actor, Slug: "types", ProjectID: f.project, API: true, Timezone: "UTC", Params: map[string]string{"pql": query}}, nil, "")
	var failure *djangoapi.HTTPError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, status, failure.Status)
	if status == 400 {
		require.Equal(t, "invalid_pql", failure.Body["code"])
	}
	return failure.Body
}

func TestAPIPQLPropertySelectionsUseAuthorizedCatalogsAndRetainInactiveIdentities(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	option, member, release, first, second, ownRelease, privateRelease, privateProject := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	issue := uuid.NewString()
	attachValueProperties(t, f,
		property.Definition{ID: issue, Title: "Linked work", Kind: property.Issue, Configuration: property.Configuration{Active: true, Multiple: true}},
		property.Definition{ID: option, Title: "Choice", Kind: property.Dropdown, Configuration: property.Configuration{Active: true, Multiple: true, Options: []property.Option{{ID: first, Name: "First"}, {ID: second, Name: "Second"}}}},
		property.Definition{ID: member, Title: "Reviewers", Kind: property.Member, Configuration: property.Configuration{Active: true, Multiple: true}},
		property.Definition{ID: release, Title: "Deliveries", Kind: property.Release, Configuration: property.Configuration{Active: true, Multiple: true}},
	)
	f.addProject(privateProject)
	f.exec(`INSERT INTO work_item_releases(id,workspace_id,project_id,name) VALUES($1,$2,$3,'Visible'),($4,$2,$5,'Private')`, ownRelease, f.workspace, f.project, privateRelease, privateProject)
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[0], map[string]any{option: []string{first, second}, member: []string{f.member}, release: []string{ownRelease}, issue: []string{ids[2]}})
	f.exec(`UPDATE issues SET custom_properties=$2::jsonb WHERE id=$1`, ids[1], map[string]any{option: []string{second}})
	assertPQLPropertyResults(t, f, repo, map[string][]string{
		pqlCF(issue) + ` = "` + ids[2] + `"`: {ids[0]}, pqlCF(issue) + ` IS NULL`: {ids[1]},
		pqlCF(issue) + ` IN ("` + ids[2] + `")`: {ids[0]}, pqlCF(issue) + ` NOT IN ("` + ids[1] + `")`: {ids[0]},
		pqlCF(option) + ` = "` + first + `"`: {ids[0]}, pqlCF(option) + ` != "` + first + `"`: {ids[1]},
		pqlCF(option) + ` IN ("` + first + `","` + second + `")`: {ids[0], ids[1]},
		pqlCF(option) + ` NOT IN ("` + first + `")`:              {ids[1]}, pqlCF(option) + ` IS NULL`: {},
		pqlCF(member) + ` = "` + f.member + `"`: {ids[0]}, pqlCF(member) + ` IS NULL`: {ids[1]},
		pqlCF(member) + ` != "` + f.member + `"`: {}, `NOT ` + pqlCF(member) + ` IN ("` + f.member + `")`: {},
		pqlCF(release) + ` = "` + ownRelease + `"`: {ids[0]}, pqlCF(release) + ` IS NOT NULL`: {ids[0]},
	})
	requirePQLPropertyError(t, f, repo, f.owner, pqlCF(issue)+` = "`+uuid.NewString()+`"`, 400)
	for _, query := range []string{pqlCF(option) + ` = "` + uuid.NewString() + `"`, pqlCF(member) + ` = "` + f.outsider + `"`, pqlCF(release) + ` = "` + privateRelease + `"`, pqlCF(uuid.NewString()) + ` IS NULL`} {
		requirePQLPropertyError(t, f, repo, f.owner, query, 400)
	}
	foreignWorkspace, foreignProperty := uuid.NewString(), uuid.NewString()
	f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Foreign',$2,'foreign','UTC','',now(),now())`, foreignWorkspace, f.owner)
	f.exec(`INSERT INTO work_item_properties(id,workspace_id,definition) SELECT $1::uuid,$2::uuid,jsonb_set(definition,'{id}',to_jsonb($1::uuid::text)) FROM work_item_properties WHERE id=$3`, foreignProperty, foreignWorkspace, option)
	f.exec(`INSERT INTO work_item_type_properties(type_id,property_id) VALUES($1,$2)`, f.task, foreignProperty)
	requirePQLPropertyError(t, f, repo, f.owner, pqlCF(foreignProperty)+` IS NULL`, 400)
	f.exec(`UPDATE workspace_members SET is_active=false WHERE workspace_id=$1 AND member_id=$2`, f.workspace, f.member)
	assertPQLPropertyResults(t, f, repo, map[string][]string{pqlCF(member) + ` = "` + f.member + `"`: {ids[0]}})
	// Operand admission still runs when no item could be returned.
	f.exec(`UPDATE issues SET deleted_at=now()`)
	requirePQLPropertyError(t, f, repo, f.owner, pqlCF(release)+` = "`+privateRelease+`"`, 400)
}

func TestAPIPQLPropertyContextAndGuestScopeRemainAuthoritative(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	number := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}, Contexts: []property.Context{{ID: uuid.NewString(), Name: "Inactive for bugs", Types: []string{f.bug}, Overrides: []string{"active"}, Configuration: property.Configuration{Active: false}}}})
	f.exec(`INSERT INTO work_item_type_properties(type_id,property_id) VALUES($1,$2)`, f.bug, number)
	f.exec(`UPDATE issues SET custom_properties=jsonb_build_object($1::text,4)`, number)
	assertPQLPropertyResults(t, f, repo, map[string][]string{
		pqlCF(number) + ` = 4`: {ids[0], ids[1]}, pqlCF(number) + ` IS NULL`: {},
		`NOT ` + pqlCF(number) + ` = 4`: {}, pqlCF(number) + ` = 4 OR title = "Discovery 2"`: ids,
	})
	f.exec(`UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	f.exec(`UPDATE issues SET created_by_id=$2 WHERE id=$1`, ids[0], f.member)
	page := readPropertyPage(t, f, repo, pqlPropertyOperation, f.member, map[string]string{"pql": pqlCF(number) + ` = 4 OR title ~ "Discovery"`, "per_page": "10"}, true)
	require.Equal(t, 1, page.Total)
	require.Equal(t, ids[0], page.Results[0].ID)
	f.exec(`UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2`, f.project, f.member)
	requirePQLPropertyError(t, f, repo, f.member, pqlCF(number)+` IS NULL`, 403)
	requirePQLPropertyError(t, f, repo, f.outsider, pqlCF(uuid.NewString())+` IS NULL`, 403)
	f.exec(`UPDATE work_item_properties SET definition=jsonb_set(definition,'{configuration,active}','false') WHERE id=$1`, number)
	assertPQLPropertyResults(t, f, repo, map[string][]string{pqlCF(number) + ` IS NULL`: {}, `NOT ` + pqlCF(number) + ` = 4`: {}})
}

func TestAPIPQLPropertyDisabledAndInvalidTypeFailuresAreExplicit(t *testing.T) {
	f := newProjectTypeFixture(t)
	repo, ids := seedDiscoveryItems(t, f)
	number := uuid.NewString()
	attachValueProperties(t, f, property.Definition{ID: number, Title: "Effort", Kind: property.Number, Configuration: property.Configuration{Active: true}})
	query := pqlCF(number) + ` IS NULL OR title = "Discovery 2"`
	// Recreate retained untyped data from before type activation. Current writes
	// reject it while the feature is enabled.
	setRetainedPQLType := func(typeID any) {
		f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
		f.exec(`UPDATE issues SET type_id=$2 WHERE id=$1`, ids[2], typeID)
		f.exec(`UPDATE work_item_type_settings SET enabled=true WHERE workspace_id=$1`, f.workspace)
	}
	setRetainedPQLType(nil)
	assertPQLPropertyResults(t, f, repo, map[string][]string{query: ids, `NOT ` + pqlCF(number) + ` IS NULL`: {}})
	f.exec(`UPDATE issues SET type_id=$2 WHERE id=$1`, ids[2], f.bug)
	f.exec(`UPDATE issue_types SET deleted_at=now() WHERE id=$1`, f.bug)
	requirePQLPropertyError(t, f, repo, f.owner, query, 400)
	requirePQLPropertyError(t, f, repo, f.owner, `NOT `+pqlCF(number)+` IS NULL`, 400)
	f.exec(`UPDATE issue_types SET deleted_at=NULL WHERE id=$1`, f.bug)
	foreignWorkspace, foreignType := uuid.NewString(), uuid.NewString()
	f.exec(`INSERT INTO workspaces(id,name,owner_id,slug,timezone,background_color,created_at,updated_at) VALUES($1,'Foreign',$2,'foreign','UTC','',now(),now())`, foreignWorkspace, f.owner)
	f.exec(`INSERT INTO issue_types(id,workspace_id,name,description,logo_props,is_active,is_default,mandatory,level,is_epic,created_at,updated_at) VALUES($1,$2,'Foreign type','','{}',true,false,false,0,false,now(),now())`, foreignType, foreignWorkspace)
	// Seed a corrupt retained row in this disposable database only. Restore the
	// hierarchy guard before the read so production enforcement remains intact.
	f.exec(`ALTER TABLE issues DISABLE TRIGGER work_item_hierarchy_guard`)
	setRetainedPQLType(foreignType)
	f.exec(`ALTER TABLE issues ENABLE TRIGGER work_item_hierarchy_guard`)
	requirePQLPropertyError(t, f, repo, f.owner, query, 400)
	setRetainedPQLType(f.bug)
	f.exec(`UPDATE work_item_type_settings SET enabled=false WHERE workspace_id=$1`, f.workspace)
	requirePQLPropertyError(t, f, repo, f.owner, query, 400)
	page := readPropertyPage(t, f, repo, pqlPropertyOperation, f.owner, map[string]string{"pql": `title ~ "Discovery"`, "per_page": "10"}, true)
	require.Equal(t, 3, page.Total, "ordinary PQL must not opt in to property discovery")
}

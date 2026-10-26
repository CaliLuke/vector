package pagereads

import (
	"context"
	"errors"
	"strings"
	"testing"

	"uuid"

	"github.com/jackc/pgx/v5"
)

type rowFunc func(...any) error

func (row rowFunc) Scan(destinations ...any) error { return row(destinations...) }

type databaseStub struct {
	queries []string
	args    [][]any
	rows    []pgx.Row
}

func (database *databaseStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	database.queries = append(database.queries, query)
	database.args = append(database.args, args)
	if len(database.rows) == 0 {
		return rowFunc(func(...any) error { return errors.New("unexpected query") })
	}
	row := database.rows[0]
	database.rows = database.rows[1:]
	return row
}

func boolRow(value bool) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*bool) = value
		return nil
	})
}

func stringRow(value string) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*string) = value
		return nil
	})
}

func bytesRow(value []byte) pgx.Row {
	return rowFunc(func(destinations ...any) error {
		*destinations[0].(*[]byte) = value
		return nil
	})
}

func TestNewRepositoryRequiresDatabase(t *testing.T) {
	if _, err := NewRepository(nil); err == nil {
		t.Fatal("NewRepository(nil) returned nil error")
	}
}

func TestProjectAuthorizationRequiresActiveMembershipAndAllowedPageRole(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), bytesRow([]byte(`{"public_pages":1,"private_pages":2,"archived_pages":3}`))}}
	repository, _ := NewRepository(database)
	_, err := repository.Summary(context.Background(), uuid.New(), "demo", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"project_member.member_id = $1", "workspace.slug = $2", "project.id = $3",
		"project_member.role IN (5, 15, 20)", "project_member.is_active = true",
		"project_member.deleted_at IS NULL", "project.deleted_at IS NULL", "workspace.deleted_at IS NULL",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("project authorization query missing %q", fragment)
		}
	}
}

func TestProjectAuthorizationMapsMissingMembershipToForbidden(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(false)}}
	repository, _ := NewRepository(database)
	_, err := repository.List(context.Background(), uuid.New(), "demo", uuid.New(), "", "-created_at", "UTC")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
	if len(database.queries) != 1 {
		t.Fatalf("queries=%d, want authorization only", len(database.queries))
	}
}

func TestPageAuthorizationPreservesOwnershipPublicPrivateAndActiveLinkRules(t *testing.T) {
	tests := []struct {
		name     string
		access   string
		restrict bool
		want     error
	}{
		{name: "owner", access: "allowed", restrict: true},
		{name: "public member", access: "allowed"},
		{name: "private non-owner", access: "forbidden", want: ErrForbidden},
		{name: "missing active link", access: "forbidden", want: ErrForbidden},
		{name: "restricted guest detail", access: "restricted_guest", restrict: true, want: ErrRestrictedGuest},
		{name: "no visible project for non-restricted caller", access: "not_found", restrict: true, want: ErrNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &databaseStub{rows: []pgx.Row{stringRow(test.access)}}
			repository, _ := NewRepository(database)
			err := repository.authorizePage(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), test.restrict)
			if !errors.Is(err, test.want) {
				t.Fatalf("err=%v want=%v", err, test.want)
			}
			query := database.queries[0]
			for _, fragment := range []string{
				"project_page.project_id = $3", "project_page.deleted_at IS NULL", "page.id = $4",
				"page.deleted_at IS NULL", "owned_by_id = $1",
				"WHEN access = 1 AND owned_by_id <> $1 THEN 'forbidden'",
				"visible_project.archived_at IS NULL",
				"page.parent_id IS NULL AS is_top_level",
				"WHEN $5 AND (NOT has_visible_project OR NOT is_top_level) AND role = 5 AND NOT guest_view_all_features THEN 'django_page_nil'",
				"WHEN $5 AND (NOT has_visible_project OR NOT is_top_level) THEN 'not_found'",
				"WHEN $5 AND role = 5 AND NOT guest_view_all_features THEN 'restricted_guest'",
			} {
				if !strings.Contains(query, fragment) {
					t.Errorf("page authorization query missing %q", fragment)
				}
			}
			privateIndex := strings.Index(query, "WHEN access = 1 AND owned_by_id <> $1 THEN 'forbidden'")
			noVisibleIndex := strings.Index(query, "WHEN $5 AND (NOT has_visible_project OR NOT is_top_level)")
			if privateIndex < 0 || noVisibleIndex < 0 || privateIndex > noVisibleIndex {
				t.Errorf("private non-owner denial must precede no-visible-project handling")
			}
			if strings.Contains(query, "\n    AND project.archived_at IS NULL") {
				t.Errorf("ProjectPagePermission must not reject the archived URL project: %s", query)
			}
			if got := database.args[0][4]; got != test.restrict {
				t.Errorf("restrictGuest=%v, want %v", got, test.restrict)
			}
		})
	}
}
func TestOptionsAuthorizationUsesProjectPagePermissionScope(t *testing.T) {
	userID := uuid.New()
	projectID := uuid.New()
	pageID := uuid.New()

	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, _ := NewRepository(database)
	if err := repository.AuthorizeOptions(context.Background(), userID, "demo", projectID, nil); err != nil {
		t.Fatal(err)
	}
	if len(database.args[0]) != 3 {
		t.Fatalf("summary OPTIONS args=%#v", database.args[0])
	}

	database = &databaseStub{rows: []pgx.Row{stringRow("allowed")}}
	repository, _ = NewRepository(database)
	if err := repository.AuthorizeOptions(context.Background(), userID, "demo", projectID, &pageID); err != nil {
		t.Fatal(err)
	}
	if len(database.args[0]) != 5 || database.args[0][3] != pageID || database.args[0][4] != false {
		t.Fatalf("version OPTIONS args=%#v", database.args[0])
	}
}

func TestSummaryCountsVisiblePublicPrivateAndArchivedTopLevelPages(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), bytesRow([]byte(`{"public_pages":4,"private_pages":2,"archived_pages":1}`))}}
	repository, _ := NewRepository(database)
	data, err := repository.Summary(context.Background(), uuid.New(), "demo", uuid.New())
	if err != nil || string(data) != `{"public_pages":4,"private_pages":2,"archived_pages":1}` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"page.parent_id IS NULL", "page.owned_by_id = $1 OR page.access = 0",
		"page.access = 0 AND page.archived_at IS NULL", "page.access = 1 AND page.archived_at IS NULL",
		"page.archived_at IS NOT NULL", "project_member.role <> 5 OR project.guest_view_all_features",
		"visible_project.archived_at IS NULL", "visible_link.deleted_at IS NULL",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("summary query missing %q", fragment)
		}
	}
}

func TestListSearchesNamesButAlwaysOrdersFavoritesBeforeCreation(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), bytesRow([]byte(`[]`))}}
	repository, _ := NewRepository(database)
	data, err := repository.List(context.Background(), uuid.New(), "demo", uuid.New(), `road, "map plan"`, "name", "America/New_York")
	if err != nil || string(data) != `[]` {
		t.Fatalf("data=%s err=%v", data, err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"favorite.entity_type = 'page'", "favorite.deleted_at IS NULL",
		"page_label.deleted_at IS NULL", "linked_page.deleted_at IS NULL",
		"FROM unnest($4::text[]) AS search_term",
		"UPPER(page.name) NOT LIKE UPPER('%' || search_term || '%') ESCAPE E'\\\\'",
		"ORDER BY is_favorite DESC, created_at DESC",
		"project_member.role <> 5 OR project.guest_view_all_features OR page.owned_by_id = $1",
		"AT TIME ZONE $5",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("list query missing %q", fragment)
		}
	}
	got, ok := database.args[1][3].([]string)
	if !ok || len(got) != 2 || got[0] != "road" || got[1] != "map plan" {
		t.Fatalf("search terms=%#v", database.args[1][3])
	}
	if database.args[1][4] != "America/New_York" {
		t.Fatalf("timezone=%#v", database.args[1][4])
	}
	if strings.Contains(query, "ORDER BY name") {
		t.Fatalf("client order unexpectedly changed final ordering: %s", query)
	}
}

func TestListRejectsInvalidAndExplicitlyEmptyDjangoOrderFieldBeforeReadingPages(t *testing.T) {
	for _, orderBy := range []string{"not_a_page_field", ""} {
		database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
		repository, _ := NewRepository(database)
		_, err := repository.List(context.Background(), uuid.New(), "demo", uuid.New(), "", orderBy, "UTC")
		if !errors.Is(err, ErrInvalidOrder) {
			t.Fatalf("orderBy=%q err=%v", orderBy, err)
		}
		if len(database.queries) != 1 {
			t.Fatalf("orderBy=%q queries=%d, want authorization only", orderBy, len(database.queries))
		}
	}
}

func TestDetailRejectsInvalidAndExplicitlyEmptyDjangoOrderFieldBeforeReadingPage(t *testing.T) {
	for _, orderBy := range []string{"not_a_page_field", ""} {
		database := &databaseStub{rows: []pgx.Row{stringRow("allowed")}}
		repository, _ := NewRepository(database)
		_, err := repository.Detail(
			context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "", orderBy, "UTC",
		)
		if !errors.Is(err, ErrInvalidOrder) {
			t.Fatalf("orderBy=%q err=%v", orderBy, err)
		}
		if len(database.queries) != 1 {
			t.Fatalf("orderBy=%q queries=%d, want authorization only", orderBy, len(database.queries))
		}
	}
}

func TestListAndDetailRejectNullSearchAsDRFValidationError(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true)}}
	repository, _ := NewRepository(database)
	_, err := repository.List(
		context.Background(), uuid.New(), "demo", uuid.New(), "before\x00after", "-created_at", "UTC",
	)
	if !errors.Is(err, ErrInvalidSearch) || len(database.queries) != 1 {
		t.Fatalf("list err=%v queries=%d", err, len(database.queries))
	}

	database = &databaseStub{rows: []pgx.Row{stringRow("allowed")}}
	repository, _ = NewRepository(database)
	_, err = repository.Detail(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "before\x00after", "-created_at", "UTC",
	)
	if !errors.Is(err, ErrInvalidSearch) || len(database.queries) != 1 {
		t.Fatalf("detail err=%v queries=%d", err, len(database.queries))
	}
}
func TestPageOrderValidationMatchesDjangoLazyFirstSegmentResolution(t *testing.T) {
	valid := map[string]pageOrder{
		"pk":                           {field: "id"},
		"-pk":                          {field: "id", descending: true},
		"assets":                       {field: "assets"},
		"page_versions":                {field: "page_versions"},
		"owned_by__email":              {field: "owned_by__email"},
		"workspace__slug":              {field: "workspace__slug"},
		"parent__parent__name":         {field: "parent__parent__name"},
		"assets__size":                 {field: "assets__size"},
		"page_versions__last_saved_at": {field: "page_versions__last_saved_at"},
		"view_props__full_width":       {field: "view_props__full_width"},
		"logo_props__emoji__skin_tone": {field: "logo_props__emoji__skin_tone"},
		"created_at__year":             {field: "created_at__year"},
		"created_at__date__month":      {field: "created_at__date__month"},
		"archived_at__year":            {field: "archived_at__year"},
		"owned_by__created_at__date":   {field: "owned_by__created_at__date"},
		"project_pages__project__logo_props__emoji": {field: "project_pages__project__logo_props__emoji"},
		"owned_by__does_not_exist":                  {field: "owned_by__does_not_exist"},
		"name__email":                               {field: "name__email"},
		"parent__":                                  {field: "parent__"},
		"created_at__bogus":                         {field: "created_at__bogus"},
		"created_at__year__month":                   {field: "created_at__year__month"},
		"archived_at__hour":                         {field: "archived_at__hour"},
		"name__year":                                {field: "name__year"},
	}
	for value, want := range valid {
		got, ok := parsePageOrder(value)
		if !ok || got != want {
			t.Errorf("parsePageOrder(%q)=(%+v,%v), want (%+v,true)", value, got, ok, want)
		}
	}
	for _, value := range []string{"", "-?", "does_not_exist", "does_not_exist__name", "--pk", "__name"} {
		if _, ok := parsePageOrder(value); ok {
			t.Errorf("parsePageOrder(%q) unexpectedly valid", value)
		}
	}
}

func TestListEscapesDjangoLikeWildcards(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), bytesRow([]byte(`[]`))}}
	repository, _ := NewRepository(database)
	_, err := repository.List(
		context.Background(), uuid.New(), "demo", uuid.New(), `100%_done\later`, "-created_at", "UTC",
	)
	if err != nil {
		t.Fatal(err)
	}
	got := database.args[1][3].([]string)
	if len(got) != 1 || got[0] != `100\%\_done\\later` {
		t.Fatalf("escaped search terms=%q", got)
	}
}

func TestPageSearchTermsMatchDRFSmartSplit(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{name: "unmatched quote is literal", value: `"unterminated`, want: []string{`"unterminated`}},
		{name: "embedded quote stays literal", value: `pre"quoted phrase"post`, want: []string{`pre"quoted phrase"post`}},
		{name: "fully quoted phrase is unescaped", value: `"quoted \"phrase\""`, want: []string{`quoted "phrase"`}},
		{name: "commas split non-quoted tokens", value: "one,two,, three", want: []string{"one", "two", "three"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := pageSearchTerms(test.value)
			if len(got) != len(test.want) {
				t.Fatalf("terms=%q want=%q", got, test.want)
			}
			for index := range got {
				if got[index] != test.want[index] {
					t.Fatalf("terms=%q want=%q", got, test.want)
				}
			}
		})
	}
}

func TestListTreatsUnmatchedQuoteAsNormalSearchTerm(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{boolRow(true), bytesRow([]byte(`[]`))}}
	repository, _ := NewRepository(database)
	_, err := repository.List(context.Background(), uuid.New(), "demo", uuid.New(), `"unterminated`, "-created_at", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	got := database.args[1][3].([]string)
	if len(got) != 1 || got[0] != `"unterminated` {
		t.Fatalf("search terms=%q", got)
	}
}

func TestDetailReturnsStoredHTMLIssueIDsAndMapsMissingRow(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{stringRow("allowed"), bytesRow([]byte(`{"description_html":"<p>stored</p>","issue_ids":[]}`))}}
	repository, _ := NewRepository(database)
	data, err := repository.Detail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "stored page", "-created_at", "Asia/Kolkata")
	if err != nil || !strings.Contains(string(data), "<p>stored</p>") {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for _, fragment := range []string{
		"'description_html', page.description_html", "page_log.entity_name = 'issue'",
		"page_log.deleted_at IS NULL", "project_page.deleted_at IS NULL",
		"page.parent_id IS NULL",
		"project_member.is_active = true", "page.owned_by_id = $1 OR page.access = 0",
		"project_member.role <> 5 OR project.guest_view_all_features OR page.owned_by_id = $1",
		"FROM unnest($5::text[]) AS search_term",
		"UPPER(page.name) NOT LIKE UPPER('%' || search_term || '%') ESCAPE E'\\\\'",
		"AT TIME ZONE $6",
	} {
		if !strings.Contains(database.queries[1], fragment) {
			t.Errorf("detail query missing %q", fragment)
		}
	}
	searchTerms, ok := database.args[1][4].([]string)
	if !ok || len(searchTerms) != 2 || searchTerms[0] != "stored" || searchTerms[1] != "page" {
		t.Fatalf("detail search terms=%#v", database.args[1][4])
	}
	if database.args[1][5] != "Asia/Kolkata" {
		t.Fatalf("timezone=%#v", database.args[1][5])
	}

	database = &databaseStub{rows: []pgx.Row{stringRow("allowed"), rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ = NewRepository(database)
	_, err = repository.Detail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "nonmatching", "-created_at", "UTC")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
func TestRestrictedGuestNonmatchingSearchPreservesDjangoNilDereference(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{stringRow("restricted_guest"), boolRow(false)}}
	repository, _ := NewRepository(database)
	_, err := repository.Detail(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), `missing%_page`, "-created_at", "UTC",
	)
	if !errors.Is(err, ErrDjangoPageNil) {
		t.Fatalf("err=%v", err)
	}
	if len(database.queries) != 2 || !strings.Contains(database.queries[1], "UPPER(page.name) NOT LIKE") {
		t.Fatalf("queries=%q", database.queries)
	}
	patterns := database.args[1][1].([]string)
	if len(patterns) != 1 || patterns[0] != `missing\%\_page` {
		t.Fatalf("patterns=%q", patterns)
	}

	database = &databaseStub{rows: []pgx.Row{stringRow("restricted_guest"), boolRow(true)}}
	repository, _ = NewRepository(database)
	_, err = repository.Detail(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "matching", "-created_at", "UTC",
	)
	if !errors.Is(err, ErrRestrictedGuest) {
		t.Fatalf("matching search err=%v", err)
	}
}

func TestDetailPreservesDjangoNilDereferenceOnlyForRestrictedGuestWithoutVisibleProject(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{stringRow("django_page_nil")}}
	repository, _ := NewRepository(database)
	_, err := repository.Detail(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "", "-created_at", "UTC",
	)
	if !errors.Is(err, ErrDjangoPageNil) {
		t.Fatalf("err=%v", err)
	}
	if len(database.queries) != 1 {
		t.Fatalf("queries=%d, want authorization only", len(database.queries))
	}
	for _, fragment := range []string{
		"visible_project.archived_at IS NULL",
		"page.parent_id IS NULL AS is_top_level",
		"WHEN $5 AND (NOT has_visible_project OR NOT is_top_level) AND role = 5 AND NOT guest_view_all_features THEN 'django_page_nil'",
		"WHEN $5 AND (NOT has_visible_project OR NOT is_top_level) THEN 'not_found'",
	} {
		if !strings.Contains(database.queries[0], fragment) {
			t.Errorf("authorization query missing %q", fragment)
		}
	}

	database = &databaseStub{rows: []pgx.Row{stringRow("not_found")}}
	repository, _ = NewRepository(database)
	_, err = repository.Detail(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "", "-created_at", "UTC",
	)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-restricted caller err=%v", err)
	}

	database = &databaseStub{rows: []pgx.Row{stringRow("not_found")}}
	repository, _ = NewRepository(database)
	_, err = repository.Detail(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "", "not_a_page_field", "UTC",
	)
	if !errors.Is(err, ErrInvalidOrder) {
		t.Fatalf("no-visible invalid order err=%v", err)
	}

	database = &databaseStub{rows: []pgx.Row{stringRow("not_found")}}
	repository, _ = NewRepository(database)
	_, err = repository.Detail(
		context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "before\x00after", "-created_at", "UTC",
	)
	if !errors.Is(err, ErrInvalidSearch) {
		t.Fatalf("no-visible NUL search err=%v", err)
	}
	if len(database.queries) != 1 {
		t.Fatalf("non-restricted caller queries=%d, want authorization only", len(database.queries))
	}
}

func TestDetailChildPageUsesCallerSpecificDjangoOutcome(t *testing.T) {
	tests := []struct {
		name   string
		access string
		want   error
	}{
		{name: "ordinary member gets not found", access: "not_found", want: ErrNotFound},
		{name: "restricted guest preserves nil dereference", access: "django_page_nil", want: ErrDjangoPageNil},
		{name: "private non-owner is forbidden first", access: "forbidden", want: ErrForbidden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database := &databaseStub{rows: []pgx.Row{stringRow(test.access)}}
			repository, _ := NewRepository(database)
			_, err := repository.Detail(
				context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "", "-created_at", "UTC",
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("err=%v want=%v", err, test.want)
			}
			if len(database.queries) != 1 {
				t.Fatalf("queries=%d, want classification only", len(database.queries))
			}
		})
	}
}
func TestDescriptionReturnsRawBinaryAndCoalescesNullToEmpty(t *testing.T) {
	binary := []byte{0x00, 0x01, 0xff}
	database := &databaseStub{rows: []pgx.Row{stringRow("allowed"), bytesRow(binary)}}
	repository, _ := NewRepository(database)
	data, err := repository.Description(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New())
	if err != nil || string(data) != string(binary) {
		t.Fatalf("data=%v err=%v", data, err)
	}
	query := database.queries[1]
	for _, fragment := range []string{
		"COALESCE(page.description_binary, ''::bytea)", "project_page.deleted_at IS NULL",
		"project_member.is_active = true", "page.owned_by_id=$1 OR page.access=0",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("description query missing %q", fragment)
		}
	}
}

func TestVersionListOrdersNewestFirstAndVersionDetailPreservesStoredFields(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{stringRow("allowed"), bytesRow([]byte(`[]`))}}
	repository, _ := NewRepository(database)
	_, err := repository.VersionList(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), "Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{
		"ORDER BY version.created_at DESC", "project_member.is_active = true",
		"JOIN workspaces workspace", "workspace.id = version.workspace_id",
		"page.owned_by_id = $1 OR page.access = 0", "AT TIME ZONE $5",
	} {
		if !strings.Contains(database.queries[1], fragment) {
			t.Errorf("version list query missing %q", fragment)
		}
	}
	if database.args[1][4] != "Europe/Paris" {
		t.Fatalf("version list timezone=%#v", database.args[1][4])
	}

	database = &databaseStub{rows: []pgx.Row{stringRow("allowed"), bytesRow([]byte(`{"description_binary":"AAE=","description_html":"<p>stored</p>","description_json":{"type":"doc"}}`))}}
	repository, _ = NewRepository(database)
	data, err := repository.VersionDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), uuid.New(), "Europe/Paris")
	if err != nil || !strings.Contains(string(data), `"description_json":{"type":"doc"}`) {
		t.Fatalf("data=%s err=%v", data, err)
	}
	for _, fragment := range []string{
		"version.description_binary IS NULL THEN NULL", "encode(version.description_binary, 'base64')",
		"'description_html', version.description_html", "'description_json', version.description_json",
		"version.deleted_at IS NULL", "project_page.deleted_at IS NULL",
		"JOIN workspaces workspace", "workspace.id = version.workspace_id",
		"project_member.is_active = true", "page.owned_by_id = $1 OR page.access = 0",
		"AT TIME ZONE $6",
	} {
		if !strings.Contains(database.queries[1], fragment) {
			t.Errorf("version detail query missing %q", fragment)
		}
	}
	if database.args[1][5] != "Europe/Paris" {
		t.Fatalf("version detail timezone=%#v", database.args[1][5])
	}
}

func TestVersionDetailMapsMissingVersionToNotFound(t *testing.T) {
	database := &databaseStub{rows: []pgx.Row{stringRow("allowed"), rowFunc(func(...any) error { return pgx.ErrNoRows })}}
	repository, _ := NewRepository(database)
	_, err := repository.VersionDetail(context.Background(), uuid.New(), "demo", uuid.New(), uuid.New(), uuid.New(), "UTC")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

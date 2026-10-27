package workspaceprojectreads

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/djangoapi"
	"github.com/CaliLuke/go-plane/internal/djangoinput"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// These assertions retain the write obligations from the retired Django
// test_projects.py and test_project_app.py contracts. Real DB/queue behavior
// remains covered by the source-recorded lifecycle replay.
type projectContractTx struct {
	pgx.Tx
	queries   []string
	args      [][]any
	committed bool
}

func (tx *projectContractTx) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	tx.queries = append(tx.queries, query)
	tx.args = append(tx.args, args)
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
func (tx *projectContractTx) Commit(context.Context) error { tx.committed = true; return nil }
func (*projectContractTx) Rollback(context.Context) error  { return nil }

type projectContractDB struct {
	workspaceProjectDatabase
	tx *projectContractTx
}

func (db *projectContractDB) Begin(context.Context) (pgx.Tx, error) { return db.tx, nil }
func (*projectContractDB) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected nontransactional write")
}

type projectContractEffects struct {
	tx    *projectContractTx
	err   error
	calls int
}

func (effects *projectContractEffects) Task(context.Context, string, []any, map[string]any) error {
	if !effects.tx.committed {
		return errors.New("activity published before creation committed")
	}
	effects.calls++
	return effects.err
}
func (*projectContractEffects) SoftDelete(context.Context, string, string, string) error {
	return errors.New("unexpected cascade")
}

func TestProjectCreateRetainsCreatorLeadDeduplicationAndDefaultWorkflow(t *testing.T) {
	const creator = "00000000-0000-4000-8000-000000000001"
	const other = "00000000-0000-4000-8000-000000000004"
	for _, app := range []bool{false, true} {
		for _, lead := range []string{"", creator, other} {
			t.Run(map[bool]string{false: "API", true: "session"}[app]+"/"+lead, func(t *testing.T) {
				fields := map[string]any{"name": "Created", "identifier": "NEW"}
				rows := []pgx.Row{projectLifecycleWorkspaceRow()}
				if lead != "" {
					fields["project_lead"] = lead
					rows = append(rows, projectLifecycleBoolRow(true)) // User relation exists.
				}
				if app {
					rows = append(rows, projectLifecycleBoolRow(false), projectLifecycleBoolRow(false)) // Unique name/identifier.
				}
				if lead != "" {
					rows = append(rows, projectLifecycleBoolRow(true)) // Active workspace member.
				}
				rows = append(rows, projectLifecycleBoolRow(false), workspaceProjectJSONRow(`{"id":"created"}`))
				tx := &projectContractTx{}
				db := &projectContractDB{workspaceProjectDatabase: workspaceProjectDatabase{rows: rows}, tx: tx}
				effects := &projectContractEffects{tx: tx}
				repo, _ := NewLifecycleRepository(db, effects, "", validProjectCover)
				if _, err := repo.CreateProject(context.Background(), creator, "space", djangoinput.Input{Fields: fields}, app); err != nil {
					t.Fatal(err)
				}
				members, properties, states := []string{}, 0, []string{}
				for i, query := range tx.queries {
					switch {
					case strings.HasPrefix(query, "INSERT INTO project_members"):
						members = append(members, tx.args[i][2].(string))
						if !strings.Contains(query, ",20,NULL,") || !strings.Contains(query, "65535,true,") {
							t.Fatalf("creator/lead was not initialized as active admin: %s", query)
						}
					case strings.HasPrefix(query, "INSERT INTO project_user_properties"):
						properties++
					case strings.HasPrefix(query, "INSERT INTO states"):
						states = append(states, tx.args[i][3].(string))
					}
				}
				wantMembers := []string{creator}
				if lead == other {
					wantMembers = append(wantMembers, other)
				}
				if !slices.Equal(members, wantMembers) || properties != len(wantMembers) {
					t.Fatalf("admin members=%v properties=%d want %v", members, properties, wantMembers)
				}
				// Django's State.objects hides the source's sixth, triage-group state.
				if !slices.Equal(states, []string{"Backlog", "Todo", "In Progress", "Done", "Cancelled", "Triage"}) {
					t.Fatalf("source default workflow=%v", states)
				}
				if !tx.committed || effects.calls != 1 {
					t.Fatalf("committed=%v activity calls=%d", tx.committed, effects.calls)
				}
			})
		}
	}
}

func TestAPIProjectCreateRetainsSuccessWhenPostCommitActivityBrokerFails(t *testing.T) {
	tx := &projectContractTx{}
	db := &projectContractDB{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		projectLifecycleWorkspaceRow(), projectLifecycleBoolRow(false), workspaceProjectJSONRow(`{"id":"persisted"}`),
	}}, tx: tx}
	effects := &projectContractEffects{tx: tx, err: errors.New("broker unavailable")}
	repo, _ := NewLifecycleRepository(db, effects, "", validProjectCover)
	body, err := repo.CreateProject(context.Background(), "00000000-0000-4000-8000-000000000001", "space", djangoinput.Input{Fields: map[string]any{"name": "Broker Down", "identifier": "BD"}}, false)
	if err != nil || string(body) != `{"id":"persisted"}` || !tx.committed || effects.calls != 1 {
		t.Fatalf("post-commit result=%s,%v committed=%v activity=%d", body, err, tx.committed, effects.calls)
	}
}

func TestSessionProjectRetainsRequiredBlankAndDuplicateFieldErrors(t *testing.T) {
	for _, vector := range []struct {
		name   string
		fields map[string]any
		create bool
		rows   []pgx.Row
		want   map[string]any
	}{
		{"empty create", map[string]any{}, true, nil, map[string]any{"name": []string{"This field is required."}, "identifier": []string{"This field is required."}}},
		{"missing name", map[string]any{"identifier": "NEW"}, true, []pgx.Row{projectLifecycleBoolRow(false)}, map[string]any{"name": []string{"This field is required."}}},
		{"missing identifier", map[string]any{"name": "Created"}, true, []pgx.Row{projectLifecycleBoolRow(false)}, map[string]any{"identifier": []string{"This field is required."}}},
		{"blank PATCH name", map[string]any{"name": ""}, false, nil, map[string]any{"name": []string{"This field may not be blank."}}},
		{"create duplicate name", map[string]any{"name": "Existing", "identifier": "NEW"}, true, []pgx.Row{projectLifecycleBoolRow(true), projectLifecycleBoolRow(false)}, map[string]any{"name": []string{"PROJECT_NAME_ALREADY_EXIST"}}},
		{"create duplicate identifier", map[string]any{"name": "Created", "identifier": "OLD"}, true, []pgx.Row{projectLifecycleBoolRow(false), projectLifecycleBoolRow(true)}, map[string]any{"identifier": []string{"PROJECT_IDENTIFIER_ALREADY_EXIST"}}},
		{"PATCH duplicate name", map[string]any{"name": "Existing"}, false, []pgx.Row{projectLifecycleBoolRow(true)}, map[string]any{"name": []string{"PROJECT_NAME_ALREADY_EXIST"}}},
		{"PATCH duplicate identifier", map[string]any{"identifier": "OLD"}, false, []pgx.Row{projectLifecycleBoolRow(true)}, map[string]any{"identifier": []string{"PROJECT_IDENTIFIER_ALREADY_EXIST"}}},
	} {
		t.Run(vector.name, func(t *testing.T) {
			db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: vector.rows}}
			repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
			_, err := repo.validateProjectFields(context.Background(), "workspace", "project", djangoinput.Input{Fields: vector.fields}, true, false, vector.create)
			var failure *djangoapi.HTTPError
			if !errors.As(err, &failure) || failure.Status != 400 || !reflect.DeepEqual(failure.Body, vector.want) {
				t.Fatalf("validation=%v want 400/%v", err, vector.want)
			}
			if len(db.execs) != 0 {
				t.Fatalf("rejected input wrote database: %v", db.execs)
			}
		})
	}
}

func TestAPIProjectCreateRejectsForeignLeadBeforeTransaction(t *testing.T) {
	db := &projectContractDB{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		projectLifecycleWorkspaceRow(), projectLifecycleBoolRow(true), projectLifecycleBoolRow(false),
	}}}
	repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	_, err := repo.CreateProject(context.Background(), "actor", "space", djangoinput.Input{Fields: map[string]any{"name": "Outsider", "identifier": "OUT", "project_lead": "00000000-0000-4000-8000-000000000009"}}, false)
	var failure *djangoapi.HTTPError
	if !errors.As(err, &failure) || failure.Status != 400 || !reflect.DeepEqual(failure.Body, map[string]any{"project_lead": []string{"The provided user is not a member of this workspace."}}) {
		t.Fatalf("foreign lead=%v", err)
	}
}

func TestSessionProjectPATCHRetainsMutableFields(t *testing.T) {
	db := &projectLifecycleDBFake{workspaceProjectDatabase: workspaceProjectDatabase{rows: []pgx.Row{
		workspaceProjectJSONRow(`{"workspace":"00000000-0000-4000-8000-000000000002","archived_at":null,"intake_view":false}`),
		projectLifecycleBoolRow(false), workspaceProjectJSONRow(`{"id":"updated"}`),
	}}}
	repo, _ := NewLifecycleRepository(db, &projectLifecycleEffectsFake{}, "", validProjectCover)
	fields := map[string]any{"name": "Updated Project", "description": "Updated description", "cycle_view": true, "module_view": false}
	if _, err := repo.UpdateProject(context.Background(), "actor", "space", "project", djangoinput.Input{Fields: fields}, true, false); err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	if err := json.Unmarshal([]byte(db.execArgs[0][0].(string)), &stored); err != nil {
		t.Fatal(err)
	}
	for field, value := range fields {
		if stored[field] != value {
			t.Errorf("stored %s=%v want %v", field, stored[field], value)
		}
	}
}

func (db *projectContractDB) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	return db.Begin(ctx)
}

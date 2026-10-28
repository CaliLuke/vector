//go:build integration

package workitemtypes_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	types "github.com/CaliLuke/go-plane/internal/workitemtypes"
	"github.com/CaliLuke/go-plane/internal/workitemtypes/property"
	"github.com/jackc/pgx/v5"
)

func TestDeletedItemUsageSerializesWithStructuralEdit(t *testing.T) {
	f := newTypeFixture(t)
	repo := types.NewRepository(f.db)
	catalog, err := repo.Mutate(f.ctx, f.owner, "types", "enable", "", types.Input{})
	if err != nil {
		t.Fatal(err)
	}
	def := property.Definition{ID: uuid.New().String(), Title: "Score", Kind: property.Number, Configuration: property.Configuration{Active: true}}
	properties, err := repo.MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: catalog.Revision, Action: "create", Definition: def})
	if err != nil {
		t.Fatal(err)
	}
	def = properties.Properties[0].Definition
	issue := uuid.New().String()
	f.exec(`INSERT INTO issues(id,workspace_id,project_id,type_id,name,description_json,description_html,priority,sort_order,sequence_id,is_draft,created_at,updated_at,deleted_at) VALUES($1,$2,$3,$4,'Deleted work','{}','','none',0,1,false,now(),now(),now())`, issue, f.workspace, f.project, catalog.Types[0].ID)
	editor, err := pgx.ConnectConfig(f.ctx, f.db.Config().Copy())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, editor, context.WithoutCancel(f.ctx)) })
	writer, err := f.db.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = writer.Rollback(context.WithoutCancel(f.ctx)) }()
	if _, err = writer.Exec(f.ctx, `UPDATE issues SET custom_properties=jsonb_build_object($2::text,0) WHERE id=$1`, issue, def.ID); err != nil {
		t.Fatal(err)
	}
	def.Kind = property.Boolean
	result := make(chan error, 1)
	go func() {
		_, failure := types.NewRepository(editor).MutateProperty(f.ctx, f.owner, "types", types.PropertyInput{Revision: properties.Revision, Action: "update", Definition: def})
		result <- failure
	}()
	// Observe a real database lock rather than assuming scheduling from a delay.
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	blocked := false
	for !blocked {
		if err = writer.QueryRow(f.ctx, `SELECT $1::integer=ANY(pg_blocking_pids($2::integer))`, f.db.PgConn().PID(), editor.PgConn().PID()).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		select {
		case err = <-result:
			t.Fatalf("schema edit did not wait for uncommitted usage: %v", err)
		case <-deadline.C:
			t.Fatal("schema edit never reached its lock")
		case <-tick.C:
		}
	}
	if err = writer.Commit(f.ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-result:
		assertTypeError(t, err, 422)
	case <-f.ctx.Done():
		t.Fatal(f.ctx.Err())
	}
	after, err := repo.PropertyCatalog(f.ctx, f.owner, "types")
	if err != nil || !after.Properties[0].Used || after.Properties[0].Definition.Kind != property.Number {
		t.Fatalf("concurrent usage bypassed structural lock: %+v %v", after, err)
	}
}

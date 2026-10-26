package workitems

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// enqueueEffect is used only by a repository bound to the mutation transaction.
// Admission is performed by the durable worker, never by a request goroutine.
func (r *Repository) enqueueEffect(ctx context.Context, kind string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `INSERT INTO work_item_effects(id,kind,payload) VALUES($1,$2,$3)`, uuid.NewString(), kind, raw)
	return err
}

func (r *Repository) softDelete(ctx context.Context, model, id string) error {
	if r.transactionalEffects {
		return r.enqueueEffect(ctx, "soft_delete", map[string]string{"app": "db", "model": model, "id": id})
	}
	return r.publisher.SoftDelete(ctx, "db", model, id)
}

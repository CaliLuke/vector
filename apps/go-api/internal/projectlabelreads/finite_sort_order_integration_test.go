//go:build integration

package projectlabelreads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/CaliLuke/go-plane/internal/projectconfiguration"
	"github.com/jackc/pgx/v5"
)

func TestLabelWritersRejectNonfiniteOrderWithoutChangingStorage(t *testing.T) {
	for _, api := range []bool{false, true} {
		t.Run(fmt.Sprintf("api-v1=%t", api), func(t *testing.T) {
			db := parentIntegrityFixture(t)
			methods := []string{http.MethodPost, http.MethodPatch}
			if !api {
				methods = append(methods, http.MethodPut)
			}
			for _, method := range methods {
				for _, value := range []string{`"NaN"`, `"Infinity"`, `"-Infinity"`, `1e309`, `"-1e309"`} {
					t.Run(method+"/"+value, func(t *testing.T) {
						ctx := t.Context()
						tx, err := db.Begin(ctx)
						if err != nil {
							t.Fatal(err)
						}
						defer func() {
							if rollbackErr := tx.Rollback(context.WithoutCancel(ctx)); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
								t.Errorf("rollback fixture: %v", rollbackErr)
							}
						}()
						before := labelStorageSnapshot(t, tx)
						body := fmt.Sprintf(`{"name":"Numeric validation regression","sort_order":%s}`, value)
						id := labelA
						if method == http.MethodPost {
							id = ""
						}
						err = writeLabelOrder(t, tx, api, method, id, body)
						var status int
						var response any
						var appFailure *HTTPError
						var apiFailure *projectconfiguration.HTTPError
						switch {
						case errors.As(err, &appFailure):
							status, response = appFailure.Status, appFailure.Body
						case errors.As(err, &apiFailure):
							status, response = apiFailure.Status, apiFailure.Body
						}
						if status != http.StatusBadRequest {
							t.Errorf("write error = %v, want HTTP 400", err)
						}
						encoded, marshalErr := json.Marshal(response)
						if marshalErr != nil {
							t.Fatal(marshalErr)
						}
						if string(encoded) != `{"sort_order":["A valid number is required."]}` {
							t.Errorf("validation body = %s", encoded)
						}
						if after := labelStorageSnapshot(t, tx); after != before {
							t.Errorf("rejected order changed stored labels or durable effects: before=%s after=%s", before, after)
						}
					})
				}
			}
		})
	}
}

func labelStorageSnapshot(t *testing.T, tx pgx.Tx) string {
	t.Helper()
	var snapshot string
	err := tx.QueryRow(t.Context(), `SELECT jsonb_build_object(
 'labels',(SELECT jsonb_agg(to_jsonb(label) ORDER BY id) FROM labels label),
 'effects',(SELECT jsonb_agg(to_jsonb(effect) ORDER BY id) FROM work_item_effects effect))::text`).Scan(&snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestLabelWritersPersistFiniteSortOrder(t *testing.T) {
	for _, api := range []bool{false, true} {
		t.Run(fmt.Sprintf("api-v1=%t", api), func(t *testing.T) {
			db := parentIntegrityFixture(t)
			tx, err := db.Begin(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if rollbackErr := tx.Rollback(context.WithoutCancel(t.Context())); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
					t.Errorf("rollback fixture: %v", rollbackErr)
				}
			}()
			if err := writeLabelOrder(t, tx, api, http.MethodPatch, labelA, `{"sort_order":123.25}`); err != nil {
				t.Fatal(err)
			}
			var saved, untouched float64
			if err := tx.QueryRow(t.Context(), `SELECT sort_order FROM labels WHERE id=$1`, labelA).Scan(&saved); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRow(t.Context(), `SELECT sort_order FROM labels WHERE id=$1`, foreignLabel).Scan(&untouched); err != nil {
				t.Fatal(err)
			}
			if saved != 123.25 || untouched != 40 {
				t.Fatalf("saved=%v foreign=%v, want 123.25 and 40", saved, untouched)
			}
		})
	}
}

func writeLabelOrder(t *testing.T, tx pgx.Tx, api bool, method, id, body string) error {
	t.Helper()
	if api {
		repository, err := projectconfiguration.NewRepository(tx)
		if err != nil {
			t.Fatal(err)
		}
		_, err = repository.WriteAPILabel(t.Context(), parentActor, "demo", parentProject, id, method, strings.NewReader(body), "application/json")
		return err
	}
	repository, err := NewWriteRepository(tx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.Write(t.Context(), parentActor, "demo", parentProject, id, method, strings.NewReader(body), "application/json")
	return err
}

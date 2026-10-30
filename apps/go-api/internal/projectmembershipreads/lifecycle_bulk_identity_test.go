package projectmembershipreads

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const bulkIdentityUser = "00000000-0000-4000-8000-00000000000a"

type bulkIdentityDB struct {
	role      int
	missing   bool
	existing  []string
	lookups   []string
	writes    []string
	arguments [][]any
}

func (db *bulkIdentityDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	return rowFunc(func(dest ...any) error {
		switch {
		case strings.Contains(query, "SELECT EXISTS"):
			*dest[0].(*bool) = true
		case strings.Contains(query, "SELECT wm.role"):
			db.lookups = append(db.lookups, args[1].(string))
			if db.missing {
				return pgx.ErrNoRows
			}
			*dest[0].(*int) = db.role
		case strings.Contains(query, "jsonb_agg(member_id::text)"):
			value := db.existing
			if value == nil {
				value = []string{}
			}
			body, _ := json.Marshal(value)
			*dest[0].(*[]byte) = body
		default:
			*dest[0].(*[]byte) = []byte(`[]`)
		}
		return nil
	})
}
func (db *bulkIdentityDB) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	db.writes = append(db.writes, query)
	db.arguments = append(db.arguments, args)
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func runBulkIdentity(t *testing.T, db *bulkIdentityDB, body string) error {
	t.Helper()
	repo, err := NewLifecycleRepository(db, &lifecycleTaskSpy{}, "http://localhost:8000")
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.lifecycleBulkCreate(context.Background(), LifecycleCommand{Body: strings.NewReader(body), ContentType: "application/json"})
	return err
}

func TestLifecycleBulkIdentityLookupOrderPrecedesLaterUUIDErrors(t *testing.T) {
	valid := `{"member_id":"` + bulkIdentityUser + `","role":15}`
	malformed := `{"member_id":"not-a-uuid","role":15}`
	for _, test := range []struct {
		first, second         string
		missing               bool
		role, status, lookups int
		message               string
	}{
		{valid, malformed, true, 15, 404, 1, "The required object does not exist."},
		{malformed, valid, true, 15, 400, 0, "Please provide valid detail"},
		{`{"member_id":"` + bulkIdentityUser + `","role":20}`, malformed, false, 5, 400, 1, "You cannot add a user with role higher than the workspace role"},
		{`{"member_id":null}`, `{"member_id":[]}`, true, 15, 400, 0, "Please provide valid detail"},
		{malformed, `{"member_id":{}}`, true, 15, 400, 0, "Please provide valid detail"},
	} {
		db := &bulkIdentityDB{role: test.role, missing: test.missing}
		err := runBulkIdentity(t, db, `{"members":[`+test.first+`,`+test.second+`]}`)
		lifecycleReviewError(t, err, test.status, map[string]any{"error": test.message})
		if len(db.lookups) != test.lookups || len(db.writes) != 0 {
			t.Fatalf("lookups=%v writes=%v", db.lookups, db.writes)
		}
	}
}

func TestLifecycleBulkCanonicalKeyMissRetainsDjangoKeyError(t *testing.T) {
	for _, identity := range []string{strings.ToUpper(bulkIdentityUser), strings.ReplaceAll(bulkIdentityUser, "-", "")} {
		db := &bulkIdentityDB{role: 15, existing: []string{bulkIdentityUser}}
		err := runBulkIdentity(t, db, `{"members":[{"member_id":"`+identity+`","role":15}]}`)
		lifecycleReviewError(t, err, 400, map[string]any{"error": "The required key does not exist."})
		if len(db.writes) != 0 || len(db.lookups) != 1 || db.lookups[0] != bulkIdentityUser {
			t.Fatalf("lookups=%v writes=%v", db.lookups, db.writes)
		}
	}
}

func TestLifecycleBulkPropertySortLookupRetainsOriginalIdentitySpelling(t *testing.T) {
	for _, identity := range []string{bulkIdentityUser, strings.ToUpper(bulkIdentityUser), strings.ReplaceAll(bulkIdentityUser, "-", "")} {
		db := &bulkIdentityDB{role: 15}
		if err := runBulkIdentity(t, db, `{"members":[{"member_id":"`+identity+`","role":15}]}`); err != nil {
			t.Fatal(err)
		}
		if len(db.writes) != 3 {
			t.Fatalf("writes=%v", db.writes)
		}
		var properties []map[string]any
		if err := json.Unmarshal([]byte(db.arguments[2][1].(string)), &properties); err != nil {
			t.Fatal(err)
		}
		if len(properties) != 1 || properties[0]["user_id"] != bulkIdentityUser || properties[0]["lookup_key"] != identity || !strings.Contains(db.writes[2], "WHERE user_id::text=incoming.lookup_key") {
			t.Fatalf("sort lookup lost raw spelling: properties=%#v SQL=%s", properties, db.writes[2])
		}
	}
}

func TestLifecycleBulkDictionaryRetainsPythonNumericKeyIdentity(t *testing.T) {
	for _, test := range []struct {
		body            string
		status, lookups int
		message         string
	}{
		{`{"members":[{"member_id":true,"role":5},{"member_id":1.0,"role":20}]}`, 400, 1, "You cannot add a user with role higher than the workspace role"},
		{`{"members":[{"member_id":1.0,"role":5},{"member_id":true,"role":20}]}`, 400, 0, "Please provide valid detail"},
	} {
		db := &bulkIdentityDB{role: 5}
		err := runBulkIdentity(t, db, test.body)
		lifecycleReviewError(t, err, test.status, map[string]any{"error": test.message})
		if len(db.lookups) != test.lookups || len(db.writes) != 0 {
			t.Fatalf("lookups=%v writes=%v", db.lookups, db.writes)
		}
	}
	key := func(value any) string {
		key, ok := lifecycleBulkIdentityKey(value)
		if !ok {
			t.Fatalf("unhashable scalar %#v", value)
		}
		return key
	}
	if key(true) != key(json.Number("1")) || key(true) != key(json.Number("1.0")) || key(false) != key(json.Number("-0.0")) || key("1") == key(json.Number("1")) || key(nil) == key("None") {
		t.Fatal("Python numeric/string/null dictionary identities changed")
	}
}

func TestLifecycleBulkOriginalValuesAreCoercedAgainBeforeExistingMemberLookup(t *testing.T) {
	db := &bulkIdentityDB{role: 15, existing: []string{"00000000-0000-0000-0000-000000000001"}}
	err := runBulkIdentity(t, db, `{"members":[{"member_id":true,"role":15},{"member_id":1.0,"role":15}]}`)
	lifecycleReviewError(t, err, 400, map[string]any{"error": "Please provide valid detail"})
	if len(db.lookups) != 1 || db.lookups[0] != "00000000-0000-0000-0000-000000000001" || len(db.writes) != 0 {
		t.Fatalf("lookups=%v writes=%v", db.lookups, db.writes)
	}
}

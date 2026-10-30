package backgroundtasks

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"

	dbutil "github.com/CaliLuke/go-plane/internal/database"

	"github.com/CaliLuke/go-plane/internal/authentication/password"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Source-recorded domain seed content and database defaults. PostgreSQL performs
// typed record conversion; this is not a second implementation of the ORM.
//
//go:embed workspace_seed_rows.json
var workspaceSeedRows []byte

type seedRecords struct {
	WorkspaceID string                      `json:"workspace_id"`
	OwnerID     string                      `json:"owner_id"`
	CapturedAt  time.Time                   `json:"captured_at"`
	Rows        map[string][]map[string]any `json:"rows"`
}
type WorkspaceSeedInput struct {
	WorkspaceID string    `json:"workspace_id"`
	At          time.Time `json:"at"`
}
type PostgresWorkspaceSeedStore struct {
	pool   *pgxpool.Pool
	webURL string
}

func NewPostgresWorkspaceSeedStore(pool *pgxpool.Pool, webURL string) (*PostgresWorkspaceSeedStore, error) {
	if pool == nil {
		return nil, errors.New("workspace seed database is required")
	}
	return &PostgresWorkspaceSeedStore{pool, webURL}, nil
}
func seedIdentifier(name string) string {
	letters := make([]rune, 0, 5)
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			letters = append(letters, r)
			if len(letters) == 5 {
				break
			}
		}
	}
	return cases.Upper(language.Und).String(string(letters))
}
func seedStableID(workspace uuid.UUID, key string) string {
	return uuid.NewSHA1(workspace, []byte("plane.workspace-seed.v1:"+key)).String()
}
func decodeSeedRecords() (seedRecords, error) {
	var records seedRecords
	err := json.Unmarshal(workspaceSeedRows, &records)
	return records, err
}

func (s *PostgresWorkspaceSeedStore) Seed(ctx context.Context, in WorkspaceSeedInput) (_ struct{}, resultErr error) {
	workspace, err := uuid.Parse(in.WorkspaceID)
	if err != nil || in.At.IsZero() {
		return struct{}{}, errors.New("workspace seed requires a UUID and timestamp")
	}
	records, err := decodeSeedRecords()
	if err != nil {
		return struct{}{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return struct{}{}, err
	}
	defer func() { resultErr = dbutil.RollbackTransaction(ctx, tx.Rollback, resultErr) }()
	// Serialize repeat admissions; the bot and all rows commit in one transaction.
	var name, zone, owner string
	err = tx.QueryRow(ctx, `SELECT name,timezone,owner_id::text FROM workspaces WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, workspace).Scan(&name, &zone, &owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return struct{}{}, nil
	}
	if err != nil {
		return struct{}{}, err
	}
	bot := seedStableID(workspace, records.Rows["users"][0]["id"].(string))
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1)`, bot).Scan(&exists); err != nil {
		return struct{}{}, err
	}
	if exists {
		return struct{}{}, nil
	}
	if _, err = tx.Exec(ctx, `SET CONSTRAINTS ALL DEFERRED`); err != nil {
		return struct{}{}, err
	}
	ids := map[string]string{records.WorkspaceID: workspace.String(), records.OwnerID: owner}
	for _, rows := range records.Rows {
		for _, row := range rows {
			key := row["id"].(string)
			ids[key] = seedStableID(workspace, key)
		}
	}
	hash, err := password.Default().Encode(uuid.NewString())
	if err != nil {
		return struct{}{}, err
	}
	host := "localhost"
	if u, e := url.Parse(s.webURL); e == nil && u.Hostname() != "" {
		host = u.Hostname()
	}
	username := "bot_user_" + workspace.String()
	type member struct {
		id   string
		role int
	}
	members := []member{{bot, 20}}
	membershipRows, err := tx.Query(ctx, `SELECT member_id::text,role FROM workspace_members WHERE workspace_id=$1 AND deleted_at IS NULL ORDER BY member_id`, workspace)
	if err != nil {
		return struct{}{}, err
	}
	for membershipRows.Next() {
		var m member
		if err = membershipRows.Scan(&m.id, &m.role); err != nil {
			membershipRows.Close()
			return struct{}{}, err
		}
		members = append(members, m)
	}
	err = membershipRows.Err()
	membershipRows.Close()
	if err != nil {
		return struct{}{}, err
	}
	tables := make([]string, 0, len(records.Rows))
	for table := range records.Rows {
		tables = append(tables, table)
	}
	slices.Sort(tables)
	for _, table := range tables {
		for _, source := range records.Rows[table] {
			row := map[string]any{}
			for key, value := range source {
				row[key] = value
				text, ok := value.(string)
				if !ok {
					continue
				}
				if mapped, ok := ids[text]; ok {
					row[key] = mapped
					continue
				}
				switch key {
				case "created_at", "updated_at", "date_joined", "last_active", "start_date", "end_date", "target_date":
					if stamp, e := time.Parse(time.RFC3339Nano, text); e == nil {
						row[key] = in.At.UTC().Add(stamp.Sub(records.CapturedAt)).Format(time.RFC3339Nano)
					} else if stamp, e := time.Parse(time.DateOnly, text); e == nil {
						base := records.CapturedAt.Truncate(24 * time.Hour)
						row[key] = in.At.UTC().Add(stamp.Sub(base)).Format(time.DateOnly)
					}
				}
			}
			switch table {
			case "users":
				row["username"] = username
				row["email"] = username + "@" + host
				row["password"] = hash
			case "projects":
				row["name"] = name
				row["identifier"] = seedIdentifier(name)
				row["timezone"] = zone
			case "issue_activities":
				row["epoch"] = float64(in.At.UnixNano()) / 1e9
			}
			if table == "project_members" || table == "project_user_properties" {
				for _, m := range members {
					row["id"] = seedStableID(workspace, table+":"+row["project_id"].(string)+":"+m.id)
					if table == "project_members" {
						row["member_id"] = m.id
						row["role"] = m.role
					} else {
						row["user_id"] = m.id
					}
					if err = insertSeedRow(ctx, tx, table, row); err != nil {
						return struct{}{}, err
					}
				}
			} else if err = insertSeedRow(ctx, tx, table, row); err != nil {
				return struct{}{}, err
			}
		}
	}
	return struct{}{}, tx.Commit(ctx)
}
func insertSeedRow(ctx context.Context, tx pgx.Tx, table string, row map[string]any) error {
	// Table names only come from the embedded, reviewed seed corpus.
	if strings.ContainsAny(table, `".; `) {
		return errors.New("invalid embedded seed table")
	}
	body, err := json.Marshal(row)
	if err != nil {
		return err
	}
	name := pgx.Identifier{"public", table}.Sanitize()
	// Preserve database defaults for columns added after the seed was captured.
	// A whole populated record would supply NULL for those missing columns.
	columns := make([]string, 0, len(row))
	for column := range row {
		columns = append(columns, pgx.Identifier{column}.Sanitize())
	}
	slices.Sort(columns)
	fields := strings.Join(columns, ",")
	_, err = tx.Exec(ctx, `INSERT INTO `+name+` (`+fields+`) SELECT `+fields+` FROM jsonb_populate_record(NULL::`+name+`,$1::jsonb)`, body)
	if err != nil {
		return fmt.Errorf("seed %s: %w", table, err)
	}
	return nil
}

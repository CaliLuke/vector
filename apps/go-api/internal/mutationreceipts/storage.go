package mutationreceipts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func prepare(cmd Command, op Operation) ([32]byte, error) {
	if cmd.ActorID == uuid.Nil || cmd.WorkspaceID == uuid.Nil || cmd.Tool == "" || cmd.Action == "" ||
		len(cmd.Tool) > 128 || len(cmd.Action) > 128 || op.Authorize == nil || op.Apply == nil ||
		cmd.ProjectID != nil && *cmd.ProjectID == uuid.Nil || cmd.RequestID != nil && *cmd.RequestID == uuid.Nil {
		return [32]byte{}, ErrInvalidRequest
	}
	decoder := json.NewDecoder(bytes.NewReader(cmd.NormalizedInput))
	decoder.UseNumber() // Never round large integer inputs through float64.
	var input map[string]any
	if err := decoder.Decode(&input); err != nil || input == nil {
		return [32]byte{}, ErrInvalidRequest
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return [32]byte{}, ErrInvalidRequest
	}
	canonical, err := json.Marshal(input)
	if err != nil {
		return [32]byte{}, ErrInvalidRequest
	}
	return sha256.Sum256(canonical), nil
}

func lock(ctx context.Context, tx pgx.Tx, cmd Command) error {
	// A collision only serializes unrelated requests. Full scope equality below,
	// rather than the advisory lock, determines receipt identity.
	key, err := json.Marshal([]any{"vector.mutation-receipt/v1", cmd.ActorID, cmd.WorkspaceID, cmd.ProjectID, cmd.Tool, cmd.Action, cmd.RequestID})
	if err != nil {
		return err
	}
	digest := sha256.Sum256(key)
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(binary.BigEndian.Uint64(digest[:8])))
	return err
}

const receiptScope = `actor_id=$1 AND workspace_id=$2 AND project_id IS NOT DISTINCT FROM $3::uuid AND tool=$4 AND action=$5 AND request_id=$6`

func scopeArgs(cmd Command) []any {
	return []any{cmd.ActorID, cmd.WorkspaceID, cmd.ProjectID, cmd.Tool, cmd.Action, cmd.RequestID}
}

func replay(ctx context.Context, tx pgx.Tx, cmd Command, hash [32]byte) (Result, error) {
	var value json.RawMessage
	var storedHash []byte
	var expires time.Time
	err := tx.QueryRow(ctx, `SELECT input_hash,result,replay_expires_at FROM mutation_receipts WHERE `+receiptScope+` AND replay_expires_at > clock_timestamp() FOR UPDATE`, scopeArgs(cmd)...).Scan(&storedHash, &value, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		return Result{}, nil
	}
	if err != nil {
		return Result{}, err
	}
	if !bytes.Equal(storedHash, hash[:]) {
		return Result{}, ErrConflict
	}
	return Result{Value: value, ReplayExpiresAt: &expires, Replayed: true}, nil
}

func record(ctx context.Context, tx pgx.Tx, cmd Command, hash [32]byte, value json.RawMessage) (Result, error) {
	args := append(scopeArgs(cmd), hash[:], value)
	var result Result
	var expires time.Time
	// Forty-eight hours leaves margin above the 24-hour contract. The advertised
	// database timestamp is authoritative. Only an expired receipt is replaceable;
	// callers for the same key are serialized until commit or rollback.
	err := tx.QueryRow(ctx, `INSERT INTO mutation_receipts
 (actor_id,workspace_id,project_id,tool,action,request_id,input_hash,result,created_at,replay_expires_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,clock_timestamp(),clock_timestamp()+INTERVAL '48 hours')
 ON CONFLICT (actor_id,workspace_id,project_id,tool,action,request_id) DO UPDATE
 SET input_hash=EXCLUDED.input_hash,result=EXCLUDED.result,created_at=EXCLUDED.created_at,replay_expires_at=EXCLUDED.replay_expires_at
 WHERE mutation_receipts.replay_expires_at <= clock_timestamp()
 RETURNING result,replay_expires_at`, args...).Scan(&result.Value, &expires)
	if err != nil {
		return Result{}, err
	}
	result.ReplayExpiresAt = &expires
	return result, nil
}

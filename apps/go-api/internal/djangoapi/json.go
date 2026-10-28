package djangoapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
)

// DjangoTimestampSQL renders a PostgreSQL timestamp as DRF's ISO timestamp,
// retaining Python's microsecond precision and the UTC Z suffix.
func DjangoTimestampSQL(column string) string {
	return `CASE WHEN ` + column + ` IS NULL THEN NULL ELSE to_char(` + column + `, 'YYYY-MM-DD"T"HH24:MI:SS') || CASE WHEN to_char(` + column + `, 'US') = '000000' THEN '' ELSE '.' || to_char(` + column + `, 'US') END || CASE WHEN to_char(` + column + `, 'TZH:TZM') = '+00:00' THEN 'Z' ELSE to_char(` + column + `, 'TZH:TZM') END END`
}

func JSON(ctx context.Context, db Database, query string, args ...any) (any, error) {
	var data []byte
	if err := db.QueryRow(ctx, query, args...).Scan(&data); err != nil {
		return nil, err
	}
	var result any
	if err := DecodeJSON(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DecodeJSON retains JSON numbers without rounding through float64. It accepts
// exactly one value, as json.Unmarshal does, including for embedded JSON inputs.
func DecodeJSON(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("expected one JSON value")
	}
	return nil
}

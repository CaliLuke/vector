// Package database contains shared PostgreSQL configuration for the Go backend.
package database

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ConfigurePool installs the standard-library UUID adapter on each connection.
// It preserves an existing AfterConnect hook.
func ConfigurePool(config *pgxpool.Config) error {
	if config == nil {
		return fmt.Errorf("pgx pool configuration is required")
	}
	previous := config.AfterConnect
	config.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		if previous != nil {
			if err := previous(ctx, connection); err != nil {
				return err
			}
		}
		return RegisterUUIDTypes(connection.TypeMap())
	}
	return nil
}

// RegisterUUIDTypes adds scalar and array support for the Go 1.27 uuid.UUID type.
// Call this function once for each pgtype map before the first query.
func RegisterUUIDTypes(typeMap *pgtype.Map) error {
	if typeMap == nil {
		return fmt.Errorf("pgx type map is required")
	}
	typeMap.TryWrapEncodePlanFuncs = append(
		[]pgtype.TryWrapEncodePlanFunc{tryWrapUUIDEncodePlan},
		typeMap.TryWrapEncodePlanFuncs...,
	)
	typeMap.TryWrapScanPlanFuncs = append(
		[]pgtype.TryWrapScanPlanFunc{tryWrapUUIDScanPlan},
		typeMap.TryWrapScanPlanFuncs...,
	)
	typeMap.RegisterDefaultPgType(uuid.UUID{}, "uuid")
	typeMap.RegisterDefaultPgType([]uuid.UUID{}, "_uuid")
	return nil
}

type uuidValue uuid.UUID

func (value uuidValue) UUIDValue() (pgtype.UUID, error) {
	return pgtype.UUID{Bytes: [16]byte(value), Valid: true}, nil
}

type wrapUUIDEncodePlan struct {
	next pgtype.EncodePlan
}

func (plan *wrapUUIDEncodePlan) SetNext(next pgtype.EncodePlan) {
	plan.next = next
}

func (plan *wrapUUIDEncodePlan) Encode(value any, buffer []byte) ([]byte, error) {
	return plan.next.Encode(uuidValue(value.(uuid.UUID)), buffer)
}

func tryWrapUUIDEncodePlan(value any) (pgtype.WrappedEncodePlanNextSetter, any, bool) {
	identifier, ok := value.(uuid.UUID)
	if !ok {
		return nil, nil, false
	}
	return &wrapUUIDEncodePlan{}, uuidValue(identifier), true
}

type uuidScanner struct {
	target *uuid.UUID
}

func (scanner *uuidScanner) ScanUUID(value pgtype.UUID) error {
	if !value.Valid {
		return fmt.Errorf("cannot scan NULL into uuid.UUID")
	}
	*scanner.target = uuid.UUID(value.Bytes)
	return nil
}

type wrapUUIDScanPlan struct {
	next pgtype.ScanPlan
}

func (plan *wrapUUIDScanPlan) SetNext(next pgtype.ScanPlan) {
	plan.next = next
}

func (plan *wrapUUIDScanPlan) Scan(source []byte, target any) error {
	return plan.next.Scan(source, &uuidScanner{target: target.(*uuid.UUID)})
}

func tryWrapUUIDScanPlan(target any) (pgtype.WrappedScanPlanNextSetter, any, bool) {
	identifier, ok := target.(*uuid.UUID)
	if !ok {
		return nil, nil, false
	}
	scanner := &uuidScanner{target: identifier}
	return &wrapUUIDScanPlan{}, scanner, true
}

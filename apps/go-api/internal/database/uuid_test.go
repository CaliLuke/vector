package database

import (
	"reflect"
	"testing"
	"uuid"

	generated "github.com/CaliLuke/go-plane/internal/database/db"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestConfigurePoolRequiresConfiguration(t *testing.T) {
	if err := ConfigurePool(nil); err == nil {
		t.Fatal("ConfigurePool() error = nil, want missing configuration error")
	}
}

func TestRegisterUUIDTypesSupportsScalarAndArrayPlans(t *testing.T) {
	typeMap := pgtype.NewMap()
	if err := RegisterUUIDTypes(typeMap); err != nil {
		t.Fatalf("RegisterUUIDTypes() error = %v", err)
	}

	identifiers := []uuid.UUID{
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290271"),
		uuid.MustParse("0198a28d-d227-7752-82ac-bc4c1c290272"),
	}

	scalarBytes, err := typeMap.Encode(pgtype.UUIDOID, pgtype.BinaryFormatCode, identifiers[0], nil)
	if err != nil {
		t.Fatalf("encode scalar UUID: %v", err)
	}
	var scalar uuid.UUID
	if err := typeMap.Scan(pgtype.UUIDOID, pgtype.BinaryFormatCode, scalarBytes, &scalar); err != nil {
		t.Fatalf("scan scalar UUID: %v", err)
	}
	if scalar != identifiers[0] {
		t.Fatalf("scalar UUID = %s, want %s", scalar, identifiers[0])
	}
	if err := typeMap.Scan(pgtype.UUIDOID, pgtype.BinaryFormatCode, nil, &scalar); err == nil {
		t.Fatal("scan NULL scalar UUID error = nil, want non-null type error")
	}

	arrayBytes, err := typeMap.Encode(pgtype.UUIDArrayOID, pgtype.BinaryFormatCode, identifiers, nil)
	if err != nil {
		t.Fatalf("encode UUID array: %v", err)
	}
	var array []uuid.UUID
	if err := typeMap.Scan(pgtype.UUIDArrayOID, pgtype.BinaryFormatCode, arrayBytes, &array); err != nil {
		t.Fatalf("scan UUID array: %v", err)
	}
	if !reflect.DeepEqual(array, identifiers) {
		t.Fatalf("UUID array = %v, want %v", array, identifiers)
	}
}

func TestGeneratedUUIDFieldTypes(t *testing.T) {
	row := generated.PlaneGoUuidProbe{}
	if reflect.TypeOf(row.ID) != reflect.TypeFor[uuid.UUID]() {
		t.Fatalf("generated ID type = %T, want uuid.UUID", row.ID)
	}
	if reflect.TypeOf(row.NullableID) != reflect.TypeFor[pgtype.UUID]() {
		t.Fatalf("generated nullable ID type = %T, want pgtype.UUID", row.NullableID)
	}
	if reflect.TypeOf(row.RelatedIDs) != reflect.TypeFor[[]uuid.UUID]() {
		t.Fatalf("generated related IDs type = %T, want []uuid.UUID", row.RelatedIDs)
	}
}

func TestRegisterUUIDTypesRequiresMap(t *testing.T) {
	if err := RegisterUUIDTypes(nil); err == nil {
		t.Fatal("RegisterUUIDTypes() error = nil, want missing map error")
	}
}

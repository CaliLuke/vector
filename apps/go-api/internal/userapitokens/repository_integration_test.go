//go:build integration

package userapitokens_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	"github.com/CaliLuke/go-plane/internal/bootstrap"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	postgresfixture "github.com/CaliLuke/go-plane/internal/testsupport/postgres"
	"github.com/CaliLuke/go-plane/internal/userapitokens"
	"github.com/jackc/pgx/v5"
)

func TestCreateTokenOnApplicationSchema(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pair, err := postgresfixture.StartUncloned(ctx, bootstrap.MigrateSchema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.Close(t, pair) })
	connection, err := pgx.Connect(ctx, pair.SeedURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testsupport.CloseContext(t, connection, context.WithoutCancel(ctx)) })
	owner := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	_, err = connection.Exec(ctx, `INSERT INTO users
	(id,username,email,password,first_name,last_name,avatar,date_joined,created_at,updated_at,
	last_location,created_location,is_superuser,is_managed,is_password_expired,is_active,
	is_staff,is_email_verified,is_password_autoset,token,user_timezone,last_login_ip,
	last_logout_ip,last_login_medium,last_login_uagent,is_bot,display_name,is_email_valid,is_password_reset_required)
	VALUES ($1,'token-owner','token-owner@example.test','!','','','',now(),now(),now(),
	'','',false,false,false,true,false,true,false,'','UTC','','','','',false,'Token owner',true,false)`, owner)
	if err != nil {
		t.Fatal(err)
	}
	repository, err := userapitokens.NewRepository(connection)
	if err != nil {
		t.Fatal(err)
	}

	created, err := repository.Create(ctx, owner, "Sync", "Integration key", "plane_api_regression", nil)
	if err != nil {
		t.Fatalf("create API token on the application schema: %v", err)
	}
	var token struct {
		ID               string    `json:"id"`
		CreatedAt        time.Time `json:"created_at"`
		UpdatedAt        time.Time `json:"updated_at"`
		CreatedBy        string    `json:"created_by"`
		IsActive         bool      `json:"is_active"`
		IsService        bool      `json:"is_service"`
		AllowedRateLimit string    `json:"allowed_rate_limit"`
	}
	if err = json.Unmarshal(created, &token); err != nil {
		t.Fatal(err)
	}
	if _, err = uuid.Parse(token.ID); err != nil || token.CreatedAt.IsZero() || token.UpdatedAt.IsZero() {
		t.Fatal("created token is missing its generated ID or timestamps")
	}
	if token.CreatedBy != owner.String() || !token.IsActive || token.IsService || token.AllowedRateLimit != "60/min" {
		t.Fatalf("incorrect token defaults: %+v", token)
	}
	validator, err := apikey.New(connection)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = validator.Validate(ctx, "plane_api_regression"); err != nil {
		t.Fatalf("new token cannot authenticate: %v", err)
	}
}

package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"uuid"

	api "github.com/CaliLuke/go-plane/apiuseridentity/gen/api_user_identity"
	"github.com/CaliLuke/go-plane/internal/authentication/apikey"
	testsupport "github.com/CaliLuke/go-plane/internal/testsupport"
	"github.com/CaliLuke/loom/security"
	googleuuid "github.com/google/uuid"
)

type apiIdentityRepositoryStub struct {
	body  json.RawMessage
	reads int
}

func (r *apiIdentityRepositoryStub) APIV1Identity(context.Context, uuid.UUID) (json.RawMessage, error) {
	r.reads++
	return r.body, nil
}
func TestAPIIdentityThrottlePrecedesTimezoneAndRead(t *testing.T) {
	repository := &apiIdentityRepositoryStub{}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: googleuuid.New(), Timezone: "Invalid/Timezone"}}
	throttle := &activityThrottleStub{rate: apikey.Rate{Allowed: false}}
	value, _ := NewAPIUserIdentity(repository, keys, throttle)
	service := value.(*apiUserIdentityReadsService)
	status, body := service.AuthorizeAPIIdentityProtocol(context.Background(), "key")
	if status != 429 || body["detail"] == nil || throttle.calls != 1 || repository.reads != 0 {
		t.Fatalf("status=%d body=%v throttle=%d reads=%d", status, body, throttle.calls, repository.reads)
	}
}
func TestAPIIdentityRendersCompactJSONBeforeCompression(t *testing.T) {
	repository := &apiIdentityRepositoryStub{body: json.RawMessage(`{"id": "user", "email": null, "first_name": "Zoë"}`)}
	keys := &activityAPIKeyStub{principal: apikey.Principal{UserID: googleuuid.New(), Timezone: "UTC"}}
	value, _ := NewAPIUserIdentity(repository, keys, &activityThrottleStub{rate: apikey.Rate{Allowed: true}})
	service := value.(*apiUserIdentityReadsService)
	ctx, err := service.APIKeyAuth(context.Background(), "key", &security.APIKeyScheme{})
	if err != nil {
		t.Fatal(err)
	}
	_, body, err := service.GetAPIV1UsersMe(ctx, &api.GetAPIV1UsersMePayload{})
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, body)
	got, err := io.ReadAll(body)
	if err != nil || string(got) != `{"id":"user","email":null,"first_name":"Zoë"}` {
		t.Fatalf("body=%s err=%v", got, err)
	}
}
func TestAPIIdentityRejectsInvalidKeyBeforeThrottle(t *testing.T) {
	throttle := &activityThrottleStub{}
	value, _ := NewAPIUserIdentity(&apiIdentityRepositoryStub{}, &activityAPIKeyStub{err: apikey.ErrInvalid}, throttle)
	service := value.(*apiUserIdentityReadsService)
	_, err := service.APIKeyAuth(context.Background(), "invalid", &security.APIKeyScheme{})
	var denied *api.APIUserIdentityErrorBody
	if !errors.As(err, &denied) || denied.Name != "status_403" || throttle.calls != 0 {
		t.Fatalf("err=%v throttle=%d", err, throttle.calls)
	}
}

package systemapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	useraccountsapi "github.com/CaliLuke/go-plane/gen/user_accounts"
	"github.com/CaliLuke/go-plane/internal/authentication/djangosession"
	useraccountsstore "github.com/CaliLuke/go-plane/internal/useraccounts"
	"github.com/CaliLuke/loom/security"
)

type userAccountsRepository interface {
	List(context.Context, uuid.UUID) (json.RawMessage, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (json.RawMessage, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}

type userAccountsSessionValidator interface {
	Validate(context.Context, string) (djangosession.Principal, error)
}

type userAccountsService struct {
	repository userAccountsRepository
	sessions   userAccountsSessionValidator
}

func NewUserAccounts(repository userAccountsRepository, sessions userAccountsSessionValidator) (useraccountsapi.Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("user accounts repository is required")
	}
	if sessions == nil {
		return nil, fmt.Errorf("Django session validator is required")
	}
	return &userAccountsService{repository: repository, sessions: sessions}, nil
}

func (service *userAccountsService) APIKeyAuth(ctx context.Context, key string, _ *security.APIKeyScheme) (context.Context, error) {
	principal, err := service.sessions.Validate(ctx, key)
	if err != nil {
		return ctx, &useraccountsapi.UserAccountUnauthenticated{Detail: unauthenticatedDetail}
	}
	userID, err := uuid.Parse(principal.UserID)
	if err != nil {
		return ctx, &useraccountsapi.UserAccountUnauthenticated{Detail: unauthenticatedDetail}
	}
	return context.WithValue(ctx, userAccountsPrincipalKey{}, userID), nil
}

func (service *userAccountsService) List(ctx context.Context, _ *useraccountsapi.ListPayload) ([]*useraccountsapi.UserAccount, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	data, err := service.repository.List(ctx, userID)
	if err != nil {
		return nil, userAccountInternalError()
	}
	var result []*useraccountsapi.UserAccount
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, userAccountInternalError()
	}
	return result, nil
}

func (service *userAccountsService) Retrieve(ctx context.Context, payload *useraccountsapi.RetrievePayload) (*useraccountsapi.UserAccount, error) {
	userID, err := service.userID(ctx)
	if err != nil {
		return nil, err
	}
	accountID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return nil, userAccountNotFoundError()
	}
	data, err := service.repository.Get(ctx, userID, accountID)
	if err != nil {
		return nil, userAccountRepositoryError(err)
	}
	var result useraccountsapi.UserAccount
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, userAccountInternalError()
	}
	return &result, nil
}

func (service *userAccountsService) Delete(ctx context.Context, payload *useraccountsapi.DeletePayload) error {
	userID, err := service.userID(ctx)
	if err != nil {
		return err
	}
	accountID, err := uuid.Parse(payload.Pk)
	if err != nil {
		return userAccountNotFoundError()
	}
	return userAccountRepositoryError(service.repository.Delete(ctx, userID, accountID))
}

func (service *userAccountsService) DeleteCollection(ctx context.Context, _ *useraccountsapi.DeleteCollectionPayload) error {
	if _, err := service.userID(ctx); err != nil {
		return err
	}
	return userAccountInternalError()
}

func (service *userAccountsService) userID(ctx context.Context) (uuid.UUID, error) {
	userID, ok := ctx.Value(userAccountsPrincipalKey{}).(uuid.UUID)
	if !ok {
		return uuid.Nil(), &useraccountsapi.UserAccountUnauthenticated{Detail: unauthenticatedDetail}
	}
	return userID, nil
}

func userAccountRepositoryError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, useraccountsstore.ErrNotFound) {
		return userAccountNotFoundError()
	}
	return userAccountInternalError()
}

func userAccountNotFoundError() error {
	return &useraccountsapi.UserAccountNotFound{Message: "The required object does not exist."}
}

func userAccountInternalError() error {
	return &useraccountsapi.UserAccountInternalError{Message: "Something went wrong please try again later"}
}

type userAccountsPrincipalKey struct{}

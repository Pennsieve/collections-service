package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/store/users"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/stretchr/testify/require"
)

type GetUserFunc func(ctx context.Context, userID int64) (users.GetUserResponse, error)
type UsersStore struct {
	GetUserFunc
}

func NewUsersStore() *UsersStore {
	return &UsersStore{}
}

func (u *UsersStore) GetUser(ctx context.Context, userID int64) (users.GetUserResponse, error) {
	if u.GetUserFunc == nil {
		panic("mock GetUser function not set")
	}
	return u.GetUserFunc(ctx, userID)
}

func (u *UsersStore) WithGetUserFunc(getUserFunc GetUserFunc) *UsersStore {
	u.GetUserFunc = getUserFunc
	return u
}

func NewGetUserFunc(t require.TestingT, user userstest.User) GetUserFunc {
	return func(_ context.Context, userID int64) (users.GetUserResponse, error) {
		require.Equal(t, user.GetID(), userID)
		userResponse := users.GetUserResponse{
			FirstName:     emptyStringToNil(user.GetFirstName()),
			MiddleInitial: emptyStringToNil(user.GetMiddleInitial()),
			LastName:      emptyStringToNil(user.GetLastName()),
			Degree:        emptyStringToNil(user.GetDegree()),
		}
		if orcidAuth := user.GetORCIDAuthorization(); orcidAuth != nil {
			userResponse.ORCID = &orcidAuth.ORCID
		}
		return userResponse, nil
	}
}

func emptyStringToNil(value string) *string {
	if len(value) == 0 {
		return nil
	}
	return &value
}

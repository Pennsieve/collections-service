package mocks

import (
	"context"
	"github.com/pennsieve/collections-service/internal/api/store/users"
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

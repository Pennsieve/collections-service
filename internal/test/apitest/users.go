package apitest

import (
	"fmt"
	"github.com/google/uuid"
)

type SeedUser struct {
	ID           int64
	NodeID       string
	IsSuperAdmin bool
}

// These users are already present in the Pennsieve seed DB Docker container used for tests

var User = SeedUser{
	ID:           1,
	NodeID:       "N:user:99f02be5-009c-4ecd-9006-f016d48628bf",
	IsSuperAdmin: false,
}

var User2 = SeedUser{
	ID:           2,
	NodeID:       "N:user:29cb5354-b471-4a72-adae-6fcb262447d9",
	IsSuperAdmin: false,
}

var SuperUser = SeedUser{
	ID:           3,
	NodeID:       "N:user:4e8c459b-bffb-49e1-8c6a-de2d8190d84e",
	IsSuperAdmin: true,
}

type TestUser struct {
	ID           *int64
	NodeID       string
	Email        string
	IsSuperAdmin bool
}

func NewTestUser(options ...TestUserOption) *TestUser {
	testUser := &TestUser{
		NodeID: fmt.Sprintf("N:user:%s", uuid.NewString()),
		Email:  fmt.Sprintf("%s@example.com", uuid.NewString()),
	}
	for _, option := range options {
		option(testUser)
	}
	return testUser
}

type TestUserOption func(testUser *TestUser)

func WithID(id int64) TestUserOption {
	return func(testUser *TestUser) {
		testUser.ID = &id
	}
}

func WithNodeID(nodeID string) TestUserOption {
	return func(testUser *TestUser) {
		testUser.NodeID = nodeID
	}
}

func WithIsSuperAdmin(isSuperAdmin bool) TestUserOption {
	return func(testUser *TestUser) {
		testUser.IsSuperAdmin = isSuperAdmin
	}
}

func WithEmail(email string) TestUserOption {
	return func(testUser *TestUser) {
		testUser.Email = email
	}
}

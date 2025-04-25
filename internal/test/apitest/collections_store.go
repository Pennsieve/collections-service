package apitest

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/require"
	"slices"
)

// ExpectedCollection is what we expect the collection to look like
// in Postgres, so it doesn't include things not persisted there. Like banners for
// example.
type ExpectedCollection struct {
	ID          *int64
	Name        string
	Description string
	// NodeID is optional since it may not be known depending
	// on the level we are testing. We can have an expected nodeID
	// if testing collection creation at the store level, but not at the route handling level
	// for example
	NodeID *string
	Users  []ExpectedUser
	DOIs   ExpectedDOIs
}

func NewExpectedCollection() *ExpectedCollection {
	return &ExpectedCollection{
		Name:        uuid.NewString(),
		Description: uuid.NewString(),
	}
}

func (c *ExpectedCollection) WithNodeID() *ExpectedCollection {
	nodeID := uuid.NewString()
	c.NodeID = &nodeID
	return c
}

// WithMockID is meant for cases where this ExpectedCollection is not persisted to the test DB
// but still needs an ID for the test.
func (c *ExpectedCollection) WithMockID(mockID int64) *ExpectedCollection {
	c.ID = &mockID
	return c
}

type ExpectedUser struct {
	UserID        int64
	PermissionBit pgdb.DbPermission
}

func (c *ExpectedCollection) WithUser(userID int64, permission pgdb.DbPermission) *ExpectedCollection {
	c.Users = append(c.Users, ExpectedUser{userID, permission})
	return c
}

type ExpectedDOI struct {
	DOI string
}

// WithDOIs adds to the current ExpectedDOI slice
func (c *ExpectedCollection) WithDOIs(dois ...string) *ExpectedCollection {
	for _, doi := range dois {
		c.DOIs = append(c.DOIs, ExpectedDOI{DOI: doi})
	}
	return c
}

// SetDOIs replaces the current ExpectedDOI slice with the given DOIs
func (c *ExpectedCollection) SetDOIs(dois ...string) *ExpectedCollection {
	var newDOIs []ExpectedDOI
	for _, doi := range dois {
		newDOIs = append(newDOIs, ExpectedDOI{DOI: doi})
	}
	c.DOIs = newDOIs
	return c
}

func (c *ExpectedCollection) WithNPennsieveDOIs(n int) *ExpectedCollection {
	var dois []string
	for i := 0; i < n; i++ {
		dois = append(dois, NewPennsieveDOI())
	}
	return c.WithDOIs(dois...)
}

type ExpectedDOIs []ExpectedDOI

func (d ExpectedDOIs) Strings() []string {
	if len(d) == 0 {
		return nil
	}
	strs := make([]string, len(d))
	for i, doi := range d {
		strs[i] = doi.DOI
	}
	return strs
}

func (c *ExpectedCollection) GetCollectionFunc(t require.TestingT) mocks.GetCollectionFunc {
	test.Helper(t)
	return func(ctx context.Context, userID int64, nodeID string) (store.GetCollectionResponse, error) {
		require.NotNil(t, c.NodeID, "expected collection does not have NodeID set")
		require.Equal(t, *c.NodeID, nodeID, "expected NodeID is %s; got %s", *c.NodeID, nodeID)
		userIdx := slices.IndexFunc(c.Users, func(user ExpectedUser) bool {
			return user.UserID == userID
		})
		require.NotEqual(t, -1, userIdx, "given user %d has no permission for collection %s", userID, nodeID)
		user := c.Users[userIdx]
		collectionBase := store.CollectionBase{
			NodeID:      nodeID,
			Name:        c.Name,
			Description: c.Description,
			Size:        len(c.DOIs),
			UserRole:    user.PermissionBit.ToRole(),
		}
		if c.ID != nil {
			// The id will be set if this ExpectedCollection was created by an ExpectationDB, but
			// we may not know it otherwise. We can set it manually in a test if required.
			collectionBase.ID = *c.ID
		}
		return store.GetCollectionResponse{
			CollectionBase: collectionBase,
			DOIs:           c.DOIs.Strings(),
		}, nil
	}
}

func (c *ExpectedCollection) UpdateCollectionFunc(t require.TestingT) mocks.UpdateCollectionFunc {
	return func(ctx context.Context, userID int64, collectionID int64, update store.UpdateCollectionRequest) (store.GetCollectionResponse, error) {
		test.Helper(t)
		require.NotNil(t, c.NodeID, "expected collection does not have NodeID set")
		require.NotNil(t, c.ID, "expected collection does not have ID set")
		require.Equal(t, *c.ID, collectionID, "expected ID is %d; got %d", *c.ID, collectionID)
		userIdx := slices.IndexFunc(c.Users, func(user ExpectedUser) bool {
			return user.UserID == userID
		})
		require.NotEqual(t, -1, userIdx, "given user %d has no permission for collection %d", userID, collectionID)
		user := c.Users[userIdx]

		updatedName := c.Name
		if update.Name != nil {
			updatedName = *update.Name
		}

		updatedDescription := c.Description
		if update.Description != nil {
			updatedDescription = *update.Description
		}

		toDeleteSet := map[string]bool{}
		for _, toDelete := range update.DOIs.Remove {
			toDeleteSet[toDelete] = true
		}

		var updatedDOIs []string
		for _, doi := range c.DOIs.Strings() {
			if _, deleted := toDeleteSet[doi]; !deleted {
				updatedDOIs = append(updatedDOIs, doi)
			}
		}

		updatedDOIs = append(updatedDOIs, update.DOIs.Add...)

		collectionBase := store.CollectionBase{
			NodeID:      *c.NodeID,
			ID:          *c.ID,
			Name:        updatedName,
			Description: updatedDescription,
			Size:        len(updatedDOIs),
			UserRole:    user.PermissionBit.ToRole(),
		}
		return store.GetCollectionResponse{
			CollectionBase: collectionBase,
			DOIs:           updatedDOIs,
		}, nil
	}
}

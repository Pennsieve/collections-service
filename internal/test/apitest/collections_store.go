package apitest

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
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

// WithRandomID is meant for cases where this ExpectedCollection is not persisted to the test DB
// but still needs an ID for the test, but you don't care what it is.
func (c *ExpectedCollection) WithRandomID() *ExpectedCollection {
	id := rand.Int64() + 1
	c.ID = &id
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
	DOI        string
	Datasource datasource.DOIDatasource
}

// WithDOIs adds to the current ExpectedDOI slice
func (c *ExpectedCollection) WithDOIs(dois ...store.DOI) *ExpectedCollection {
	for _, doi := range dois {
		c.DOIs = append(c.DOIs, ExpectedDOI{DOI: doi.Value, Datasource: doi.Datasource})
	}
	return c
}

// SetDOIs replaces the current ExpectedDOI slice with the given DOIs
func (c *ExpectedCollection) SetDOIs(dois ...store.DOI) *ExpectedCollection {
	var newDOIs []ExpectedDOI
	for _, doi := range dois {
		newDOIs = append(newDOIs, ExpectedDOI{DOI: doi.Value, Datasource: doi.Datasource})
	}
	c.DOIs = newDOIs
	return c
}

func (c *ExpectedCollection) WithNPennsieveDOIs(n int) *ExpectedCollection {
	var dois []store.DOI
	for i := 0; i < n; i++ {
		dois = append(dois, NewPennsieveDOI())
	}
	return c.WithDOIs(dois...)
}

// WithPublicDatasets appends the DOIs of the given publicDatasets to the ExpectedDOIs
func (c *ExpectedCollection) WithPublicDatasets(publicDatasets ...dto.PublicDataset) *ExpectedCollection {
	for _, publicDataset := range publicDatasets {
		c.DOIs = append(c.DOIs, ExpectedDOI{DOI: publicDataset.DOI, Datasource: datasource.Pennsieve})
	}
	return c
}

// WithTombstones appends the DOIs of the given tombstones to the ExpectedDOIs
func (c *ExpectedCollection) WithTombstones(tombstones ...dto.Tombstone) *ExpectedCollection {
	for _, tombstone := range tombstones {
		c.DOIs = append(c.DOIs, ExpectedDOI{DOI: tombstone.DOI, Datasource: datasource.Pennsieve})
	}
	return c
}

// SetPublicDatasets replaces the current ExpectedDOI slice with the DOIs of the given publicDatasets
func (c *ExpectedCollection) SetPublicDatasets(publicDatasets ...dto.PublicDataset) *ExpectedCollection {
	var newDOIs []ExpectedDOI
	for _, publicDataset := range publicDatasets {
		newDOIs = append(newDOIs, ExpectedDOI{DOI: publicDataset.DOI, Datasource: datasource.Pennsieve})
	}
	c.DOIs = newDOIs
	return c
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

func (d ExpectedDOIs) AsDOIs() []store.DOI {
	if len(d) == 0 {
		return nil
	}
	strs := make([]store.DOI, len(d))
	for i, doi := range d {
		strs[i] = store.DOI{
			Value:      doi.DOI,
			Datasource: doi.Datasource,
		}
	}
	return strs
}

func (c *ExpectedCollection) ToGetCollectionResponse(t require.TestingT, expectedUserID int64) store.GetCollectionResponse {
	test.Helper(t)
	require.NotNil(t, c.ID, "expected collection does not have ID set")
	require.NotNil(t, c.NodeID, "expected collection does not have NodeID set")
	userIdx := slices.IndexFunc(c.Users, func(user ExpectedUser) bool {
		return user.UserID == expectedUserID
	})
	require.NotEqual(t, -1, userIdx, "given user %d has no permission for expected collection", expectedUserID)
	user := c.Users[userIdx]
	collectionBase := store.CollectionBase{
		ID:          *c.ID,
		NodeID:      *c.NodeID,
		Name:        c.Name,
		Description: c.Description,
		Size:        len(c.DOIs),
		UserRole:    user.PermissionBit.ToRole(),
	}
	return store.GetCollectionResponse{
		CollectionBase: collectionBase,
		DOIs:           c.DOIs.AsDOIs(),
	}
}

func (c *ExpectedCollection) GetCollectionFunc(t require.TestingT) mocks.GetCollectionFunc {
	test.Helper(t)
	return func(ctx context.Context, userID int64, nodeID string) (store.GetCollectionResponse, error) {
		require.Equal(t, *c.NodeID, nodeID, "expected NodeID is %s; got %s", *c.NodeID, nodeID)
		return c.ToGetCollectionResponse(t, userID), nil
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

		var updatedDOIs []store.DOI
		for _, doi := range c.DOIs.AsDOIs() {
			if _, deleted := toDeleteSet[doi.Value]; !deleted {
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

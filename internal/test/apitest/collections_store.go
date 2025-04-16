package apitest

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/store"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/stretchr/testify/require"
	"slices"
)

// ExpectedCollection is what we expect the collection to look like
// in Postgres, so it doesn't include things not persisted there. Like banners for
// example.
type ExpectedCollection struct {
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

func (c *ExpectedCollection) WithDOIs(dois ...string) *ExpectedCollection {
	for _, doi := range dois {
		c.DOIs = append(c.DOIs, ExpectedDOI{DOI: doi})
	}
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
	return func(ctx context.Context, userID int64, nodeID string) (*store.GetCollectionResponse, error) {
		require.NotNil(t, c.NodeID, "expected collection does not have NodeID set")
		require.Equal(t, *c.NodeID, nodeID, "expected NodeID is %s; got %s", *c.NodeID, nodeID)
		userIdx := slices.IndexFunc(c.Users, func(user ExpectedUser) bool {
			return user.UserID == userID
		})
		require.NotEqual(t, -1, userIdx, "given user %d has no permission for collection %s", userID, nodeID)
		user := c.Users[userIdx]
		return &store.GetCollectionResponse{
			CollectionBase: store.CollectionBase{
				NodeID:      nodeID,
				Name:        c.Name,
				Description: c.Description,
				Size:        len(c.DOIs),
				UserRole:    user.PermissionBit.ToRole().String(),
			},
			DOIs: c.DOIs.Strings(),
		}, nil
	}
}

package apitest

import (
	"context"
	"github.com/google/uuid"
	"github.com/pennsieve/collections-service/internal/api/datasource"
	"github.com/pennsieve/collections-service/internal/api/dto"
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/service"
	"github.com/pennsieve/collections-service/internal/api/service/jwtdiscover"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"github.com/pennsieve/collections-service/internal/test"
	"github.com/pennsieve/collections-service/internal/test/mocks"
	"github.com/pennsieve/collections-service/internal/test/userstest"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/pgdb"
	"github.com/pennsieve/pennsieve-go-core/pkg/models/role"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
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

func (c *ExpectedCollection) WithDescription(description string) *ExpectedCollection {
	c.Description = description
	return c
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
func (c *ExpectedCollection) WithDOIs(dois ...collections.DOI) *ExpectedCollection {
	for _, doi := range dois {
		c.DOIs = append(c.DOIs, ExpectedDOI{DOI: doi.Value, Datasource: doi.Datasource})
	}
	return c
}

// SetDOIs replaces the current ExpectedDOI slice with the given DOIs
func (c *ExpectedCollection) SetDOIs(dois ...collections.DOI) *ExpectedCollection {
	var newDOIs []ExpectedDOI
	for _, doi := range dois {
		newDOIs = append(newDOIs, ExpectedDOI{DOI: doi.Value, Datasource: doi.Datasource})
	}
	c.DOIs = newDOIs
	return c
}

func (c *ExpectedCollection) WithNPennsieveDOIs(n int) *ExpectedCollection {
	var dois []collections.DOI
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

func (d ExpectedDOIs) AsDOIs() collections.DOIs {
	if len(d) == 0 {
		return nil
	}
	strs := make([]collections.DOI, len(d))
	for i, doi := range d {
		strs[i] = collections.DOI{
			Value:      doi.DOI,
			Datasource: doi.Datasource,
		}
	}
	return strs
}

func (c *ExpectedCollection) ToGetCollectionResponse(t require.TestingT, expectedUserID int64, expectedPublishStatus *collections.PublishStatus) collections.GetCollectionResponse {
	test.Helper(t)
	require.NotNil(t, c.ID, "expected collection does not have ID set")
	require.NotNil(t, c.NodeID, "expected collection does not have NodeID set")
	userIdx := slices.IndexFunc(c.Users, func(user ExpectedUser) bool {
		return user.UserID == expectedUserID
	})
	require.NotEqual(t, -1, userIdx, "given user %d has no permission for expected collection", expectedUserID)
	user := c.Users[userIdx]
	collectionBase := collections.CollectionBase{
		ID:          *c.ID,
		NodeID:      *c.NodeID,
		Name:        c.Name,
		Description: c.Description,
		Size:        len(c.DOIs),
		UserRole:    user.PermissionBit.ToRole(),
	}
	if expectedPublishStatus != nil {
		collectionBase.Publication = &collections.Publication{
			Status: expectedPublishStatus.Status,
			Type:   expectedPublishStatus.Type,
		}
	}
	return collections.GetCollectionResponse{
		CollectionBase: collectionBase,
		DOIs:           c.DOIs.AsDOIs(),
	}
}

func (c *ExpectedCollection) GetCollectionFunc(t require.TestingT, expectedPublishStatus *collections.PublishStatus) mocks.GetCollectionFunc {
	test.Helper(t)
	return func(ctx context.Context, userID int64, nodeID string) (collections.GetCollectionResponse, error) {
		require.Equal(t, *c.NodeID, nodeID, "expected NodeID is %s; got %s", *c.NodeID, nodeID)
		return c.ToGetCollectionResponse(t, userID, expectedPublishStatus), nil
	}
}

func (c *ExpectedCollection) UpdateCollectionFunc(t require.TestingT) mocks.UpdateCollectionFunc {
	return func(ctx context.Context, userID int64, collectionID int64, update collections.UpdateCollectionRequest) (collections.GetCollectionResponse, error) {
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

		var updatedDOIs []collections.DOI
		for _, doi := range c.DOIs.AsDOIs() {
			if _, deleted := toDeleteSet[doi.Value]; !deleted {
				updatedDOIs = append(updatedDOIs, doi)
			}
		}

		updatedDOIs = append(updatedDOIs, update.DOIs.Add...)

		collectionBase := collections.CollectionBase{
			NodeID:      *c.NodeID,
			ID:          *c.ID,
			Name:        updatedName,
			Description: updatedDescription,
			Size:        len(updatedDOIs),
			UserRole:    user.PermissionBit.ToRole(),
		}
		return collections.GetCollectionResponse{
			CollectionBase: collectionBase,
			DOIs:           updatedDOIs,
		}, nil
	}
}

// PublishDOICollectionRequestVerification should contain assertions to verify request fields that cannot be verified
// by reference to the ExpectedCollection
type PublishDOICollectionRequestVerification func(t require.TestingT, request service.PublishDOICollectionRequest)

// PublishCollectionFunc will overwrite fields in mockResponse with values from this ExpectedCollection
func (c *ExpectedCollection) PublishCollectionFunc(t require.TestingT, mockResponse service.PublishDOICollectionResponse, verifications ...PublishDOICollectionRequestVerification) mocks.PublishCollectionFunc {
	return func(ctx context.Context, collectionID int64, userRole role.Role, request service.PublishDOICollectionRequest) (service.PublishDOICollectionResponse, error) {
		test.Helper(t)
		require.NotNil(t, c.ID, "expected collection does not have ID set")
		require.NotNil(t, c.NodeID, "expected collection does not have nodeID set")

		require.Equal(t, *c.ID, collectionID, "requested collection id %d does not match expected collection id %d", collectionID, *c.ID)
		require.Equal(t, role.Owner, userRole, "requested user role %s does not match expected user role %s", userRole, role.Owner)

		require.Equal(t, c.Description, request.Description)
		require.Equal(t, c.DOIs.Strings(), request.DOIs)

		for _, verification := range verifications {
			verification(t, request)
		}

		mockResponse.Name = c.Name
		mockResponse.SourceCollectionID = *c.ID
		mockResponse.PublicID = *c.NodeID
		return mockResponse, nil
	}
}

// FinalizeDOICollectionPublishRequestVerification should contain assertions to verify request fields that cannot be verified
// by reference to the ExpectedCollection
type FinalizeDOICollectionPublishRequestVerification func(t require.TestingT, request service.FinalizeDOICollectionPublishRequest)

// VerifyFinalizeDOICollectionRequest checks that the request has PublishSuccess == true and other expected values
func VerifyFinalizeDOICollectionRequest(expectedPublishedID, expectedPublishedVersion int64) FinalizeDOICollectionPublishRequestVerification {
	return func(t require.TestingT, request service.FinalizeDOICollectionPublishRequest) {
		require.Equal(t, expectedPublishedID, request.PublishedDatasetID)
		require.Equal(t, expectedPublishedVersion, request.PublishedVersion)
		expectedS3Key := publishing.ManifestS3Key(expectedPublishedID)
		require.Equal(t, expectedS3Key, request.ManifestKey)

		require.True(t, request.PublishSuccess)

		// right now, only one file, the manifest itself
		require.Equal(t, 1, request.FileCount)

		// don't know these values with the given info, but they shouldn't be zero
		require.NotEmpty(t, request.ManifestVersionID)
		require.Positive(t, request.TotalSize)
	}
}

// VerifyFailedFinalizeDOICollectionRequest checks that the request has PublishSuccess == false and empty values where expected
func VerifyFailedFinalizeDOICollectionRequest(expectedPublishedID, expectedPublishedVersion int64) FinalizeDOICollectionPublishRequestVerification {
	return func(t require.TestingT, request service.FinalizeDOICollectionPublishRequest) {
		require.Equal(t, expectedPublishedID, request.PublishedDatasetID)
		require.Equal(t, expectedPublishedVersion, request.PublishedVersion)

		require.False(t, request.PublishSuccess)

		// all these values should be empty or zero if we are reporting a failed publishing attempt back to discover
		require.Empty(t, request.ManifestKey)
		require.Zero(t, request.FileCount)
		require.Empty(t, request.ManifestVersionID)
		require.Zero(t, request.TotalSize)
	}
}

func VerifyFinalizeDOICollectionRequestS3VersionID(expectedS3VersionID string) FinalizeDOICollectionPublishRequestVerification {
	return func(t require.TestingT, request service.FinalizeDOICollectionPublishRequest) {
		require.Equal(t, expectedS3VersionID, request.ManifestVersionID)
	}
}

// VerifyFinalizeDOICollectionRequestTotalSize takes a function rather than int64, since we will have to capture this value in a closure
// since we won't know the correct value when this function is called during mock setup.
func VerifyFinalizeDOICollectionRequestTotalSize(expectedTotalSize func() int64) FinalizeDOICollectionPublishRequestVerification {
	return func(t require.TestingT, request service.FinalizeDOICollectionPublishRequest) {
		require.Equal(t, expectedTotalSize(), request.TotalSize)
	}
}

func (c *ExpectedCollection) FinalizeCollectionPublishFunc(t require.TestingT, mockResponse service.FinalizeDOICollectionPublishResponse, verifications ...FinalizeDOICollectionPublishRequestVerification) mocks.FinalizeCollectionPublishFunc {
	return func(ctx context.Context, collectionID int64, collectionNodeID string, userRole role.Role, request service.FinalizeDOICollectionPublishRequest) (service.FinalizeDOICollectionPublishResponse, error) {
		test.Helper(t)
		require.NotNil(t, c.ID, "expected collection does not have ID set")
		require.NotNil(t, c.NodeID, "expected collection does not have nodeID set")

		require.Equal(t, *c.ID, collectionID, "requested collection id %d does not match expected collection id %d", collectionID, *c.ID)
		require.Equal(t, *c.NodeID, collectionNodeID, "requested collection node id %s does not match expected collection node id %s", collectionNodeID, *c.NodeID)
		require.Equal(t, role.Owner, userRole, "requested user role %s does not match expected user role %s", userRole, role.Owner)

		for _, verification := range verifications {
			verification(t, request)
		}

		return mockResponse, nil
	}
}

func (c *ExpectedCollection) DatasetServiceRole(expectedRole role.Role) jwtdiscover.ServiceRole {
	return jwtdiscover.ServiceRole{
		Type:   jwtdiscover.DatasetServiceRoleType,
		Id:     strconv.FormatInt(*c.ID, 10),
		NodeId: *c.NodeID,
		Role:   strings.ToLower(expectedRole.String()),
	}
}

func (c *ExpectedCollection) StartPublishFunc(t require.TestingT, expectedUserID int64, expectedType publishing.Type) mocks.StartPublishFunc {
	return func(_ context.Context, collectionID int64, userID int64, publishingType publishing.Type) error {
		require.NotNil(t, c.ID, "expected collection does not have ID set")
		require.Equal(t, *c.ID, collectionID)
		require.Equal(t, expectedUserID, userID)
		require.Equal(t, expectedType, publishingType)
		return nil
	}
}

func (c *ExpectedCollection) FinishPublishFunc(t require.TestingT, expectedStatus publishing.Status) mocks.FinishPublishFunc {
	return func(_ context.Context, collectionID int64, publishingStatus publishing.Status, strict bool) error {
		require.NotNil(t, c.ID, "expected collection does not have ID set")
		require.Equal(t, *c.ID, collectionID)
		require.Equal(t, expectedStatus, publishingStatus)
		return nil
	}
}

func VerifyPublishingUser(expectedUser userstest.User) PublishDOICollectionRequestVerification {
	return func(t require.TestingT, request service.PublishDOICollectionRequest) {
		test.Helper(t)
		assert.Equal(t, expectedUser.GetID(), request.OwnerID)
		assert.Equal(t, expectedUser.GetNodeID(), request.OwnerNodeID)
		assert.Equal(t, expectedUser.GetFirstName(), request.OwnerFirstName)
		assert.Equal(t, expectedUser.GetLastName(), request.OwnerLastName)
		assert.Equal(t, expectedUser.GetORCIDOrEmpty(), request.OwnerORCID)
	}
}

func VerifyInternalContributors(expectedContributors ...service.InternalContributor) PublishDOICollectionRequestVerification {
	return func(t require.TestingT, request service.PublishDOICollectionRequest) {
		test.Helper(t)
		assert.Equal(t, expectedContributors, request.Contributors)
	}
}

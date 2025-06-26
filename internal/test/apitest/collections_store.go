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
	"time"
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

type ExpectedPublishStatus struct {
	CollectionID *int64
	// PreCondition is an optional status that already exists prior to
	// the test
	PreCondition *publishing.PublishStatus
	// ExpectedStatus and other Expected* fields are what we expect
	// the status fields to be after the test
	ExpectedStatus publishing.Status
	ExpectedType   publishing.Type
	ExpectedUserID int64
}

func NewExpectedPublishStatus(pubStatus publishing.Status, pubType publishing.Type, pubUser int64) *ExpectedPublishStatus {
	return &ExpectedPublishStatus{
		ExpectedStatus: pubStatus,
		ExpectedType:   pubType,
		ExpectedUserID: pubUser,
	}
}

// NewExpectedInProgressPublishStatus returns an ExpectedPublishStatus that makes sense if we are
// expecting an InProgress status at the end of a test. This status should only be temporary, so if we expect this
// status at the end of the test, there must be an existing pre-condition of that status already
// existing.
func NewExpectedInProgressPublishStatus(pubUser int64) *ExpectedPublishStatus {
	return NewExpectedPublishStatus(publishing.InProgressStatus, publishing.PublicationType, pubUser).WithExistingInProgressPublishStatus(pubUser)
}

func (s *ExpectedPublishStatus) WithCollectionID(collectionID int64) *ExpectedPublishStatus {
	s.CollectionID = &collectionID
	if s.PreCondition != nil {
		s.PreCondition.CollectionID = collectionID
	}
	return s
}

func (s *ExpectedPublishStatus) WithExistingInProgressPublishStatus(userID int64) *ExpectedPublishStatus {
	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	s.PreCondition = &publishing.PublishStatus{
		Status:    publishing.InProgressStatus,
		Type:      publishing.PublicationType,
		StartedAt: startedAt,
		UserID:    &userID,
	}
	if s.CollectionID != nil {
		s.PreCondition.CollectionID = *s.CollectionID
	}
	return s
}

func (s *ExpectedPublishStatus) WithExistingCompletedPublishStatus(userID int64) *ExpectedPublishStatus {
	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	finishedAt := startedAt.Add(time.Minute)
	s.PreCondition = &publishing.PublishStatus{
		Status:     publishing.CompletedStatus,
		Type:       publishing.PublicationType,
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
		UserID:     &userID,
	}
	if s.CollectionID != nil {
		s.PreCondition.CollectionID = *s.CollectionID
	}
	return s
}

func (s *ExpectedPublishStatus) WithExistingFailedPublishStatus(userID int64) *ExpectedPublishStatus {
	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	finishedAt := startedAt.Add(time.Minute)
	s.PreCondition = &publishing.PublishStatus{
		Status:     publishing.FailedStatus,
		Type:       publishing.PublicationType,
		StartedAt:  startedAt,
		FinishedAt: &finishedAt,
		UserID:     &userID,
	}
	if s.CollectionID != nil {
		s.PreCondition.CollectionID = *s.CollectionID
	}
	return s
}

func (c *ExpectedCollection) ToGetCollectionResponse(t require.TestingT, expectedUserID int64) collections.GetCollectionResponse {
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
	return collections.GetCollectionResponse{
		CollectionBase: collectionBase,
		DOIs:           c.DOIs.AsDOIs(),
	}
}

func (c *ExpectedCollection) GetCollectionFunc(t require.TestingT) mocks.GetCollectionFunc {
	test.Helper(t)
	return func(ctx context.Context, userID int64, nodeID string) (collections.GetCollectionResponse, error) {
		require.Equal(t, *c.NodeID, nodeID, "expected NodeID is %s; got %s", *c.NodeID, nodeID)
		return c.ToGetCollectionResponse(t, userID), nil
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

func (c *ExpectedCollection) PublishCollectionFunc(t require.TestingT, expectedPublishedID, expectedPublishedVersion int64, expectedPublishStatus string, verifications ...PublishDOICollectionRequestVerification) mocks.PublishCollectionFunc {
	return func(collectionID int64, userRole role.Role, request service.PublishDOICollectionRequest) (service.PublishDOICollectionResponse, error) {
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

		return service.PublishDOICollectionResponse{
			Name:               c.Name,
			SourceCollectionID: *c.ID,
			PublishedDatasetID: expectedPublishedID,
			PublishedVersion:   expectedPublishedVersion,
			Status:             expectedPublishStatus,
			PublicID:           *c.NodeID,
		}, nil
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
		if expectedUser.GetORCIDAuthorization() == nil {
			assert.Empty(t, request.OwnerORCID)
		} else {
			assert.Equal(t, expectedUser.GetORCIDAuthorization().ORCID, request.OwnerORCID)
		}
	}
}

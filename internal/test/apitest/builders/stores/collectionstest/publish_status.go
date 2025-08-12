package collectionstest

import (
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"time"
)

type PublishStatusBuilder struct {
	s *collections.PublishStatus
}

func NewPublishStatusBuilder() *PublishStatusBuilder {
	return &PublishStatusBuilder{s: &collections.PublishStatus{}}
}

func (b *PublishStatusBuilder) WithCollectionID(collectionID int64) *PublishStatusBuilder {
	b.s.CollectionID = collectionID
	return b
}

func (b *PublishStatusBuilder) WithStatus(status publishing.Status) *PublishStatusBuilder {
	b.s.Status = status
	return b
}

func (b *PublishStatusBuilder) WithType(pubType publishing.Type) *PublishStatusBuilder {
	b.s.Type = pubType
	return b
}

func (b *PublishStatusBuilder) WithStartedAt(startedAt time.Time) *PublishStatusBuilder {
	b.s.StartedAt = startedAt
	return b
}
func (b *PublishStatusBuilder) WithFinishedAt(finishedAt *time.Time) *PublishStatusBuilder {
	b.s.FinishedAt = finishedAt
	return b
}

func (b *PublishStatusBuilder) WithUserID(userID *int64) *PublishStatusBuilder {
	b.s.UserID = userID
	return b
}

func (b *PublishStatusBuilder) Build() collections.PublishStatus {
	return *b.s
}

// NewInProgressPublishStatus returns an InProgress collections.PublishStatus with a
// StartedAt value in the past and a nil FinishedAt value
func NewInProgressPublishStatus(collectionID, userID int64) collections.PublishStatus {
	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	b := NewPublishStatusBuilder().
		WithCollectionID(collectionID).
		WithUserID(&userID).
		WithStatus(publishing.InProgressStatus).
		WithType(publishing.PublicationType).
		WithStartedAt(startedAt)

	return b.Build()
}

// NewCompletedPublishStatus returns a Completed collections.PublishStatus with a
// StartedAt value in the past and a non-nil FinishedAt value later than StartedAt
func NewCompletedPublishStatus(collectionID, userID int64) collections.PublishStatus {
	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	finishedAt := startedAt.Add(time.Minute)
	b := NewPublishStatusBuilder().
		WithCollectionID(collectionID).
		WithUserID(&userID).
		WithStatus(publishing.CompletedStatus).
		WithType(publishing.PublicationType).
		WithStartedAt(startedAt).
		WithFinishedAt(&finishedAt)

	return b.Build()
}

// NewFailedPublishStatus returns a Failed collections.PublishStatus with a
// StartedAt value in the past and a non-nil FinishedAt value later than StartedAt
func NewFailedPublishStatus(collectionID, userID int64) collections.PublishStatus {
	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	finishedAt := startedAt.Add(time.Minute)
	b := NewPublishStatusBuilder().
		WithCollectionID(collectionID).
		WithUserID(&userID).
		WithStatus(publishing.FailedStatus).
		WithType(publishing.PublicationType).
		WithStartedAt(startedAt).
		WithFinishedAt(&finishedAt)

	return b.Build()
}

// NewExpectedInProgressPublishStatus returns an InProgress collections.PublishStatus with a
// zero StartedAt value and a nil FinishedAt value
func NewExpectedInProgressPublishStatus(collectionID, userID int64) collections.PublishStatus {
	b := NewPublishStatusBuilder().
		WithCollectionID(collectionID).
		WithUserID(&userID).
		WithStatus(publishing.InProgressStatus).
		WithType(publishing.PublicationType)

	return b.Build()
}

// NewExpectedCompletedPublishStatus returns a Completed collections.PublishStatus with a
// zero StartedAt value and a nil FinishedAt value
func NewExpectedCompletedPublishStatus(collectionID, userID int64) collections.PublishStatus {
	b := NewPublishStatusBuilder().
		WithCollectionID(collectionID).
		WithUserID(&userID).
		WithStatus(publishing.CompletedStatus).
		WithType(publishing.PublicationType)

	return b.Build()
}

// NewExpectedFailedPublishStatus returns an Failed collections.PublishStatus with a
// zero StartedAt value and a nil FinishedAt value
func NewExpectedFailedPublishStatus(collectionID, userID int64) collections.PublishStatus {
	b := NewPublishStatusBuilder().
		WithCollectionID(collectionID).
		WithUserID(&userID).
		WithStatus(publishing.FailedStatus).
		WithType(publishing.PublicationType)

	return b.Build()
}

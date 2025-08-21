package collectionstest

import (
	"github.com/pennsieve/collections-service/internal/api/publishing"
	"github.com/pennsieve/collections-service/internal/api/store/collections"
	"time"
)

type PublishStatusBuilder struct {
	s *collections.PublishStatus
}

func NewPublishStatusBuilder(collectionID int64, pubType publishing.Type, pubStatus publishing.Status) *PublishStatusBuilder {
	return &PublishStatusBuilder{
		s: &collections.PublishStatus{
			CollectionID: collectionID,
			Type:         pubType,
			Status:       pubStatus,
		},
	}
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

func (b *PublishStatusBuilder) WithInProgressStartedAt() *PublishStatusBuilder {
	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	return b.WithStatus(publishing.InProgressStatus).WithStartedAt(startedAt)
}

func (b *PublishStatusBuilder) WithStartedAndFinishedAt() *PublishStatusBuilder {
	startedAt := time.Now().UTC().AddDate(0, -1, 2)
	finishedAt := startedAt.Add(time.Minute)
	return b.WithStartedAt(startedAt).WithFinishedAt(&finishedAt)
}

func (b *PublishStatusBuilder) Build() collections.PublishStatus {
	return *b.s
}

// NewInProgressPublishStatusBuilder returns an InProgress PublishStatusBuilder with a
// StartedAt value in the past and a nil FinishedAt value
func NewInProgressPublishStatusBuilder(collectionID int64, pubType publishing.Type) *PublishStatusBuilder {
	return NewPublishStatusBuilder(collectionID, pubType, publishing.InProgressStatus).
		WithInProgressStartedAt()
}

// NewTerminalPublishStatusBuilder returns a PublishStatusBuilder with a
// StartedAt value in the past and a non-nil FinishedAt value later than StartedAt
func NewTerminalPublishStatusBuilder(collectionID int64, pubType publishing.Type, pubStatus publishing.Status) *PublishStatusBuilder {
	return NewPublishStatusBuilder(collectionID, pubType, pubStatus).WithStartedAndFinishedAt()
}

// NewInProgressPublishStatus returns an InProgress Publication collections.PublishStatus with a
// StartedAt value in the past and a nil FinishedAt value
func NewInProgressPublishStatus(collectionID, userID int64) collections.PublishStatus {
	return NewInProgressPublishStatusBuilder(collectionID, publishing.PublicationType).WithUserID(&userID).Build()
}

// NewCompletedPublishStatus returns a Completed Publication collections.PublishStatus with a
// StartedAt value in the past and a non-nil FinishedAt value later than StartedAt
func NewCompletedPublishStatus(collectionID, userID int64) collections.PublishStatus {
	return NewTerminalPublishStatusBuilder(collectionID, publishing.PublicationType, publishing.CompletedStatus).WithUserID(&userID).Build()
}

// NewFailedPublishStatus returns a Failed Publication collections.PublishStatus with a
// StartedAt value in the past and a non-nil FinishedAt value later than StartedAt
func NewFailedPublishStatus(collectionID, userID int64) collections.PublishStatus {
	return NewTerminalPublishStatusBuilder(collectionID, publishing.PublicationType, publishing.FailedStatus).WithUserID(&userID).Build()
}

// NewExpectedInProgressPublishStatus returns an InProgress Publication collections.PublishStatus with a
// zero StartedAt value and a nil FinishedAt value
func NewExpectedInProgressPublishStatus(collectionID, userID int64) collections.PublishStatus {
	b := NewPublishStatusBuilder(collectionID, publishing.PublicationType, publishing.InProgressStatus).
		WithUserID(&userID)
	return b.Build()
}

// NewExpectedCompletedPublishStatus returns a Completed Publication collections.PublishStatus with a
// zero StartedAt value and a nil FinishedAt value
func NewExpectedCompletedPublishStatus(collectionID, userID int64) collections.PublishStatus {
	b := NewPublishStatusBuilder(collectionID, publishing.PublicationType, publishing.CompletedStatus).
		WithUserID(&userID)
	return b.Build()
}

// NewExpectedFailedPublishStatus returns a Failed Publication collections.PublishStatus with a
// zero StartedAt value and a nil FinishedAt value
func NewExpectedFailedPublishStatus(collectionID, userID int64) collections.PublishStatus {
	b := NewPublishStatusBuilder(collectionID, publishing.PublicationType, publishing.FailedStatus).
		WithUserID(&userID)
	return b.Build()
}

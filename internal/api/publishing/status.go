package publishing

import (
	"github.com/pennsieve/collections-service/internal/api/dto"
	"time"
)

type PublishStatus struct {
	CollectionID int64      `db:"collection_id"`
	Status       Status     `db:"status"`
	Type         Type       `db:"type"`
	StartedAt    time.Time  `db:"started_at"`
	FinishedAt   *time.Time `db:"finished_at"`
	// UserID is the user that started the publish. Should only be
	// nil if the user is deleted.
	UserID *int64 `db:"user_id"`
}
type Status string

const InProgressStatus Status = "InProgress"
const CompletedStatus Status = "Completed"
const FailedStatus Status = "Failed"

type Type string

const PublicationType Type = "Publication"
const RevisionType Type = "Revision"
const RemovalType Type = "Removal"

func FromDiscoverPublishStatus(discoverStatus dto.PublishStatus) Status {
	switch discoverStatus {
	case dto.PublishSucceeded, dto.Unpublished:
		return CompletedStatus
	case dto.PublishFailed:
		return FailedStatus
		// Don't think we should see any other PublishStatus than those listed in the first two cases.
		// Fail if we get an unexpected PublishStatus
	default:
		return FailedStatus
	}
}

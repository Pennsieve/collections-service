package publishing

import (
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

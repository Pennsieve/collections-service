package publishing

type Status string

// DraftStatus means there has never been an attempt to publish the collection
const DraftStatus Status = "Draft"

// InProgressStatus means there is a publication process in progress. Should be a temporary state
const InProgressStatus Status = "InProgress"

// CompletedStatus means that a publication process ran and finished without error
const CompletedStatus Status = "Completed"

// FailedStatus means that a publication process ran and finished with an error
const FailedStatus Status = "Failed"

type Type string

const PublicationType Type = "Publication"
const RevisionType Type = "Revision"
const RemovalType Type = "Removal"

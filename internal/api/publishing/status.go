package publishing

type Status string

const InProgressStatus Status = "InProgress"
const CompletedStatus Status = "Completed"
const FailedStatus Status = "Failed"

type Type string

const PublicationType Type = "Publication"
const RevisionType Type = "Revision"
const RemovalType Type = "Removal"

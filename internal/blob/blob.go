package blob

type BlobKind string

const (
	KindFeature       BlobKind = "Feature"
	KindBugFix        BlobKind = "BugFix"
	KindMigration     BlobKind = "Migration"
	KindInvestigation BlobKind = "Investigation"
	KindRefactor      BlobKind = "Refactor"
	KindArchitecture  BlobKind = "Architecture"
	KindReview        BlobKind = "Review"
	KindIncident      BlobKind = "Incident"
	KindCheckpoint    BlobKind = "Checkpoint" // lore internal: knowledge base state snapshot
)

type Blob struct {
	ID                string
	Kind              BlobKind
	Title             string
	Summary           string
	Recap             string
	UserIntent        string
	InferredReasoning string
	Tags              []string
	TrustLevel        int
	AISource          string
	StartedAt         int64
	EndedAt           int64
	CommitStart       string
	CommitEnd         string
	PrimaryNodeID     string
	CreatedAt         int64
}

package store

type BlobFile struct {
	BlobID string
	Path   string
	Role   string // "written" | "deleted" | "renamed_from" | "renamed_to"
}

type BlobCommand struct {
	BlobID  string
	Command string
	TS      int64
}

type GraphNode struct {
	ID    string
	Kind  string // "Topic" | "Blob" | "File" | "Commit" | "Concept"
	Label string
	Ref   string
}

type GraphEdge struct {
	ID       string
	FromID   string
	ToID     string
	Relation string
	Weight   int
}

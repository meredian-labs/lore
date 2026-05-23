package store

import "github.com/nishchay/lore/internal/blob"

// BlobFile and BlobCommand are defined in internal/blob to avoid circular
// imports. These aliases let existing store code use the unqualified names.
type BlobFile = blob.BlobFile
type BlobCommand = blob.BlobCommand

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

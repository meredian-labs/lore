package blob

// BlobFile is a file associated with a blob.
// Defined here (not in store) so blob package can reference it without
// creating circular imports with internal/store.
type BlobFile struct {
	BlobID string
	Path   string
	Role   string // "written" | "deleted" | "renamed_from" | "renamed_to"
}

// BlobCommand is a command recorded during a blob's work session.
type BlobCommand struct {
	BlobID  string
	Command string
	TS      int64
}

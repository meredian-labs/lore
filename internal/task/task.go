package task

type Task struct {
	ID            string
	Kind          TaskKind
	Path          string
	Detail        string
	Source        string
	TrustLevel    int
	TS            int64
	Extracted     bool
	ExtractedInto string
}

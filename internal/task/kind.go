package task

type TaskKind string

const (
	KindFileWrite     TaskKind = "FileWrite"
	KindFileDelete    TaskKind = "FileDelete"
	KindFileRename    TaskKind = "FileRename"
	KindFileRead      TaskKind = "FileRead"
	KindCommand       TaskKind = "Command"
	KindCommitCreated TaskKind = "CommitCreated"
	KindBranchSwitch  TaskKind = "BranchSwitch"
	KindMergeEvent    TaskKind = "MergeEvent"
	KindSearchQuery   TaskKind = "SearchQuery"
	KindAgentAction   TaskKind = "AgentAction"
	KindNote          TaskKind = "Note"
	KindAgentRecap    TaskKind = "AgentRecap"
)

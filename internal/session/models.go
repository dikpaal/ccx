package session

import "time"

type TodoItem struct {
	Content string `json:"content"`
	Status  string `json:"status"` // pending, in_progress, completed
}

type Session struct {
	ID          string
	ShortID     string
	FilePath    string
	ProjectPath string
	ProjectName string
	GitBranch   string
	ModTime     time.Time
	MsgCount    int
	FirstPrompt string
	Created     time.Time
	IsWorktree  bool
	IsLive      bool
	HasMemory   bool
	Todos       []TodoItem
}

type Entry struct {
	Type      string
	Timestamp time.Time
	IsMeta    bool
	Role      string
	Content   []ContentBlock
	Model     string
	UUID      string
	ParentID  string
	AgentID   string
	RawJSON   string
}

type ContentBlock struct {
	Type      string
	Text      string
	ToolName  string
	ToolInput string
	IsError   bool
}

type Subagent struct {
	ID          string
	ShortID     string
	FilePath    string
	MsgCount    int
	FirstPrompt string
	Timestamp   time.Time
	AgentType   string
}

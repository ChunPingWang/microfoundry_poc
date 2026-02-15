package agents

import "context"

// AgentRole identifies which agent in the pipeline.
type AgentRole string

const (
	RoleAnalyzer     AgentRole = "analyzer"
	RoleDataEngineer AgentRole = "data-engineer"
	RoleDesigner     AgentRole = "designer"
	RoleDeveloper    AgentRole = "developer"
	RoleReviewer     AgentRole = "reviewer"
)

// PipelineContext carries state through the agent pipeline.
type PipelineContext struct {
	// Epic information (provided by user)
	EpicTitle       string
	EpicDescription string

	// GitHub coordinates
	RepoOwner string
	RepoName  string

	// Pipeline state (enriched by agents)
	BranchName  string
	IssueNumber int
	PRNumber    int

	// Working directory for code operations
	WorkDir string

	// Reference codebase paths (CF CLI, cf-deployment, etc.)
	RefCodebasePaths []string

	// Arbitrary metadata passed between agents
	Metadata map[string]string
}

// AgentResult represents what an agent produces.
type AgentResult struct {
	Success     bool
	Summary     string
	Artifacts   []Artifact
	NextContext *PipelineContext
	Error       error
}

// Artifact represents a produced output (issue, comment, PR, file, etc.).
type Artifact struct {
	Type    string // "issue", "comment", "pr", "file", "branch"
	Content string
	URL     string
}

// Agent is the interface all pipeline agents implement.
type Agent interface {
	// Role returns the agent's role identifier.
	Role() AgentRole
	// Run executes the agent's work given the pipeline context.
	Run(ctx context.Context, pctx *PipelineContext) (*AgentResult, error)
}

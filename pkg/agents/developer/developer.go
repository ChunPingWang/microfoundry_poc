package developer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/younjinjeong/microfoundry/pkg/agents"
	"github.com/younjinjeong/microfoundry/pkg/claude"
	"github.com/younjinjeong/microfoundry/pkg/github"
)

// Agent is the Developer Agent that writes code, builds, and creates PRs.
type Agent struct {
	claude *claude.Client
	gh     *github.Client
}

// New creates a new Developer Agent.
func New(claudeClient *claude.Client, ghClient *github.Client) *Agent {
	return &Agent{
		claude: claudeClient,
		gh:     ghClient,
	}
}

func (a *Agent) Role() agents.AgentRole {
	return agents.RoleDeveloper
}

func (a *Agent) Run(ctx context.Context, pctx *agents.PipelineContext) (*agents.AgentResult, error) {
	// 1. Read the full issue with all comments
	issue, err := a.gh.ReadIssue(pctx.IssueNumber)
	if err != nil {
		return nil, fmt.Errorf("reading issue #%d: %w", pctx.IssueNumber, err)
	}

	var commentBodies []string
	for _, c := range issue.Comments {
		commentBodies = append(commentBodies, c.Body)
	}

	// 2. Checkout the epic branch
	if pctx.WorkDir != "" && pctx.BranchName != "" {
		if err := a.gh.CheckoutBranch(pctx.WorkDir, pctx.BranchName); err != nil {
			return nil, fmt.Errorf("checking out branch %s: %w", pctx.BranchName, err)
		}
	}

	// 3. Use Claude tool-calling loop to write code
	prompt := buildDeveloperPrompt(issue.Body, commentBodies, pctx.BranchName)
	tools := claude.DeveloperTools()

	conv := claude.NewConversation(a.claude, systemPrompt, tools)
	summary, err := conv.SendMessage(ctx, prompt, func(toolName string, input map[string]interface{}) (string, error) {
		return a.executeTool(pctx.WorkDir, toolName, input)
	})
	if err != nil {
		return nil, fmt.Errorf("developer coding session failed: %w", err)
	}

	// 4. Commit and push changes
	if pctx.WorkDir != "" {
		commitMsg := fmt.Sprintf("feat: implement %s\n\nEpic #%d", pctx.EpicTitle, pctx.IssueNumber)
		if err := a.gh.CommitAndPush(pctx.WorkDir, commitMsg); err != nil {
			return nil, fmt.Errorf("committing changes: %w", err)
		}
	}

	// 5. Create PR
	prBody := formatPRBody(issue, pctx, summary)
	prNum, prURL, err := a.gh.CreatePR(
		fmt.Sprintf("feat: %s", pctx.EpicTitle),
		prBody,
		pctx.BranchName,
		"main",
	)
	if err != nil {
		return nil, fmt.Errorf("creating PR: %w", err)
	}

	// 6. Update context
	result := *pctx
	result.PRNumber = prNum
	if result.Metadata == nil {
		result.Metadata = make(map[string]string)
	}
	result.Metadata["pr_url"] = prURL

	return &agents.AgentResult{
		Success: true,
		Summary: fmt.Sprintf("Created PR #%d with implementation", prNum),
		Artifacts: []agents.Artifact{
			{Type: "pr", Content: prBody, URL: prURL},
		},
		NextContext: &result,
	}, nil
}

// executeTool handles tool calls from Claude during the coding session.
func (a *Agent) executeTool(workDir, toolName string, input map[string]interface{}) (string, error) {
	switch toolName {
	case "read_file":
		path := getStr(input, "path")
		fullPath := resolvePath(workDir, path)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return "", err
		}
		return string(data), nil

	case "write_file":
		path := getStr(input, "path")
		content := getStr(input, "content")
		fullPath := resolvePath(workDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return "", err
		}
		return fmt.Sprintf("Wrote %d bytes to %s", len(content), path), nil

	case "list_files":
		path := getStr(input, "path")
		fullPath := resolvePath(workDir, path)
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return "", err
		}
		var names []string
		for _, e := range entries {
			suffix := ""
			if e.IsDir() {
				suffix = "/"
			}
			names = append(names, e.Name()+suffix)
		}
		return strings.Join(names, "\n"), nil

	case "run_command":
		command := getStr(input, "command")
		dir := getStr(input, "working_dir")
		if dir == "" {
			dir = workDir
		} else {
			dir = resolvePath(workDir, dir)
		}
		cmd := exec.CommandContext(context.Background(), "bash", "-c", command)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Error: %s\nOutput:\n%s", err, string(out)), nil
		}
		return string(out), nil

	case "search_files":
		pattern := getStr(input, "pattern")
		path := getStr(input, "path")
		glob := getStr(input, "file_glob")
		fullPath := resolvePath(workDir, path)
		args := []string{"-rn", pattern, fullPath}
		if glob != "" {
			args = append(args, "--include", glob)
		}
		cmd := exec.Command("grep", args...)
		out, _ := cmd.CombinedOutput()
		return string(out), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

// Unused but required for tool definition — suppress linter.
var _ = anthropic.ToolUnionParam{}

func resolvePath(workDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(workDir, path)
}

func getStr(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

func formatPRBody(issue *github.Issue, pctx *agents.PipelineContext, summary string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Summary\n\nImplementation for Epic #%d: %s\n\n", pctx.IssueNumber, pctx.EpicTitle))
	sb.WriteString("## Changes\n\n")
	sb.WriteString(summary)
	sb.WriteString("\n\n## Test Plan\n\n")
	sb.WriteString("- [ ] `go build ./...` passes\n")
	sb.WriteString("- [ ] `go test ./...` passes\n")
	sb.WriteString("- [ ] Docker image builds successfully\n")
	sb.WriteString("- [ ] Deployed to local k8s and smoke tested\n")
	sb.WriteString(fmt.Sprintf("\n\nCloses #%d\n", pctx.IssueNumber))
	sb.WriteString("\n---\n*Generated by MicroFoundry Developer Agent*\n")
	return sb.String()
}

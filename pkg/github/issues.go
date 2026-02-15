package github

import (
	"fmt"
	"strconv"
	"strings"
)

// Issue represents a GitHub issue with its comments.
type Issue struct {
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	Body     string    `json:"body"`
	State    string    `json:"state"`
	URL      string    `json:"url"`
	Labels   []Label   `json:"labels"`
	Comments []Comment `json:"comments"`
}

// Label represents a GitHub issue label.
type Label struct {
	Name string `json:"name"`
}

// Comment represents a GitHub issue or PR comment.
type Comment struct {
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

// CreateIssue creates a new GitHub issue and returns the issue number and URL.
func (c *Client) CreateIssue(title, body string, labels []string) (int, string, error) {
	args := []string{
		"issue", "create",
		"--repo", c.repoSlug(),
		"--title", title,
		"--body", body,
	}
	for _, l := range labels {
		args = append(args, "--label", l)
	}

	out, err := c.ghExec(args...)
	if err != nil {
		return 0, "", fmt.Errorf("creating issue: %w", err)
	}

	url := strings.TrimSpace(string(out))
	// Extract issue number from URL (e.g., https://github.com/owner/repo/issues/42)
	parts := strings.Split(url, "/")
	if len(parts) < 1 {
		return 0, url, fmt.Errorf("could not parse issue number from URL: %s", url)
	}
	num, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, url, fmt.Errorf("could not parse issue number from URL %s: %w", url, err)
	}

	return num, url, nil
}

// ReadIssue reads a GitHub issue including all comments.
func (c *Client) ReadIssue(issueNumber int) (*Issue, error) {
	out, err := c.ghExec(
		"issue", "view",
		fmt.Sprintf("%d", issueNumber),
		"--repo", c.repoSlug(),
		"--json", "number,title,body,state,url,labels,comments",
	)
	if err != nil {
		return nil, fmt.Errorf("reading issue #%d: %w", issueNumber, err)
	}

	issue, err := parseJSON[Issue](out)
	if err != nil {
		return nil, fmt.Errorf("parsing issue JSON: %w", err)
	}
	return &issue, nil
}

// AddIssueComment adds a comment to a GitHub issue.
func (c *Client) AddIssueComment(issueNumber int, body string) error {
	_, err := c.ghExec(
		"issue", "comment",
		fmt.Sprintf("%d", issueNumber),
		"--repo", c.repoSlug(),
		"--body", body,
	)
	if err != nil {
		return fmt.Errorf("commenting on issue #%d: %w", issueNumber, err)
	}
	return nil
}

// CreateLabel creates a label on the repo (idempotent — ignores "already exists" errors).
func (c *Client) CreateLabel(name, color, description string) error {
	_, err := c.ghExec(
		"label", "create", name,
		"--repo", c.repoSlug(),
		"--color", color,
		"--description", description,
		"--force",
	)
	return err
}

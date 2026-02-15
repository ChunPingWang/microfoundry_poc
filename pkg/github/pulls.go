package github

import (
	"fmt"
	"strconv"
	"strings"
)

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	URL    string `json:"url"`
}

// CreatePR creates a new pull request and returns the PR number and URL.
func (c *Client) CreatePR(title, body, head, base string) (int, string, error) {
	args := []string{
		"pr", "create",
		"--repo", c.repoSlug(),
		"--title", title,
		"--body", body,
		"--head", head,
		"--base", base,
	}

	out, err := c.ghExec(args...)
	if err != nil {
		return 0, "", fmt.Errorf("creating PR: %w", err)
	}

	url := strings.TrimSpace(string(out))
	parts := strings.Split(url, "/")
	if len(parts) < 1 {
		return 0, url, fmt.Errorf("could not parse PR number from URL: %s", url)
	}
	num, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, url, fmt.Errorf("could not parse PR number from URL %s: %w", url, err)
	}

	return num, url, nil
}

// AddPRComment adds a comment on a pull request.
func (c *Client) AddPRComment(prNumber int, body string) error {
	_, err := c.ghExec(
		"pr", "comment",
		fmt.Sprintf("%d", prNumber),
		"--repo", c.repoSlug(),
		"--body", body,
	)
	if err != nil {
		return fmt.Errorf("commenting on PR #%d: %w", prNumber, err)
	}
	return nil
}

// AddPRReview adds a formal review on a pull request.
// event must be one of: "APPROVE", "REQUEST_CHANGES", "COMMENT"
func (c *Client) AddPRReview(prNumber int, event, body string) error {
	args := []string{
		"pr", "review",
		fmt.Sprintf("%d", prNumber),
		"--repo", c.repoSlug(),
		"--body", body,
	}

	switch strings.ToUpper(event) {
	case "APPROVE":
		args = append(args, "--approve")
	case "REQUEST_CHANGES":
		args = append(args, "--request-changes")
	default:
		args = append(args, "--comment")
	}

	_, err := c.ghExec(args...)
	if err != nil {
		return fmt.Errorf("reviewing PR #%d: %w", prNumber, err)
	}
	return nil
}

// GetPRDiff returns the diff of a pull request.
func (c *Client) GetPRDiff(prNumber int) (string, error) {
	out, err := c.ghExec(
		"pr", "diff",
		fmt.Sprintf("%d", prNumber),
		"--repo", c.repoSlug(),
	)
	if err != nil {
		return "", fmt.Errorf("getting PR #%d diff: %w", prNumber, err)
	}
	return string(out), nil
}

// ReadPR reads a pull request.
func (c *Client) ReadPR(prNumber int) (*PullRequest, error) {
	out, err := c.ghExec(
		"pr", "view",
		fmt.Sprintf("%d", prNumber),
		"--repo", c.repoSlug(),
		"--json", "number,title,body,state,url",
	)
	if err != nil {
		return nil, fmt.Errorf("reading PR #%d: %w", prNumber, err)
	}

	pr, err := parseJSON[PullRequest](out)
	if err != nil {
		return nil, fmt.Errorf("parsing PR JSON: %w", err)
	}
	return &pr, nil
}

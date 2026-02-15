package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateBranch creates a new branch from the default branch and pushes it to remote.
// workDir should be the local git repository directory.
func (c *Client) CreateBranch(workDir, branchName string) error {
	// Fetch latest from remote
	if err := gitExec(workDir, "fetch", "origin"); err != nil {
		return fmt.Errorf("fetching origin: %w", err)
	}

	// Determine default branch
	defaultBranch, err := c.getDefaultBranch()
	if err != nil {
		return fmt.Errorf("getting default branch: %w", err)
	}

	// Create and checkout new branch from default branch
	if err := gitExec(workDir, "checkout", "-b", branchName, "origin/"+defaultBranch); err != nil {
		return fmt.Errorf("creating branch %s: %w", branchName, err)
	}

	// Push to remote with tracking
	if err := gitExec(workDir, "push", "-u", "origin", branchName); err != nil {
		return fmt.Errorf("pushing branch %s: %w", branchName, err)
	}

	return nil
}

// CheckoutBranch checks out an existing branch.
func (c *Client) CheckoutBranch(workDir, branchName string) error {
	if err := gitExec(workDir, "checkout", branchName); err != nil {
		return fmt.Errorf("checking out branch %s: %w", branchName, err)
	}
	return nil
}

// CommitAndPush stages all changes, commits with the given message, and pushes.
func (c *Client) CommitAndPush(workDir, message string) error {
	if err := gitExec(workDir, "add", "-A"); err != nil {
		return fmt.Errorf("staging changes: %w", err)
	}

	if err := gitExec(workDir, "commit", "-m", message); err != nil {
		return fmt.Errorf("committing: %w", err)
	}

	if err := gitExec(workDir, "push"); err != nil {
		return fmt.Errorf("pushing: %w", err)
	}

	return nil
}

// getDefaultBranch returns the default branch name for the repo.
func (c *Client) getDefaultBranch() (string, error) {
	out, err := c.ghExec(
		"repo", "view",
		c.repoSlug(),
		"--json", "defaultBranchRef",
		"--jq", ".defaultBranchRef.name",
	)
	if err != nil {
		return "main", nil // fallback
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "main", nil
	}
	return branch, nil
}

// gitExec runs a git command in the specified directory.
func gitExec(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w\noutput: %s", strings.Join(args, " "), err, string(out))
	}
	return nil
}

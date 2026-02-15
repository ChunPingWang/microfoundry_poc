package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps the gh CLI for GitHub operations.
type Client struct {
	Owner string
	Repo  string
}

// NewClient creates a new GitHub client targeting a specific repo.
func NewClient(owner, repo string) *Client {
	return &Client{Owner: owner, Repo: repo}
}

func (c *Client) repoSlug() string {
	return fmt.Sprintf("%s/%s", c.Owner, c.Repo)
}

// ghExec runs a gh CLI command and returns stdout.
func (c *Client) ghExec(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh %s failed: %w\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// ghExecInDir runs a gh CLI command in a specific directory.
func (c *Client) ghExecInDir(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh %s failed: %w\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// parseJSON is a helper to unmarshal JSON output from gh commands.
func parseJSON[T any](data []byte) (T, error) {
	var result T
	err := json.Unmarshal(data, &result)
	return result, err
}

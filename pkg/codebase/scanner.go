package codebase

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scanner analyzes reference codebases to build context for AI prompts.
type Scanner struct {
	MaxContextSize int // approximate token limit
}

// NewScanner creates a scanner with the given token budget.
func NewScanner(maxContextSize int) *Scanner {
	if maxContextSize == 0 {
		maxContextSize = 30000 // ~30k tokens of context
	}
	return &Scanner{MaxContextSize: maxContextSize}
}

// BuildContext scans reference codebases and produces a structured summary.
func (s *Scanner) BuildContext(paths []string) (string, error) {
	var ctx strings.Builder

	for _, basePath := range paths {
		info, err := os.Stat(basePath)
		if err != nil {
			continue // skip missing paths
		}
		if !info.IsDir() {
			continue
		}

		repoName := filepath.Base(basePath)
		ctx.WriteString(fmt.Sprintf("\n## Reference: %s\n\n", repoName))

		switch repoName {
		case "cli":
			s.scanCLIRepo(&ctx, basePath)
		case "cf-deployment":
			s.scanDeploymentRepo(&ctx, basePath)
		default:
			s.scanGenericRepo(&ctx, basePath)
		}
	}

	result := ctx.String()
	// Rough truncation if too large (4 chars ≈ 1 token)
	maxChars := s.MaxContextSize * 4
	if len(result) > maxChars {
		result = result[:maxChars] + "\n\n... [truncated for token budget]"
	}

	return result, nil
}

func (s *Scanner) scanCLIRepo(ctx *strings.Builder, basePath string) {
	// Read go.mod for dependency overview
	s.appendFile(ctx, basePath, "go.mod", "### Dependencies (go.mod)")

	// Read command list for CF feature surface
	s.appendFile(ctx, basePath, "command/common/command_list_v7.go", "### CF CLI Commands")

	// Read key resource types
	s.appendFileGlob(ctx, basePath, "resources/*_resource.go", "### Resource Types", 5)

	// Read the actor interface (feature surface)
	s.appendFile(ctx, basePath, "command/v7/actor.go", "### Actor Interface (Business Logic)")

	// Read service binding implementation
	s.appendFile(ctx, basePath, "actor/v7action/service_app_binding.go", "### Service Binding Implementation")

	// Read route implementation
	s.appendFile(ctx, basePath, "actor/v7action/route.go", "### Route Implementation")
}

func (s *Scanner) scanDeploymentRepo(ctx *strings.Builder, basePath string) {
	// Read the main deployment manifest (first 200 lines for overview)
	s.appendFileHead(ctx, basePath, "cf-deployment.yml", "### CF Deployment Manifest (excerpt)", 200)

	// Read operations README for available customizations
	s.appendFile(ctx, basePath, "operations/README.md", "### Operations (Customizations)")
}

func (s *Scanner) scanGenericRepo(ctx *strings.Builder, basePath string) {
	s.appendFile(ctx, basePath, "README.md", "### README")
}

func (s *Scanner) appendFile(ctx *strings.Builder, basePath, relPath, header string) {
	fullPath := filepath.Join(basePath, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return
	}
	ctx.WriteString(fmt.Sprintf("\n%s\n```\n%s\n```\n", header, string(data)))
}

func (s *Scanner) appendFileHead(ctx *strings.Builder, basePath, relPath, header string, maxLines int) {
	fullPath := filepath.Join(basePath, relPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return
	}
	lines := strings.SplitN(string(data), "\n", maxLines+1)
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	ctx.WriteString(fmt.Sprintf("\n%s\n```\n%s\n```\n", header, strings.Join(lines, "\n")))
}

func (s *Scanner) appendFileGlob(ctx *strings.Builder, basePath, pattern, header string, maxFiles int) {
	matches, err := filepath.Glob(filepath.Join(basePath, pattern))
	if err != nil || len(matches) == 0 {
		return
	}
	ctx.WriteString(fmt.Sprintf("\n%s\n", header))
	count := 0
	for _, match := range matches {
		if count >= maxFiles {
			ctx.WriteString(fmt.Sprintf("\n... and %d more files\n", len(matches)-maxFiles))
			break
		}
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		relPath, _ := filepath.Rel(basePath, match)
		ctx.WriteString(fmt.Sprintf("\n#### %s\n```go\n%s\n```\n", relPath, string(data)))
		count++
	}
}

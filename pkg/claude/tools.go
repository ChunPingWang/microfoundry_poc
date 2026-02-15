package claude

import (
	"github.com/anthropics/anthropic-sdk-go"
)

// DeveloperTools returns the tool definitions for the Developer Agent's coding loop.
func DeveloperTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        "read_file",
				Description: anthropic.String("Read the contents of a file at the given path"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Absolute or relative path to the file to read",
						},
					},
					Required: []string{"path"},
				},
			},
		},
		anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        "write_file",
				Description: anthropic.String("Write content to a file at the given path, creating directories as needed"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Absolute or relative path to the file to write",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "The full content to write to the file",
						},
					},
					Required: []string{"path", "content"},
				},
			},
		},
		anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        "list_files",
				Description: anthropic.String("List files and directories at the given path"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Directory path to list",
						},
						"recursive": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether to list files recursively",
						},
					},
					Required: []string{"path"},
				},
			},
		},
		anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        "run_command",
				Description: anthropic.String("Run a shell command and return stdout/stderr"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"command": map[string]interface{}{
							"type":        "string",
							"description": "The shell command to execute",
						},
						"working_dir": map[string]interface{}{
							"type":        "string",
							"description": "Working directory for the command (optional)",
						},
					},
					Required: []string{"command"},
				},
			},
		},
		anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        "search_files",
				Description: anthropic.String("Search for a pattern in files using grep-like matching"),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: map[string]interface{}{
						"pattern": map[string]interface{}{
							"type":        "string",
							"description": "The regex pattern to search for",
						},
						"path": map[string]interface{}{
							"type":        "string",
							"description": "Directory or file path to search in",
						},
						"file_glob": map[string]interface{}{
							"type":        "string",
							"description": "Glob pattern to filter files (e.g. *.go)",
						},
					},
					Required: []string{"pattern", "path"},
				},
			},
		},
	}
}

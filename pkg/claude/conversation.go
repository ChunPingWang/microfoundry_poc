package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
)

// Conversation manages a multi-turn conversation with Claude, supporting tool use.
type Conversation struct {
	client       *Client
	systemPrompt string
	messages     []anthropic.MessageParam
	tools        []anthropic.ToolUnionParam
}

// NewConversation creates a new multi-turn conversation.
func NewConversation(client *Client, systemPrompt string, tools []anthropic.ToolUnionParam) *Conversation {
	return &Conversation{
		client:       client,
		systemPrompt: systemPrompt,
		messages:     []anthropic.MessageParam{},
		tools:        tools,
	}
}

// ToolExecutor is a function that executes a tool call and returns the result.
type ToolExecutor func(toolName string, input map[string]interface{}) (string, error)

// SendMessage sends a user message and processes the response, executing tool calls as needed.
func (conv *Conversation) SendMessage(ctx context.Context, message string, executor ToolExecutor) (string, error) {
	conv.messages = append(conv.messages,
		anthropic.NewUserMessage(anthropic.NewTextBlock(message)),
	)

	maxTokens := conv.client.Cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	for {
		params := anthropic.MessageNewParams{
			Model:     conv.client.Model,
			MaxTokens: maxTokens,
			Messages:  conv.messages,
		}

		if conv.systemPrompt != "" {
			params.System = []anthropic.TextBlockParam{
				{Text: conv.systemPrompt},
			}
		}

		if len(conv.tools) > 0 {
			params.Tools = conv.tools
		}

		response, err := conv.client.API.Messages.New(ctx, params)
		if err != nil {
			return "", fmt.Errorf("conversation API call failed: %w", err)
		}

		// Append assistant response to conversation history
		conv.messages = append(conv.messages, response.ToParam())

		// If stop reason is end_turn, return the text content
		if response.StopReason == anthropic.StopReasonEndTurn {
			return ExtractText(response.Content), nil
		}

		// If stop reason is tool_use, execute tools and continue
		if response.StopReason == anthropic.StopReasonToolUse {
			var toolResults []anthropic.ContentBlockParamUnion
			for _, block := range response.Content {
				if tub, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
					var input map[string]interface{}
					if err := json.Unmarshal(tub.Input, &input); err != nil {
						input = make(map[string]interface{})
					}

					result, execErr := executor(tub.Name, input)
					if execErr != nil {
						toolResults = append(toolResults,
							anthropic.NewToolResultBlock(tub.ID, fmt.Sprintf("Error: %s", execErr.Error()), true),
						)
					} else {
						toolResults = append(toolResults,
							anthropic.NewToolResultBlock(tub.ID, result, false),
						)
					}
				}
			}
			conv.messages = append(conv.messages,
				anthropic.NewUserMessage(toolResults...),
			)
			continue
		}

		// Unexpected stop reason — return what we have
		return ExtractText(response.Content), nil
	}
}

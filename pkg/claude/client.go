package claude

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/younjinjeong/microfoundry/pkg/config"
)

// Client wraps the Anthropic Go SDK for agent interactions.
type Client struct {
	API   anthropic.Client
	Model anthropic.Model
	Cfg   config.AnthropicConfig
}

// NewClient creates a new Claude API client.
func NewClient(cfg config.AnthropicConfig) *Client {
	opts := []option.RequestOption{}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}

	model := anthropic.Model(cfg.Model)
	if cfg.Model == "" {
		model = anthropic.ModelClaudeSonnet4_5_20250929
	}

	return &Client{
		API:   anthropic.NewClient(opts...),
		Model: model,
		Cfg:   cfg,
	}
}

// Ask sends a single prompt and returns the text response.
func (c *Client) Ask(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	maxTokens := c.Cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}

	params := anthropic.MessageNewParams{
		Model:     c.Model,
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	message, err := c.API.Messages.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("claude API call failed: %w", err)
	}

	return ExtractText(message.Content), nil
}

// AskWithThinking sends a prompt with extended thinking enabled (for deep analysis).
func (c *Client) AskWithThinking(ctx context.Context, systemPrompt, userPrompt string, thinkingBudget int64) (thinking string, response string, err error) {
	if thinkingBudget == 0 {
		thinkingBudget = c.Cfg.ThinkingBudget
	}
	if thinkingBudget == 0 {
		thinkingBudget = 10000
	}

	maxTokens := c.Cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 16000
	}
	if maxTokens < thinkingBudget+4096 {
		maxTokens = thinkingBudget + 4096
	}

	params := anthropic.MessageNewParams{
		Model:     c.Model,
		MaxTokens: maxTokens,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userPrompt)),
		},
		Thinking: anthropic.ThinkingConfigParamUnion{
			OfEnabled: &anthropic.ThinkingConfigEnabledParam{
				BudgetTokens: thinkingBudget,
			},
		},
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	message, err := c.API.Messages.New(ctx, params)
	if err != nil {
		return "", "", fmt.Errorf("claude API call with thinking failed: %w", err)
	}

	var thinkingParts []string
	var responseParts []string

	for _, block := range message.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.ThinkingBlock:
			thinkingParts = append(thinkingParts, variant.Thinking)
		case anthropic.TextBlock:
			responseParts = append(responseParts, variant.Text)
		}
	}

	return strings.Join(thinkingParts, "\n"), strings.Join(responseParts, "\n"), nil
}

// ExtractText extracts all text content from message content blocks.
func ExtractText(content []anthropic.ContentBlockUnion) string {
	var parts []string
	for _, block := range content {
		if tb, ok := block.AsAny().(anthropic.TextBlock); ok {
			parts = append(parts, tb.Text)
		}
	}
	return strings.Join(parts, "\n")
}

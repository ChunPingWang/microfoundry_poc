package agents

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// PipelineOpts configures the pipeline behavior.
type PipelineOpts struct {
	StopOnFailure   bool
	DryRun          bool
	SkipAgents      []AgentRole
	ResumeFromAgent AgentRole
}

// Pipeline runs agents in sequence, passing context between them.
type Pipeline struct {
	agents []Agent
	opts   PipelineOpts
	logger *slog.Logger
}

// NewPipeline creates a pipeline with the given agents.
func NewPipeline(opts PipelineOpts, logger *slog.Logger, agents ...Agent) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pipeline{
		agents: agents,
		opts:   opts,
		logger: logger,
	}
}

// Run executes the pipeline from start to finish.
func (p *Pipeline) Run(ctx context.Context, initial *PipelineContext) error {
	pctx := initial
	started := false

	for _, agent := range p.agents {
		role := agent.Role()

		// If resuming, skip until we reach the target agent
		if p.opts.ResumeFromAgent != "" && !started {
			if role != p.opts.ResumeFromAgent {
				p.logger.Info("skipping agent (resuming later)", "agent", role)
				continue
			}
			started = true
		}

		// Check if agent is in the skip list
		if p.shouldSkip(role) {
			p.logger.Info("skipping agent (configured)", "agent", role)
			continue
		}

		p.logger.Info("starting agent",
			"agent", role,
			"epic", pctx.EpicTitle,
			"issue", pctx.IssueNumber,
		)

		if p.opts.DryRun {
			p.logger.Info("dry run — skipping execution", "agent", role)
			continue
		}

		start := time.Now()
		result, err := agent.Run(ctx, pctx)
		elapsed := time.Since(start)

		if err != nil {
			p.logger.Error("agent failed",
				"agent", role,
				"error", err,
				"elapsed", elapsed,
			)
			if p.opts.StopOnFailure {
				return fmt.Errorf("agent %s failed after %s: %w", role, elapsed, err)
			}
			continue
		}

		p.logger.Info("agent completed",
			"agent", role,
			"success", result.Success,
			"summary", result.Summary,
			"elapsed", elapsed,
			"artifacts", len(result.Artifacts),
		)

		// Update pipeline context for next agent
		if result.NextContext != nil {
			pctx = result.NextContext
		}
	}

	p.logger.Info("pipeline completed", "epic", pctx.EpicTitle)
	return nil
}

func (p *Pipeline) shouldSkip(role AgentRole) bool {
	for _, skip := range p.opts.SkipAgents {
		if skip == role {
			return true
		}
	}
	return false
}

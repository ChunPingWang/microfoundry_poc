package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/younjinjeong/microfoundry/pkg/agents"
	"github.com/younjinjeong/microfoundry/pkg/agents/analyzer"
	"github.com/younjinjeong/microfoundry/pkg/agents/dataengineer"
	"github.com/younjinjeong/microfoundry/pkg/agents/designer"
	"github.com/younjinjeong/microfoundry/pkg/agents/developer"
	"github.com/younjinjeong/microfoundry/pkg/agents/reviewer"
	"github.com/younjinjeong/microfoundry/pkg/claude"
	"github.com/younjinjeong/microfoundry/pkg/codebase"
	"github.com/younjinjeong/microfoundry/pkg/config"
	ghpkg "github.com/younjinjeong/microfoundry/pkg/github"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "mf",
		Short:   "MicroFoundry — AI-powered micro CloudFoundry for Kubernetes",
		Version: version,
	}

	rootCmd.AddCommand(newEpicCmd())
	rootCmd.AddCommand(newAnalyzeCmd())
	rootCmd.AddCommand(newDataEngineerCmd())
	rootCmd.AddCommand(newDesignCmd())
	rootCmd.AddCommand(newDevelopCmd())
	rootCmd.AddCommand(newReviewCmd())
	rootCmd.AddCommand(newConfigCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// newEpicCmd runs the full agent pipeline.
func newEpicCmd() *cobra.Command {
	var description string
	var skipAgents []string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "epic [title]",
		Short: "Run the full agent pipeline for an Epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
			claudeClient := claude.NewClient(cfg.Anthropic)
			ghClient := ghpkg.NewClient(cfg.GitHub.Owner, cfg.GitHub.Repo)
			scanner := codebase.NewScanner(0)

			// Build skip list
			var skipRoles []agents.AgentRole
			for _, s := range skipAgents {
				skipRoles = append(skipRoles, agents.AgentRole(s))
			}

			// Resolve working directory
			workDir, _ := os.Getwd()

			pctx := &agents.PipelineContext{
				EpicTitle:        title,
				EpicDescription:  description,
				RepoOwner:        cfg.GitHub.Owner,
				RepoName:         cfg.GitHub.Repo,
				WorkDir:          workDir,
				RefCodebasePaths: cfg.Pipeline.RefCodebases,
				Metadata:         make(map[string]string),
			}

			pipeline := agents.NewPipeline(
				agents.PipelineOpts{
					StopOnFailure: cfg.Pipeline.StopOnFailure,
					DryRun:        dryRun,
					SkipAgents:    skipRoles,
				},
				logger,
				analyzer.New(claudeClient, ghClient, scanner),
				dataengineer.New(claudeClient, ghClient),
				designer.New(claudeClient, ghClient, cfg.Agents.Designer.SkipIfNoUI),
				developer.New(claudeClient, ghClient),
				reviewer.New(claudeClient, ghClient),
			)

			return pipeline.Run(context.Background(), pctx)
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Epic description")
	cmd.Flags().StringSliceVar(&skipAgents, "skip", nil, "Agents to skip (analyzer,data-engineer,designer,developer,reviewer)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print pipeline steps without executing")

	return cmd
}

// newAnalyzeCmd runs only the Analyzer agent.
func newAnalyzeCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "analyze [title]",
		Short: "Run the Analyzer agent to create an Epic issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			claudeClient := claude.NewClient(cfg.Anthropic)
			ghClient := ghpkg.NewClient(cfg.GitHub.Owner, cfg.GitHub.Repo)
			scanner := codebase.NewScanner(0)

			workDir, _ := os.Getwd()
			pctx := &agents.PipelineContext{
				EpicTitle:        title,
				EpicDescription:  description,
				RepoOwner:        cfg.GitHub.Owner,
				RepoName:         cfg.GitHub.Repo,
				WorkDir:          workDir,
				RefCodebasePaths: cfg.Pipeline.RefCodebases,
				Metadata:         make(map[string]string),
			}

			agent := analyzer.New(claudeClient, ghClient, scanner)
			result, err := agent.Run(context.Background(), pctx)
			if err != nil {
				return err
			}
			fmt.Printf("Analyzer: %s\n", result.Summary)
			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Epic description")
	return cmd
}

// newDataEngineerCmd runs only the Data Engineer agent.
func newDataEngineerCmd() *cobra.Command {
	var issueNumber int

	cmd := &cobra.Command{
		Use:   "data-engineer",
		Short: "Run the Data Engineer agent on an existing issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			claudeClient := claude.NewClient(cfg.Anthropic)
			ghClient := ghpkg.NewClient(cfg.GitHub.Owner, cfg.GitHub.Repo)

			pctx := &agents.PipelineContext{
				IssueNumber: issueNumber,
				RepoOwner:   cfg.GitHub.Owner,
				RepoName:    cfg.GitHub.Repo,
			}

			agent := dataengineer.New(claudeClient, ghClient)
			result, err := agent.Run(context.Background(), pctx)
			if err != nil {
				return err
			}
			fmt.Printf("Data Engineer: %s\n", result.Summary)
			return nil
		},
	}

	cmd.Flags().IntVar(&issueNumber, "issue", 0, "GitHub issue number")
	_ = cmd.MarkFlagRequired("issue")
	return cmd
}

// newDesignCmd runs only the Designer agent.
func newDesignCmd() *cobra.Command {
	var issueNumber int

	cmd := &cobra.Command{
		Use:   "design",
		Short: "Run the Designer agent on an existing issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			claudeClient := claude.NewClient(cfg.Anthropic)
			ghClient := ghpkg.NewClient(cfg.GitHub.Owner, cfg.GitHub.Repo)

			pctx := &agents.PipelineContext{
				IssueNumber: issueNumber,
				RepoOwner:   cfg.GitHub.Owner,
				RepoName:    cfg.GitHub.Repo,
			}

			agent := designer.New(claudeClient, ghClient, cfg.Agents.Designer.SkipIfNoUI)
			result, err := agent.Run(context.Background(), pctx)
			if err != nil {
				return err
			}
			fmt.Printf("Designer: %s\n", result.Summary)
			return nil
		},
	}

	cmd.Flags().IntVar(&issueNumber, "issue", 0, "GitHub issue number")
	_ = cmd.MarkFlagRequired("issue")
	return cmd
}

// newDevelopCmd runs only the Developer agent.
func newDevelopCmd() *cobra.Command {
	var issueNumber int
	var branchName string

	cmd := &cobra.Command{
		Use:   "develop",
		Short: "Run the Developer agent on an existing issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			claudeClient := claude.NewClient(cfg.Anthropic)
			ghClient := ghpkg.NewClient(cfg.GitHub.Owner, cfg.GitHub.Repo)
			workDir, _ := os.Getwd()

			pctx := &agents.PipelineContext{
				IssueNumber: issueNumber,
				BranchName:  branchName,
				RepoOwner:   cfg.GitHub.Owner,
				RepoName:    cfg.GitHub.Repo,
				WorkDir:     workDir,
			}

			agent := developer.New(claudeClient, ghClient)
			result, err := agent.Run(context.Background(), pctx)
			if err != nil {
				return err
			}
			fmt.Printf("Developer: %s\n", result.Summary)
			return nil
		},
	}

	cmd.Flags().IntVar(&issueNumber, "issue", 0, "GitHub issue number")
	cmd.Flags().StringVar(&branchName, "branch", "", "Branch name to work on")
	_ = cmd.MarkFlagRequired("issue")
	_ = cmd.MarkFlagRequired("branch")
	return cmd
}

// newReviewCmd runs only the Reviewer agent.
func newReviewCmd() *cobra.Command {
	var prNumber int

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Run the Review agent on an existing PR",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			claudeClient := claude.NewClient(cfg.Anthropic)
			ghClient := ghpkg.NewClient(cfg.GitHub.Owner, cfg.GitHub.Repo)
			workDir, _ := os.Getwd()

			pctx := &agents.PipelineContext{
				PRNumber:  prNumber,
				RepoOwner: cfg.GitHub.Owner,
				RepoName:  cfg.GitHub.Repo,
				WorkDir:   workDir,
			}

			agent := reviewer.New(claudeClient, ghClient)
			result, err := agent.Run(context.Background(), pctx)
			if err != nil {
				return err
			}
			fmt.Printf("Reviewer: %s\n", result.Summary)
			return nil
		},
	}

	cmd.Flags().IntVar(&prNumber, "pr", 0, "Pull request number")
	_ = cmd.MarkFlagRequired("pr")
	return cmd
}

// newConfigCmd handles configuration.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage MicroFoundry configuration",
	}

	setCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.SetValue(args[0], args[1]); err != nil {
				return err
			}
			fmt.Printf("Set %s\n", args[0])
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val, err := config.GetValue(args[0])
			if err != nil {
				return err
			}
			fmt.Println(val)
			return nil
		},
	}

	cmd.AddCommand(setCmd, getCmd)
	return cmd
}

// Ensure strings import is used (for skipAgents parsing)
var _ = strings.Join

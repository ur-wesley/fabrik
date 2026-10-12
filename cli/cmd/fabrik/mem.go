package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/clierr"
	"github.com/ur-wesley/fabrik/cli/internal/export"
	"github.com/ur-wesley/fabrik/cli/internal/memory"
)

var observationIDRe = regexp.MustCompile(`^o-[0-9a-f]+$`)

func newMemCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "mem",
		Short: "Machine-wide memory (SQLite)",
	}
	root.AddCommand(newMemCheckCmd())
	root.AddCommand(newMemContextCmd())
	root.AddCommand(newMemSearchCmd())
	root.AddCommand(newMemSaveCmd())
	root.AddCommand(newMemGetCmd())
	root.AddCommand(newMemSessionCmd())
	root.AddCommand(newMemExportCmd())
	root.AddCommand(newMemUpdateCmd())
	root.AddCommand(newMemDeleteCmd())
	root.AddCommand(newMemTimelineCmd())
	root.AddCommand(newMemSavePromptCmd())
	root.AddCommand(newMemSuggestTopicKeyCmd())
	root.AddCommand(newMemStatsCmd())
	root.AddCommand(newMemCurrentProjectCmd())
	return root
}

func newMemExportCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export observations JSONL snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := export.Observations(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]interface{}{
					"observations":      res.Count,
					"path_observations": res.Path,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exported %d observations\n", res.Count)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newMemCheckCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Report memory.db health",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := memory.Check(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintCheckJSON(cmd.OutOrStdout(), r)
			}
			memory.PrintCheck(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newMemContextCmd() *cobra.Command {
	var asJSON bool
	var project string
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Load memory context for agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := memory.Context(cmd.Context(), memory.ContextOptions{Project: project})
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintContextJSON(cmd.OutOrStdout(), r)
			}
			memory.PrintContext(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&project, "project", "", "filter observations to this repo path label")
	return cmd
}

func newMemSearchCmd() *cobra.Command {
	var asJSON bool
	var limit int
	var project string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search observations",
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			obs, err := memory.Search(cmd.Context(), memory.SearchOptions{
				Query: query, Limit: limit, Project: project,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintSearchJSON(cmd.OutOrStdout(), obs)
			}
			memory.PrintSearch(cmd.OutOrStdout(), obs)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().IntVar(&limit, "limit", 20, "max results (1-100)")
	cmd.Flags().StringVar(&project, "project", "", "filter observations to this repo path label")
	return cmd
}

func newMemSaveCmd() *cobra.Command {
	var asJSON bool
	var title, body, kind, symbolID, topicKey, capturePrompt string
	cmd := &cobra.Command{
		Use:   "save",
		Short: "Save an observation",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := memory.Save(cmd.Context(), memory.SaveOptions{
				Title: title, Body: body, Kind: kind, SymbolID: symbolID, Cwd: repoStartDir(),
				TopicKey: topicKey, CapturePrompt: capturePrompt,
			})
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintObservationJSON(cmd.OutOrStdout(), o)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", o.ID, o.Title)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&title, "title", "", "observation title")
	cmd.Flags().StringVar(&body, "body", "", "observation body")
	cmd.Flags().StringVar(&kind, "kind", "discovery", "decision|bugfix|pattern|discovery|handoff")
	cmd.Flags().StringVar(&symbolID, "symbol-id", "", "optional sym:… id")
	cmd.Flags().StringVar(&topicKey, "topic-key", "", "optional topic key")
	cmd.Flags().StringVar(&capturePrompt, "capture-prompt", "", "optional prompt body")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}

func newMemGetCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get observation by id",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if !observationIDRe.MatchString(id) {
				return clierr.Usage("fabrik: observation id must match o-<hex>")
			}
			o, err := memory.Get(cmd.Context(), id)
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintGetJSON(cmd.OutOrStdout(), o)
			}
			memory.PrintGet(cmd.OutOrStdout(), o)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newMemSessionCmd() *cobra.Command {
	session := &cobra.Command{Use: "session", Short: "Session lifecycle"}
	session.AddCommand(newMemSessionStartCmd())
	session.AddCommand(newMemSessionSummaryCmd())
	session.AddCommand(newMemSessionEndCmd())
	return session
}

func newMemSessionStartCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a memory session",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := memory.SessionStart(cmd.Context(), "")
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintSessionStartJSON(cmd.OutOrStdout(), r)
			}
			memory.PrintSessionStart(cmd.OutOrStdout(), r)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newMemSessionSummaryCmd() *cobra.Command {
	var asJSON bool
	var goal, discoveries, accomplished, nextSteps, files string
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize session",
		RunE: func(cmd *cobra.Command, args []string) error {
			opt := memory.SessionSummaryOptions{Cwd: repoStartDir()}
			if cmd.Flags().Changed("goal") {
				opt.GoalSet = true
				opt.Goal = goal
			}
			if cmd.Flags().Changed("discoveries") {
				opt.DiscoveriesSet = true
				opt.Discoveries = discoveries
			}
			if cmd.Flags().Changed("accomplished") {
				opt.AccomplishedSet = true
				opt.Accomplished = accomplished
			}
			if cmd.Flags().Changed("next") {
				opt.NextStepsSet = true
				opt.NextSteps = nextSteps
			}
			if cmd.Flags().Changed("files") {
				opt.FilesSet = true
				opt.Files = files
			}
			s, err := memory.SessionSummary(cmd.Context(), opt)
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintSessionJSON(cmd.OutOrStdout(), s)
			}
			memory.PrintSession(cmd.OutOrStdout(), s)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	cmd.Flags().StringVar(&goal, "goal", "", "session goal")
	cmd.Flags().StringVar(&discoveries, "discoveries", "", "discoveries")
	cmd.Flags().StringVar(&accomplished, "accomplished", "", "accomplished work")
	cmd.Flags().StringVar(&nextSteps, "next", "", "next steps")
	cmd.Flags().StringVar(&files, "files", "", "newline-separated file paths")
	return cmd
}

func newMemSessionEndCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "end",
		Short: "End session",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := memory.SessionEnd(cmd.Context(), "")
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintSessionJSON(cmd.OutOrStdout(), s)
			}
			memory.PrintSession(cmd.OutOrStdout(), s)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newMemUpdateCmd() *cobra.Command {
	var asJSON bool
	var title, body, kind, topicKey, symbolID string
	var id string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an observation",
		RunE: func(cmd *cobra.Command, args []string) error {
			opt := memory.UpdateOptions{ID: id}
			if cmd.Flags().Changed("title") {
				opt.Title = &title
			}
			if cmd.Flags().Changed("body") {
				opt.Body = &body
			}
			if cmd.Flags().Changed("kind") {
				opt.Kind = &kind
			}
			if cmd.Flags().Changed("topic-key") {
				opt.TopicKey = &topicKey
			}
			if cmd.Flags().Changed("symbol-id") {
				opt.SymbolID = &symbolID
			}
			o, err := memory.Update(cmd.Context(), opt)
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintObservationJSON(cmd.OutOrStdout(), o)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", o.ID, o.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "observation id")
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&body, "body", "", "new body")
	cmd.Flags().StringVar(&kind, "kind", "", "new kind")
	cmd.Flags().StringVar(&topicKey, "topic-key", "", "new topic key")
	cmd.Flags().StringVar(&symbolID, "symbol-id", "", "new symbol id")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newMemDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Soft-delete an observation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return memory.Delete(cmd.Context(), id)
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "observation id")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newMemTimelineCmd() *cobra.Command {
	var asJSON bool
	var id, project string
	var limit int
	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Observations around an id",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := memory.Timeline(cmd.Context(), memory.TimelineOptions{ID: id, Limit: limit, Project: project})
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(r)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", r.Observation.ID, r.Observation.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "observation id")
	cmd.Flags().IntVar(&limit, "limit", 5, "neighbours per side")
	cmd.Flags().StringVar(&project, "project", "", "filter by project label")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func newMemSavePromptCmd() *cobra.Command {
	var asJSON bool
	var title, capture string
	cmd := &cobra.Command{
		Use:   "save-prompt",
		Short: "Save a prompt observation",
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := memory.SavePrompt(cmd.Context(), memory.SavePromptOptions{
				Title: title, CapturePrompt: capture, Cwd: repoStartDir(),
			})
			if err != nil {
				return err
			}
			if asJSON {
				return memory.PrintObservationJSON(cmd.OutOrStdout(), o)
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", o.ID, o.Title)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "prompt title")
	cmd.Flags().StringVar(&capture, "capture-prompt", "", "prompt text")
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	_ = cmd.MarkFlagRequired("capture-prompt")
	return cmd
}

func newMemSuggestTopicKeyCmd() *cobra.Command {
	var title string
	cmd := &cobra.Command{
		Use:   "suggest-topic-key",
		Short: "Suggest topic_key from title",
		RunE: func(cmd *cobra.Command, args []string) error {
			key := memory.SuggestTopicKey(memory.SuggestTopicKeyOptions{Title: title})
			fmt.Fprintln(cmd.OutOrStdout(), key)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "observation title")
	_ = cmd.MarkFlagRequired("title")
	return cmd
}

func newMemStatsCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Memory totals",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := memory.Stats(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(st)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "observations %d  sessions %d  projects %d\n",
				st.ObservationCount, st.SessionCount, len(st.Projects))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

func newMemCurrentProjectCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "current-project",
		Short: "Detect project from cwd",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := memory.CurrentProject(repoStartDir())
			if asJSON {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(r)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s (%s)\n", r.Project, r.Source)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "machine-readable output")
	return cmd
}

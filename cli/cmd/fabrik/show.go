package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/tmpl"
)

// showKeys lists the fixed single-word show keys (skill/agent/prompt take a second arg).
var showKeys = []string{"workflow", "context", "config", "readme", "skills-index", "styleguide", "prompt", "skill", "agent"}

// nameRe allows only safe skill/agent names (no traversal, no flags, no spaces).
var nameRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <key> [name]",
		Short: "Print embedded .fabrik content to stdout (no writes)",
		Long: `Print embedded templates to stdout. Keys: workflow, context, config, readme,
skills-index, styleguide, prompt <plan|build>, skill <name>, agent <name>.`,
		Args: validateShowArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveShowPath(args)
			if err != nil {
				return err
			}
			content, rerr := tmpl.Read(path)
			if rerr != nil {
				return fmt.Errorf("unknown key %q: %s", strings.Join(args, " "), validShowKeys())
			}
			fmt.Fprint(cmd.OutOrStdout(), content)
			return nil
		},
	}
}

// validateShowArgs enforces arity: single-word keys take exactly 1 arg,
// prompt/skill/agent take exactly 2 (name validated for skill/agent).
func validateShowArgs(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing key: %s", validShowKeys())
	}
	if len(args) > 2 {
		return fmt.Errorf("too many args %q: %s", strings.Join(args, " "), validShowKeys())
	}
	switch args[0] {
	case "prompt":
		if len(args) != 2 {
			return fmt.Errorf("prompt requires plan|build: %s", validShowKeys())
		}
	case "skill", "agent":
		if len(args) != 2 {
			return fmt.Errorf("%s requires a name: %s", args[0], validShowKeys())
		}
		if !nameRe.MatchString(args[1]) {
			return fmt.Errorf("invalid %s name %q: use [A-Za-z0-9_-]+", args[0], args[1])
		}
	default:
		if len(args) != 1 {
			return fmt.Errorf("unknown key %q: %s", strings.Join(args, " "), validShowKeys())
		}
	}
	return nil
}

func resolveShowPath(args []string) (string, error) {
	switch args[0] {
	case "workflow":
		return "workflow-note.md", nil
	case "context":
		return "fabrik/CONTEXT.md", nil
	case "config":
		return "fabrik/config.yaml", nil
	case "readme":
		return "fabrik/README.md", nil
	case "skills-index":
		return "fabrik/skills.md", nil
	case "styleguide":
		return "hub/STYLEGUIDE.md", nil
	case "prompt":
		if len(args) == 2 {
			switch args[1] {
			case "plan":
				return "hub/PROMPT_plan.md", nil
			case "build":
				return "hub/PROMPT_build.md", nil
			}
		}
		return "", fmt.Errorf("unknown key %q: %s", strings.Join(args, " "), validShowKeys())
	case "skill":
		if len(args) == 2 && nameRe.MatchString(args[1]) {
			return "skills/" + args[1] + ".md", nil
		}
		return "", fmt.Errorf("unknown key %q: %s", strings.Join(args, " "), validShowKeys())
	case "agent":
		if len(args) == 2 && nameRe.MatchString(args[1]) {
			return "agents/" + args[1] + ".md", nil
		}
		return "", fmt.Errorf("unknown key %q: %s", strings.Join(args, " "), validShowKeys())
	}
	return "", fmt.Errorf("unknown key %q: %s", strings.Join(args, " "), validShowKeys())
}

// validShowKeys derives from showKeys (single source with list all).
func validShowKeys() string {
	return "valid keys: " + strings.Join(showKeys, ", ") +
		" (prompt <plan|build>, skill <name>, agent <name>)"
}

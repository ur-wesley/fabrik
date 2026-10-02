package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func splitLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func TestShowEachKey(t *testing.T) {
	for _, args := range [][]string{
		{"show", "workflow"},
		{"show", "context"},
		{"show", "config"},
		{"show", "readme"},
		{"show", "skills-index"},
		{"show", "styleguide"},
		{"show", "prompt", "plan"},
		{"show", "prompt", "build"},
	} {
		out, err := run(t, args...)
		require.NoError(t, err, "%v", args)
		assert.NotEmpty(t, out, "%v", args)
	}
}

func TestShowSkillAgent(t *testing.T) {
	skillsOut, err := run(t, "list", "skills")
	require.NoError(t, err)
	require.NotEmpty(t, skillsOut)
	agentsOut, err := run(t, "list", "agents")
	require.NoError(t, err)
	require.NotEmpty(t, agentsOut)

	// listed entries must round-trip through show (first entry each group)
	skillsFirst := splitLines(skillsOut)[0]
	agentsFirst := splitLines(agentsOut)[0]
	out, err := run(t, "show", "skill", skillsFirst)
	require.NoError(t, err, "skill %s must round-trip", skillsFirst)
	assert.NotEmpty(t, out)
	out, err = run(t, "show", "agent", agentsFirst)
	require.NoError(t, err, "agent %s must round-trip", agentsFirst)
	assert.NotEmpty(t, out)
}

func TestShowUnknownKeyFails(t *testing.T) {
	_, err := run(t, "show", "nope-missing")
	require.Error(t, err)
}

func TestShowTraversalRejected(t *testing.T) {
	for _, args := range [][]string{
		{"show", "skill", "../escape"},
		{"show", "skill", "../../etc/passwd"},
		{"show", "skill", "a/b"},
		{"show", "skill", ""},
		{"show", "agent", "../escape"},
		{"show", "agent", "a b"},
		{"show", "agent", "-f"},
	} {
		_, err := run(t, args...)
		require.Error(t, err, "%v must be rejected", args)
	}
}

func TestShowPromptRequiresSubarg(t *testing.T) {
	_, err := run(t, "show", "prompt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plan|build")
	_, err = run(t, "show", "prompt", "bogus")
	require.Error(t, err)
}

func TestShowExtraArgRejected(t *testing.T) {
	for _, args := range [][]string{
		{"show", "workflow", "extra"},
		{"show", "skill", "tdd", "extra"},
		{"show", "prompt", "plan", "extra"},
	} {
		_, err := run(t, args...)
		require.Error(t, err, "%v must be rejected", args)
	}
}

func TestListGroups(t *testing.T) {
	for _, g := range []string{"skills", "agents", "all"} {
		out, err := run(t, "list", g)
		require.NoError(t, err)
		assert.NotEmpty(t, out, g)
	}
	_, err := run(t, "list", "bogus")
	require.Error(t, err)
}

func TestListAllShowsPromptVariants(t *testing.T) {
	out, err := run(t, "list", "all")
	require.NoError(t, err)
	assert.Contains(t, out, "prompt plan")
	assert.Contains(t, out, "prompt build")
	for _, line := range splitLines(out) {
		assert.NotEqual(t, "prompt", line, "bare prompt must not be listed")
	}
}

func TestMigrateRequiresPrune(t *testing.T) {
	_, err := run(t, "migrate")
	require.Error(t, err, "bare migrate must fail, not print help with exit 0")
	assert.Contains(t, err.Error(), "--prune")
}

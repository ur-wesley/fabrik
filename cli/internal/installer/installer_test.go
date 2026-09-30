package installer

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
)

func TestSetupAllPresentSkipsInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	f := &exec.Fake{
		Path: map[string]bool{"bd": true, "engram": true, "graphify": true, "pi": true},
		Outputs: map[string]string{
			"bd version":                             "bd 1.3.0",
			"engram version":                         "engram 2.2.1",
			"pi install npm:@piarium/pi-mcp-adapter": "",
			"engram setup cursor":                    "",
			"engram setup opencode":                  "",
			"engram setup pi":                        "",
			"bd setup cursor":                        "",
			"bd setup opencode":                      "",
		},
	}
	d := Deps{
		Exec: f,
		Out:  &out,
		Download: func(ctx context.Context, url string) (string, error) {
			p := filepath.Join(t.TempDir(), "skill.md")
			if err := os.WriteFile(p, []byte("skill"), 0o644); err != nil {
				return "", err
			}
			return p, nil
		},
	}
	require.NoError(t, Setup(context.Background(), Config{Yes: true}, d))
	s := out.String()
	assert.Contains(t, s, "bd already on PATH")
	assert.Contains(t, s, "engram already on PATH")
	assert.Contains(t, s, "graphify CLI already on PATH")
	assert.Contains(t, s, "Wrote")
	assert.True(t, f.Called("engram setup cursor"))
	assert.True(t, f.Called("engram setup opencode"))
	assert.True(t, f.Called("engram setup pi"))
	assert.True(t, f.Called("bd setup cursor"))
	assert.True(t, f.Called("bd setup opencode"))
	// no stray agent dirs configured
	assert.NotContains(t, s, ".claude")
}

func TestSetupSkipFlags(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{}}
	d := Deps{
		Exec: f,
		Out:  &out,
		Download: func(ctx context.Context, url string) (string, error) {
			return "", errors.New("must not download")
		},
	}
	// skip-tool-install with nothing on PATH must fail like the shell scripts
	err := Setup(context.Background(), Config{Yes: true, SkipToolInstall: true, SkipEngramSetup: true, SkipPiPackages: true}, d)
	require.Error(t, err)
	assert.False(t, f.Called("engram setup cursor"))
}

func TestSetupDryRunTouchesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{"go": true, "uv": true, "pi": true}}
	d := Deps{Exec: f, Out: &out, Download: func(ctx context.Context, url string) (string, error) {
		return "", errors.New("must not download")
	}}
	require.NoError(t, Setup(context.Background(), Config{Yes: true, DryRun: true}, d))
	assert.Contains(t, out.String(), "dry-run")
	// HOME must contain no new files except nothing
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, strings.HasSuffix(e.Name(), "ai-workflow.mdc"), "dry-run wrote files")
	}
	_ = filepath.Join
}

func TestSetupRepoHookCalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{"bd": true, "engram": true, "graphify": true}, Outputs: map[string]string{
		"engram setup cursor":   "",
		"engram setup opencode": "",
		"engram setup pi":       "",
		"bd setup cursor":       "",
		"bd setup opencode":     "",
	}}
	called := ""
	d := Deps{
		Exec: f,
		Out:  &out,
		Download: func(ctx context.Context, url string) (string, error) {
			p := filepath.Join(t.TempDir(), "skill.md")
			if err := os.WriteFile(p, []byte("skill"), 0o644); err != nil {
				return "", err
			}
			return p, nil
		},
		InitRepo: func(repo string) error {
			called = repo
			return nil
		},
	}
	require.NoError(t, Setup(context.Background(), Config{Yes: true, RepoPath: "/tmp/repo"}, d))
	assert.Equal(t, "/tmp/repo", called)
}

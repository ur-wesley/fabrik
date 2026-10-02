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
	"github.com/ur-wesley/fabrik/cli/internal/paths"
)

func TestSetupAllPresentSkipsInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	f := &exec.Fake{
		Path: map[string]bool{"bd": true, "engram": true, "graphify": true, "pi": true, "cursor": true, "opencode": true},
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
	assert.False(t, f.Called("engram setup opencode"), "OpenCode agent wiring is per-repo only")
	assert.True(t, f.Called("engram setup pi"))
	assert.True(t, f.Called("bd setup cursor"))
	assert.False(t, f.Called("bd setup opencode"), "OpenCode agent wiring is per-repo only")
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

func TestSetupInitsCurrentRepoByDefault(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(repo)
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
		InitRepo: func(path string) error {
			called = path
			return nil
		},
	}
	require.NoError(t, Setup(context.Background(), Config{Yes: true}, d))
	abs, err := filepath.Abs(repo)
	require.NoError(t, err)
	assert.Equal(t, abs, called)
}

func TestSetupWarnsNotGitRepo(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(repo)
	var out bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{"bd": true, "engram": true, "graphify": true}, Outputs: map[string]string{
		"engram setup cursor":   "",
		"engram setup opencode": "",
		"engram setup pi":       "",
		"bd setup cursor":       "",
		"bd setup opencode":     "",
	}}
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
		InitRepo: func(string) error { return nil },
	}
	require.NoError(t, Setup(context.Background(), Config{Yes: true}, d))
	assert.Contains(t, out.String(), "Not a git repository")
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
	repo := filepath.Join(t.TempDir(), "repo")
	require.NoError(t, os.MkdirAll(repo, 0o755))
	require.NoError(t, Setup(context.Background(), Config{Yes: true, RepoPath: repo}, d))
	abs, err := filepath.Abs(repo)
	require.NoError(t, err)
	assert.Equal(t, abs, called)
}

func TestSetupEngramSkipsMissingApp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	// pi absent from PATH: engram setup pi must be skipped, not fail.
	f := &exec.Fake{Path: map[string]bool{
		"bd": true, "engram": true, "graphify": true,
		"cursor": true, "opencode": true,
	}, Outputs: map[string]string{
		"engram setup cursor":   "",
		"engram setup opencode": "",
		"bd setup cursor":       "",
		"bd setup opencode":     "",
	}}
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
	assert.True(t, f.Called("engram setup cursor"))
	assert.False(t, f.Called("engram setup opencode"))
	assert.False(t, f.Called("engram setup pi"))
	assert.Contains(t, out.String(), "pi not on PATH")
}

func TestSetupMachineSetupSkipsGlobalOpenCode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{
		"bd": true, "engram": true, "graphify": true,
		"cursor": true, "opencode": true, "pi": true,
	}, Outputs: map[string]string{
		"pi install npm:@piarium/pi-mcp-adapter": "",
		"engram setup cursor":                    "",
		"engram setup pi":                        "",
		"bd setup cursor":                        "",
	}}
	d := setupDeps(t, f)
	d.Out = &out
	require.NoError(t, Setup(context.Background(), Config{Yes: true}, d))
	assert.False(t, f.Called("engram setup opencode"))
	assert.False(t, f.Called("bd setup opencode"))
	dests, err := paths.WorkflowNoteDests()
	require.NoError(t, err)
	for _, dst := range dests {
		assert.NotContains(t, dst.Path, "opencode")
	}
}

func setupDeps(t *testing.T, f *exec.Fake) Deps {
	return Deps{
		Exec: f,
		Out:  &bytes.Buffer{},
		Confirm: func(_ string, yes bool) bool {
			return yes
		},
		Download: func(ctx context.Context, url string) (string, error) {
			p := filepath.Join(t.TempDir(), "skill.md")
			if err := os.WriteFile(p, []byte("skill"), 0o644); err != nil {
				return "", err
			}
			return p, nil
		},
	}
}

func TestSetupRepoInitBeforeWorkflowNote(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
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
	d := setupDeps(t, f)
	d.Out = &out
	d.InitRepo = func(string) error { return nil }
	require.NoError(t, Setup(context.Background(), Config{RepoPath: repo, Yes: true}, d))
	s := out.String()
	assert.Less(t, strings.Index(s, "Initializing repo"), strings.Index(s, "Installing personal workflow note"))
}

func TestSetupWorkflowNoteAlreadyPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dests, err := paths.WorkflowNoteDests()
	require.NoError(t, err)
	for _, dst := range dests {
		require.NoError(t, os.MkdirAll(filepath.Dir(dst.Path), 0o755))
		require.NoError(t, os.WriteFile(dst.Path, []byte("keep"), 0o644))
	}
	var out bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{"bd": true, "engram": true, "graphify": true}, Outputs: map[string]string{
		"engram setup cursor":   "",
		"engram setup opencode": "",
		"engram setup pi":       "",
		"bd setup cursor":       "",
		"bd setup opencode":     "",
	}}
	d := setupDeps(t, f)
	d.Out = &out
	require.NoError(t, Setup(context.Background(), Config{}, d))
	s := out.String()
	assert.Contains(t, s, "already present")
	assert.NotContains(t, s, "Write ")
	assert.NotContains(t, s, "Personal workflow notes skipped")
	for _, dst := range dests {
		body, err := os.ReadFile(dst.Path)
		require.NoError(t, err)
		assert.Equal(t, "keep", string(body))
	}
}

func TestSetupWorkflowNoteSkippedWithoutYes(t *testing.T) {
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
	d := setupDeps(t, f)
	d.Out = &out
	require.NoError(t, Setup(context.Background(), Config{}, d))
	s := out.String()
	assert.Contains(t, s, "Personal workflow notes skipped")
	dests, err := paths.WorkflowNoteDests()
	require.NoError(t, err)
	for _, dst := range dests {
		assert.NoFileExists(t, dst.Path)
	}
}

func TestSetupWorkflowNoteYesWritesMissing(t *testing.T) {
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
	d := setupDeps(t, f)
	d.Out = &out
	require.NoError(t, Setup(context.Background(), Config{Yes: true}, d))
	s := out.String()
	assert.Contains(t, s, "Wrote")
	dests, err := paths.WorkflowNoteDests()
	require.NoError(t, err)
	for _, dst := range dests {
		assert.FileExists(t, dst.Path)
	}
}

func TestSetupGraphifySkillAlreadyPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	dests, err := paths.GraphifySkills()
	require.NoError(t, err)
	for _, dst := range dests {
		require.NoError(t, os.MkdirAll(filepath.Dir(dst), 0o755))
		require.NoError(t, os.WriteFile(dst, []byte("keep"), 0o644))
	}
	var out bytes.Buffer
	downloaded := false
	f := &exec.Fake{Path: map[string]bool{"bd": true, "engram": true, "graphify": true}, Outputs: map[string]string{
		"engram setup cursor":   "",
		"engram setup opencode": "",
		"engram setup pi":       "",
		"bd setup cursor":       "",
		"bd setup opencode":     "",
	}}
	d := setupDeps(t, f)
	d.Out = &out
	d.Download = func(ctx context.Context, url string) (string, error) {
		downloaded = true
		return "", errors.New("must not download")
	}
	require.NoError(t, Setup(context.Background(), Config{}, d))
	assert.False(t, downloaded)
	assert.Contains(t, out.String(), "already present")
}

func TestSetupEngramBestEffort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{
		"bd": true, "engram": true, "graphify": true,
		"cursor": true, "opencode": true, "pi": true,
	}, Outputs: map[string]string{
		"pi install npm:@piarium/pi-mcp-adapter": "",
		"engram setup cursor":                    "",
		"engram setup opencode":                  "",
		"engram setup pi":                        "",
		"bd setup cursor":                        "",
		"bd setup opencode":                      "",
	}, Errors: map[string]error{
		"engram setup opencode": errors.New("boom"),
	}}
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
	assert.True(t, f.Called("engram setup pi"), "later apps run after a failure")
}

func TestSetupSelectiveAppsCursorOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	f := &exec.Fake{Path: map[string]bool{
		"bd": true, "engram": true, "graphify": true,
		"cursor": true, "pi": true,
	}, Outputs: map[string]string{
		"engram setup cursor": "",
		"bd setup cursor":     "",
	}}
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
	require.NoError(t, Setup(context.Background(), Config{Yes: true, Apps: []string{"cursor"}}, d))
	assert.True(t, f.Called("engram setup cursor"))
	assert.False(t, f.Called("engram setup pi"), "pi setup should be skipped when only cursor is selected")
	assert.False(t, f.Called("pi install npm:@piarium/pi-mcp-adapter"), "pi package install should be skipped")
}


package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirmYesSkipsPrompt(t *testing.T) {
	assert.True(t, Confirm("anything?", true))
}

func TestStylesRender(t *testing.T) {
	assert.Contains(t, Step.Render("==> x"), "==>")
	assert.Contains(t, OK.Render("done"), "done")
}

func TestHeaderAndLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "out.txt")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0o644)
	require.NoError(t, err)
	Header(f, "hello")
	Line(f, "world")
	require.NoError(t, f.Close())
	body, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Contains(t, string(body), "==> hello")
	assert.Contains(t, string(body), "world")
}

func TestConfirmPipedStdinDefaultsYes(t *testing.T) {
	// In `go test` stdin is not a TTY, so non-forced confirm defaults to yes.
	if f, err := os.Stdin.Stat(); err == nil && (f.Mode()&os.ModeCharDevice) != 0 {
		t.Skip("stdin is a TTY")
	}
	assert.True(t, Confirm("anything?", false))
}

func TestSelectAppsYesSkipsPrompt(t *testing.T) {
	apps, err := SelectApps(nil, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"cursor", "pi", "antigravity", "opencode"}, apps)

	custom, err := SelectApps([]string{"pi", "antigravity"}, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"pi", "antigravity"}, custom)
}


package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeRoundTrip(t *testing.T) {
	f := &Fake{
		Path:    map[string]bool{"bd": true},
		Outputs: map[string]string{"bd version": "bd 1.3.0"},
	}
	p, err := f.LookPath("bd")
	require.NoError(t, err)
	assert.NotEmpty(t, p)
	_, err = f.LookPath("missing")
	require.Error(t, err)

	out, err := f.Run(context.Background(), "bd", "version")
	require.NoError(t, err)
	assert.Equal(t, "bd 1.3.0", out)
	assert.True(t, f.Called("bd version"))
	assert.False(t, f.Called("engram"))
}

func TestFakeUnscriptedErrors(t *testing.T) {
	f := &Fake{}
	_, err := f.Run(context.Background(), "bd", "version")
	require.Error(t, err)
}

func TestFakeErrorOverride(t *testing.T) {
	f := &Fake{Errors: map[string]error{"bd version": assert.AnError}}
	_, err := f.Run(context.Background(), "bd", "version")
	assert.ErrorIs(t, err, assert.AnError)
}

func TestExecErrorFormat(t *testing.T) {
	e := &Error{Name: "bd", Msg: "boom", Err: assert.AnError}
	assert.Equal(t, "bd: boom", e.Error())
	assert.ErrorIs(t, e.Unwrap(), assert.AnError)
}

func TestOSRunnerSuccessAndFailure(t *testing.T) {
	out, err := OSRunner{}.Run(context.Background(), "go", "version")
	require.NoError(t, err)
	assert.Contains(t, out, "go version")
	_, err = OSRunner{}.Run(context.Background(), "go", "nonexistent-subcommand-xyz")
	require.Error(t, err)
}

func TestFakeCalledPrefix(t *testing.T) {
	f := &Fake{Outputs: map[string]string{"engram setup cursor": ""}}
	_, _ = f.Run(context.Background(), "engram", "setup", "cursor")
	assert.True(t, f.Called("engram setup"))
	assert.True(t, f.Called("engram"))
	assert.False(t, f.Called("bd"))
}

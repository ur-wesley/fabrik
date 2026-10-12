package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveValidation(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	_, err := Save(context.Background(), SaveOptions{})
	require.Error(t, err)
	_, err = Save(context.Background(), SaveOptions{Title: "x"})
	require.Error(t, err)
}

func TestSaveCreatesDB(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	o, err := Save(context.Background(), SaveOptions{
		Title: "Hash ids",
		Body:  "SHA256 plus nonce.",
		Kind:  "decision",
	})
	require.NoError(t, err)
	assert.True(t, len(o.ID) > 2)
	assert.Equal(t, "o-", o.ID[:2])
	assert.Equal(t, "decision", o.Kind)
}

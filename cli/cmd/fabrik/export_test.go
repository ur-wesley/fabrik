package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportNoStore(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	_, err := run(t, "export")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no store")
}

func TestExportJSON(t *testing.T) {
	skipWithoutCGO(t)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	t.Chdir(dir)
	out, err := run(t, "export", "--json")
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(out), &m))
	assert.Equal(t, float64(0), m["issues"])
}

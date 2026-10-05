package version

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFallsBackToPackageJSON(t *testing.T) {
	old := Version
	Version = ""
	defer func() { Version = old }()
	var p struct {
		Version string `json:"version"`
	}
	require.NoError(t, json.Unmarshal(packageJSON, &p))
	assert.Equal(t, p.Version, Get())
}

func TestGetPrefersStampedVersion(t *testing.T) {
	old := Version
	Version = "9.9.9"
	defer func() { Version = old }()
	assert.Equal(t, "9.9.9", Get())
}

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFallsBackToPackageJSON(t *testing.T) {
	old := Version
	Version = ""
	defer func() { Version = old }()
	assert.Equal(t, "1.0.0", Get())
}

func TestGetPrefersStampedVersion(t *testing.T) {
	old := Version
	Version = "9.9.9"
	defer func() { Version = old }()
	assert.Equal(t, "9.9.9", Get())
}

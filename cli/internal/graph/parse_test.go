package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCallgraphApp(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "callgraph", "pkg", "app", "app.go"))
	require.NoError(t, err)
	fe := parseFile("pkg/app/app.go", "go", data)
	assert.True(t, fe.OK)
	assert.Contains(t, fe.Symbols, extractedSymbol{Name: "Run", Line: 5})
	var names []string
	for _, c := range fe.Calls {
		names = append(names, c.Name)
	}
	assert.Contains(t, names, "Helper")
	assert.Contains(t, names, "Other")
}

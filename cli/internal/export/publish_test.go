package export

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveExportErrorMessage(t *testing.T) {
	inner := errors.New("disk full")
	err := &SaveExportError{IssueID: "f-abc12345", Err: inner}
	assert.Equal(t, "fabrik: saved f-abc12345 but export failed", err.Error())
	assert.Equal(t, inner, errors.Unwrap(err))
}

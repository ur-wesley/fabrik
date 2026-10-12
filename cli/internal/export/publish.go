package export

import "fmt"

type SaveExportError struct {
	IssueID string
	Err     error
}

func (e *SaveExportError) Error() string {
	return fmt.Sprintf("fabrik: saved %s but export failed", e.IssueID)
}

func (e *SaveExportError) Unwrap() error {
	return e.Err
}

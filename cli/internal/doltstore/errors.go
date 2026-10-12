package doltstore

import (
	"errors"
	"fmt"

	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
)

var ErrNotFound = errors.New("issue not found")

var ErrNoReady = errors.New("no ready issues")

type ErrClaimLost struct {
	ID string
}

func (e *ErrClaimLost) Error() string {
	return fmt.Sprintf("fabrik: issue %s was claimed by another agent", e.ID)
}

type ErrNotAssignee struct {
	ID       string
	Assignee string
}

func (e *ErrNotAssignee) Error() string {
	return fmt.Sprintf("fabrik: issue %s is assigned to %s", e.ID, e.Assignee)
}

type ErrIssueStatus struct {
	ID     string
	Status string
}

func (e *ErrIssueStatus) Error() string {
	return fmt.Sprintf("fabrik: issue %s is %s", e.ID, e.Status)
}

type ErrIssueBlocked struct {
	ID        string
	BlockerID string
}

func (e *ErrIssueBlocked) Error() string {
	return fmt.Sprintf("fabrik: issue %s is blocked by %s", e.ID, e.BlockerID)
}

type ErrDepSelf struct{}

func (e *ErrDepSelf) Error() string {
	return "fabrik: an issue cannot block itself"
}

type ErrDepCycle struct {
	ID string
}

func (e *ErrDepCycle) Error() string {
	return fmt.Sprintf("fabrik: dep would cycle at %s", e.ID)
}

type ErrNoteTooLong struct{}

func (e *ErrNoteTooLong) Error() string {
	return "fabrik: note is too long"
}

type ErrBranchNotFound struct {
	Name string
}

func (e *ErrBranchNotFound) Error() string {
	return fmt.Sprintf("fabrik: branch %s not found", e.Name)
}

type ErrUnknownCommit struct {
	Ref string
}

func (e *ErrUnknownCommit) Error() string {
	return fmt.Sprintf("fabrik: unknown commit %s", e.Ref)
}

func StoreErr(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*sqlmigrate.ErrSchemaNewer); ok {
		return err
	}
	return fmt.Errorf("fabrik: store: %w", err)
}

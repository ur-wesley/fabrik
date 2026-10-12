package doltstore

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var issueIDRe = regexp.MustCompile(`^f-[0-9a-f]{8,32}$`)

var validIssueKinds = map[string]bool{
	"task": true, "bug": true, "feature": true, "epic": true, "chore": true,
}

func normalizeCreateInput(in *CreateInput) error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return fmt.Errorf("fabrik: title is required")
	}
	if utf8.RuneCountInString(in.Title) > 200 {
		return fmt.Errorf("fabrik: title is too long")
	}
	if in.Priority == nil {
		p := 2
		in.Priority = &p
	}
	if *in.Priority < 0 || *in.Priority > 4 {
		return fmt.Errorf("fabrik: priority must be 0..4")
	}
	if in.Kind == "" {
		in.Kind = "task"
	}
	if !validIssueKinds[in.Kind] {
		return fmt.Errorf("fabrik: kind must be task, bug, feature, epic, or chore")
	}
	return nil
}

func validateIssueID(id string) error {
	if !issueIDRe.MatchString(id) {
		return fmt.Errorf("fabrik: invalid issue id %q", id)
	}
	return nil
}

func validateAssignee(by string) error {
	if utf8.RuneCountInString(by) > 128 {
		return fmt.Errorf("fabrik: assignee is too long")
	}
	return nil
}

func validateCloseReason(reason string) error {
	if reason == "" {
		return nil
	}
	if utf8.RuneCountInString(reason) > 4000 {
		return fmt.Errorf("fabrik: reason is too long")
	}
	return nil
}

func validateNoteBody(body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("fabrik: note is empty")
	}
	if utf8.RuneCountInString(body) > 8000 {
		return &ErrNoteTooLong{}
	}
	return nil
}

func validateDepKind(kind string) error {
	if kind == "" {
		return nil
	}
	if kind != "blocks" && kind != "relates" {
		return fmt.Errorf("fabrik: kind must be blocks or relates")
	}
	return nil
}

package memory

import (
	"context"
	"strings"
)

type SavePromptOptions struct {
	Title         string
	CapturePrompt string
	Cwd           string
}

func SavePrompt(ctx context.Context, opt SavePromptOptions) (Observation, error) {
	title := strings.TrimSpace(opt.Title)
	if title == "" {
		title = "prompt"
	}
	body := strings.TrimSpace(opt.CapturePrompt)
	if body == "" {
		body = title
	}
	return Save(ctx, SaveOptions{
		Title: title,
		Body:  body,
		Kind:  "prompt",
		Cwd:   opt.Cwd,
	})
}

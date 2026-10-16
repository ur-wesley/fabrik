package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/ur-wesley/fabrik/cli/internal/export"
	"github.com/ur-wesley/fabrik/cli/internal/issuestore"
	"github.com/ur-wesley/fabrik/cli/internal/listen"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type listenIssueAPI struct{}

func (listenIssueAPI) Note(ctx context.Context, repo, issueID, body string) (listen.NoteJSON, error) {
	st, err := openStoreAt(ctx, repo)
	if err != nil {
		return listen.NoteJSON{}, err
	}
	defer st.Close()
	note, err := st.Note(ctx, issueID, body)
	if err != nil {
		return listen.NoteJSON{}, mapListenStoreErr(issueID, err)
	}
	if err := export.PublishAfterSave(ctx, st, issueID, "note"); err != nil {
		return listen.NoteJSON{}, err
	}
	return listen.NoteJSON{
		ID:        note.ID,
		IssueID:   note.IssueID,
		Body:      note.Body,
		CreatedAt: note.CreatedAt,
	}, nil
}

func (listenIssueAPI) ShowJSON(ctx context.Context, repo, issueID string) ([]byte, error) {
	st, err := openStoreAt(ctx, repo)
	if err != nil {
		return nil, err
	}
	defer st.Close()
	iss, err := st.Show(ctx, issueID)
	if err != nil {
		return nil, mapListenStoreErr(issueID, err)
	}
	return json.Marshal(iss)
}

func openStoreAt(ctx context.Context, repo string) (*issuestore.Store, error) {
	if !hasRepoStore(repo) {
		return nil, &listen.APIError{Status: http.StatusNotFound, Msg: "not found"}
	}
	st, err := issuestore.Open(ctx, repo)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &listen.APIError{Status: http.StatusNotFound, Msg: "not found"}
		}
		return nil, err
	}
	return st, nil
}

func hasRepoStore(repo string) bool {
	_, err := store.RequireRepo(repo)
	return err == nil
}

func mapListenStoreErr(issueID string, err error) error {
	if errors.Is(err, issuestore.ErrNotFound) {
		return &listen.APIError{Status: http.StatusNotFound, Msg: "not found"}
	}
	var tooLong *issuestore.ErrNoteTooLong
	if errors.As(err, &tooLong) {
		return &listen.APIError{Status: http.StatusBadRequest, Msg: "bad request"}
	}
	if err != nil && err.Error() == "fabrik: note is empty" {
		return &listen.APIError{Status: http.StatusBadRequest, Msg: "bad request"}
	}
	return issuestore.StoreErr(err)
}

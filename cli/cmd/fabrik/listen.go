package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/ur-wesley/fabrik/cli/internal/listen"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func newListenCmd() *cobra.Command {
	var addr string
	cmd := &cobra.Command{
		Use:   "listen",
		Short: "Loopback SSE for local GUI refresh",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := store.ListenJSONPath()
			if err != nil {
				return err
			}
			srv, err := listen.NewServer(addr, cfgPath, listenIssueAPI{})
			if errors.Is(err, listen.ErrAlreadyRunning) {
				return err
			}
			if err != nil {
				return err
			}

			cfg := srv.Config()
			line, err := json.Marshal(cfg)
			if err != nil {
				_ = srv.Close()
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(line))

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			errCh := make(chan error, 1)
			go func() { errCh <- srv.Serve() }()

			select {
			case <-sigCh:
				_ = srv.Close()
				return nil
			case err := <-errCh:
				_ = srv.Close()
				if err != nil && !errors.Is(err, http.ErrServerClosed) {
					return err
				}
				return nil
			}
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:0", "loopback listen address")
	return cmd
}

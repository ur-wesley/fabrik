// Command fabrik is the Fabrik machine-setup + repo-init CLI.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ur-wesley/fabrik/cli/internal/check"
	"github.com/ur-wesley/fabrik/cli/internal/clierr"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var u *clierr.UsageError
		if errors.As(err, &u) {
			os.Exit(2)
		}
		var ce *check.ExitError
		if errors.As(err, &ce) {
			os.Exit(ce.Code)
		}
		os.Exit(1)
	}
}

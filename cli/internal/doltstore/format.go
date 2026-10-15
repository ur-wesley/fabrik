package doltstore

import (
	"fmt"
	"io"
)

func PrintIssueLine(w io.Writer, i Issue) {
	fmt.Fprintf(w, "%s  %-7s p%d  %s  %s\n", i.ID, i.Status, i.Priority, i.Kind, i.Title)
}

func PrintShowText(w io.Writer, i Issue) {
	fmt.Fprintf(w, "%s  %s  p%d  %s\n", i.ID, i.Status, i.Priority, i.Kind)
	fmt.Fprintf(w, "%s\n\n", i.Title)
	if i.Body == "" {
		fmt.Fprintln(w, "(none)")
	} else {
		fmt.Fprintln(w, i.Body)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Acceptance:")
	if i.Acceptance == "" {
		fmt.Fprintln(w, "(none)")
	} else {
		fmt.Fprintln(w, i.Acceptance)
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Blocked by:")
	if len(i.Blockers) == 0 {
		fmt.Fprintln(w, "(none)")
	} else {
		for _, b := range i.Blockers {
			fmt.Fprintf(w, "%s  %s  %s\n", b.ID, b.Status, b.Title)
		}
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Blocks:")
	if len(i.Blocking) == 0 {
		fmt.Fprintln(w, "(none)")
	} else {
		for _, b := range i.Blocking {
			fmt.Fprintf(w, "%s  %s  %s\n", b.ID, b.Status, b.Title)
		}
	}
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Notes:")
	if len(i.Notes) == 0 {
		fmt.Fprintln(w, "(none)")
	} else {
		for _, n := range i.Notes {
			fmt.Fprintf(w, "%s  %s  %s\n", n.ID, n.CreatedAt, n.Body)
		}
	}
}

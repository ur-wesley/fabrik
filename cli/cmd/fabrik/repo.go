package main

import (
	"os"
)

var repoDirFlag string

func repoStartDir() string {
	if repoDirFlag != "" {
		return repoDirFlag
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

package main

import (
	"os"
	"path/filepath"
)

var (
	templDir string = ".templ"
)

func homePath() string {
	home := os.Getenv("TEMPL_HOME")
	if home != "" {
		return home
	}

	home = os.Getenv("HOME")
	if home != "" {
		return filepath.Join(home, templDir)
	}

	return ""
}

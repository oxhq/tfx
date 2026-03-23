package main

import (
	"os"
	"strings"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func versionString() string {
	version := Version
	if version == "dev" {
		if fileVersion := versionFromFile(); fileVersion != "" {
			version = fileVersion
		}
	}

	parts := []string{"tfx " + version}
	if trimmed := strings.TrimSpace(Commit); trimmed != "" && trimmed != "unknown" {
		parts = append(parts, "commit="+trimmed)
	}
	if trimmed := strings.TrimSpace(BuildDate); trimmed != "" && trimmed != "unknown" {
		parts = append(parts, "built="+trimmed)
	}
	return strings.Join(parts, " ")
}

func versionFromFile() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	path, err := findNearestFile(cwd, "VERSION")
	if err != nil {
		return ""
	}
	// #nosec G304 -- VERSION is resolved from the current workspace tree for local version fallback.
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

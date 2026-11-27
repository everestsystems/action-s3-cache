package main

import (
	"fmt"
	"io"
	"os"
)

const (
	gitHubOutputEnv = "GITHUB_OUTPUT"
	cacheHitOutput  = "cache-hit"
	appendOnly      = os.O_APPEND | os.O_WRONLY
	unusedFileMode  = os.FileMode(0)
)

func writeOutput(w io.Writer, name, value string) error {
	_, err := fmt.Fprintf(w, "%s=%s\n", name, value)
	return err
}

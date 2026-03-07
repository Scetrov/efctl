package main

import (
	"log"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra/doc"

	"efctl/cmd"
)

func main() {
	_, b, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("Failed to get runtime caller information")
	}
	basepath := filepath.Dir(b)
	docsPath := filepath.Join(basepath, "..", "..", "docs")

	root := cmd.GetRootCmd()
	root.DisableAutoGenTag = true
	err := doc.GenMarkdownTree(root, docsPath)
	if err != nil {
		log.Fatalf("Failed to generate docs: %v", err)
	}
	log.Printf("Successfully generated markdown docs in %s", docsPath)
}

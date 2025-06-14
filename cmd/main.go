package main

import (
	"fmt"
	"os"

	"github.com/eoinhurrell/mdnotes/cmd/root"
)

func main() {
	if err := root.NewRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
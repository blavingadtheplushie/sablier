// gendocs writes one md page per command
//
//	go run ./tools/gendocs -o docs/reference
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/espcaa/sablier/internal/cli"
	"github.com/spf13/cobra/doc"
)

func main() {
	out := flag.String("o", "docs/reference", "output directory")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	root := cli.NewRootCmd()
	root.DisableAutoGenTag = true // keeps output stable between runs

	if err := doc.GenMarkdownTree(root, *out); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote docs to %s\n", *out)
}

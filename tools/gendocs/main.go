// go run ./tools/gendocs -o website/src/content/docs/docs/reference/cli
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/espcaa/sablier/internal/cli"
	"github.com/spf13/cobra/doc"
)

func main() {
	out := flag.String("o", "website/src/content/docs/docs/reference/cli", "output directory")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}

	root := cli.NewRootCmd()
	root.DisableAutoGenTag = true // keeps output stable between runs

	front := func(file string) string {
		name := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(file), ".md"), "_", " ")
		return "---\ntitle: " + name + "\n---\n\n"
	}
	link := func(name string) string { return "../" + strings.TrimSuffix(name, ".md") + "/" }
	if err := doc.GenMarkdownTreeCustom(root, *out, front, link); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote docs to %s\n", *out)
}

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"sqldash/tools/format/css"
	"sqldash/tools/format/django"
)

func main() {
	check := flag.Bool("check", false, "Check formatting without modifying files")
	write := flag.Bool("write", false, "Format files in place")
	stdinPath := flag.String("stdin-filepath", "", "Format standard input using this file extension")
	flag.Parse()
	if *stdinPath != "" {
		source, readError := io.ReadAll(os.Stdin)
		if readError != nil {
			fail(readError)
		}
		formatted, formatError := format(*stdinPath, source)
		if formatError != nil {
			fail(formatError)
		}
		if *check {
			if !bytes.Equal(source, formatted) {
				fail(fmt.Errorf("%s needs formatting", *stdinPath))
			}
			return
		}
		if _, writeError := os.Stdout.Write(formatted); writeError != nil {
			fail(writeError)
		}
		return
	}
	if !*check && !*write {
		fail(fmt.Errorf("use --write or --check"))
	}
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"templates", "static/css"}
	}
	changed := false
	for _, path := range paths {
		walkError := filepath.WalkDir(path, func(path string, entry fs.DirEntry, visitError error) error {
			if visitError != nil {
				return visitError
			}
			if entry.IsDir() {
				switch entry.Name() {
				case ".git", "node_modules", "vendor", "data", "bin":
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if filepath.Ext(path) != ".django" && filepath.Ext(path) != ".css" {
				return nil
			}
			source, readError := os.ReadFile(path)
			if readError != nil {
				return readError
			}
			formatted, formatError := format(path, source)
			if formatError != nil {
				return fmt.Errorf("%s: %w", path, formatError)
			}
			if bytes.Equal(source, formatted) {
				return nil
			}
			changed = true
			fmt.Println(path)
			if *write {
				information, statError := entry.Info()
				if statError != nil {
					return statError
				}
				return os.WriteFile(path, formatted, information.Mode().Perm())
			}
			return nil
		})
		if walkError != nil {
			fail(walkError)
		}
	}
	if *check && changed {
		os.Exit(1)
	}
}

func format(path string, source []byte) ([]byte, error) {
	switch filepath.Ext(path) {
	case ".css":
		return css.Format(source)
	case ".django":
		return django.Format(source)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", path)
	}
}

func fail(failure error) {
	fmt.Fprintln(os.Stderr, failure)
	os.Exit(1)
}

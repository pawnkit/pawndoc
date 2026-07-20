package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pawnkit/pawn-project/fsx"
	"github.com/pawnkit/pawn-project/workspace"
	"github.com/pawnkit/pawndoc/doc"
	"github.com/pawnkit/pawndoc/render"
)

const (
	maxFiles     = 10_000
	maxFileSize  = 32 << 20
	maxTotalSize = 256 << 20
)

var version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("pawndoc", flag.ContinueOnError)
	flags.SetOutput(stderr)
	project := flags.String("project", ".", "Pawn project, directory, or source file")
	name := flags.String("name", "", "package name used in generated documentation")
	format := flags.String("format", "markdown", "output format: markdown, html, json, or search")
	strict := flags.Bool("strict", false, "fail when documentation diagnostics are reported")
	showVersion := flags.Bool("version", false, "print the pawndoc version")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		if _, err := fmt.Fprintln(stderr, "pawndoc: unexpected arguments"); err != nil {
			return 1
		}
		return 2
	}
	if *showVersion {
		if _, err := fmt.Fprintln(stdout, "pawndoc", version); err != nil {
			return 1
		}
		return 0
	}
	if !validFormat(*format) {
		if _, err := fmt.Fprintf(stderr, "pawndoc: unknown output format %q\n", *format); err != nil {
			return 1
		}
		return 2
	}

	root, selected, err := resolveProject(*project)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, "pawndoc:", err); writeErr != nil {
			return 1
		}
		return 1
	}
	inputs, err := collect(root, selected)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, "pawndoc:", err); writeErr != nil {
			return 1
		}
		return 1
	}
	packageName := strings.TrimSpace(*name)
	if packageName == "" {
		packageName = filepath.Base(root)
	}
	pkg, err := doc.Extract(context.Background(), packageName, inputs)
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, "pawndoc:", err); writeErr != nil {
			return 1
		}
		return 1
	}
	for _, issue := range pkg.Diagnostics {
		if _, err := fmt.Fprintf(stderr, "%s:%d: %s [%s]\n", issue.File, issue.Line, issue.Message, issue.Code); err != nil {
			return 1
		}
	}

	switch *format {
	case "markdown":
		err = render.Markdown(stdout, pkg)
	case "html":
		err = render.HTML(stdout, pkg)
	case "json":
		err = render.JSON(stdout, pkg)
	case "search":
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(render.SearchIndex(pkg))
	}
	if err != nil {
		if _, writeErr := fmt.Fprintln(stderr, "pawndoc:", err); writeErr != nil {
			return 1
		}
		return 1
	}
	if *strict && len(pkg.Diagnostics) > 0 {
		return 1
	}
	return 0
}

func validFormat(value string) bool {
	switch value {
	case "markdown", "html", "json", "search":
		return true
	default:
		return false
	}
}

func resolveProject(path string) (root, selected string, err error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "", fmt.Errorf("resolve project path: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", "", fmt.Errorf("inspect project path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", errors.New("project path must not be a symbolic link")
	}
	if !info.IsDir() {
		if !pawnFile(absolute) {
			return "", "", errors.New("project file must end in .pwn or .inc")
		}
		return filepath.Dir(absolute), absolute, nil
	}
	discovered, findErr := workspace.FindRoot(fsx.OS{}, absolute)
	if findErr == nil {
		return filepath.FromSlash(discovered.Dir), "", nil
	}
	if !errors.Is(findErr, workspace.ErrNotFound) {
		return "", "", findErr
	}
	return absolute, "", nil
}

func collect(root, selected string) ([]doc.Input, error) {
	inputs := make([]doc.Input, 0)
	var total int64
	add := func(path string, entry os.DirEntry) error {
		if entry.Type()&os.ModeSymlink != 0 || !pawnFile(path) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if len(inputs) >= maxFiles {
			return fmt.Errorf("project contains more than %d Pawn files", maxFiles)
		}
		if info.Size() > maxFileSize {
			return fmt.Errorf("%s exceeds the %d MiB file limit", path, maxFileSize>>20)
		}
		file, err := os.Open(path) //nolint:gosec // collect validates project paths.
		if err != nil {
			return err
		}
		text, readErr := io.ReadAll(io.LimitReader(file, maxFileSize+1))
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		if len(text) > maxFileSize {
			return fmt.Errorf("%s exceeds the %d MiB file limit", path, maxFileSize>>20)
		}
		total += int64(len(text))
		if total > maxTotalSize {
			return fmt.Errorf("project exceeds the %d MiB source limit", maxTotalSize>>20)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("source path %q is outside the project", path)
		}
		inputs = append(inputs, doc.Input{Path: filepath.ToSlash(relative), Text: text})
		return nil
	}

	if selected != "" {
		entry, err := os.Stat(selected)
		if err != nil {
			return nil, err
		}
		if err := add(selected, fileEntry{FileInfo: entry}); err != nil {
			return nil, err
		}
	} else {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				switch entry.Name() {
				case ".git", ".pawn", "vendor", "node_modules":
					if path != root {
						return filepath.SkipDir
					}
				}
				return nil
			}
			return add(path, entry)
		})
		if err != nil {
			return nil, err
		}
	}
	if len(inputs) == 0 {
		return nil, errors.New("no .pwn or .inc files found")
	}
	return inputs, nil
}

func pawnFile(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".pwn" || extension == ".inc"
}

type fileEntry struct{ os.FileInfo }

func (entry fileEntry) Type() os.FileMode          { return entry.Mode().Type() }
func (entry fileEntry) Info() (os.FileInfo, error) { return entry.FileInfo, nil }

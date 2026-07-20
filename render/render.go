// Package render writes pawndoc output formats.
package render

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"

	"github.com/pawnkit/pawndoc/doc"
)

// JSON writes the renderer-neutral model.
func JSON(w io.Writer, pkg doc.Package) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(pkg)
}

// Markdown writes a Markdown document.
func Markdown(w io.Writer, pkg doc.Package) error {
	if _, err := fmt.Fprintf(w, "# %s\n\n", inlineMarkdown(pkg.Name)); err != nil {
		return err
	}
	for _, symbol := range pkg.Symbols {
		if err := writeMarkdownSymbol(w, symbol); err != nil {
			return err
		}
	}
	return nil
}

func writeMarkdownSymbol(w io.Writer, symbol doc.Symbol) error {
	if _, err := fmt.Fprintf(w, "## %s\n\n", inlineMarkdown(symbol.Name)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "`%s` · `%s:%d`\n\n", inlineMarkdown(symbol.Kind), inlineMarkdown(symbol.File), symbol.Line); err != nil {
		return err
	}
	if symbol.Library != "" {
		if _, err := fmt.Fprintf(w, "**Library:** `%s`\n\n", inlineMarkdown(symbol.Library)); err != nil {
			return err
		}
	}
	if symbol.Deprecated != "" {
		if _, err := fmt.Fprintf(w, "**Deprecated:** %s\n\n", markdown(symbol.Deprecated)); err != nil {
			return err
		}
	}
	if symbol.Summary != "" {
		if _, err := fmt.Fprintf(w, "%s\n\n", markdown(symbol.Summary)); err != nil {
			return err
		}
	}
	if symbol.Remarks != "" {
		if _, err := fmt.Fprintf(w, "### Remarks\n\n%s\n\n", markdown(symbol.Remarks)); err != nil {
			return err
		}
	}
	if len(symbol.Parameters) > 0 {
		if _, err := fmt.Fprintln(w, "### Parameters"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		for _, name := range sortedKeys(symbol.Parameters) {
			if _, err := fmt.Fprintf(w, "- `%s`: %s\n", inlineMarkdown(name), markdown(symbol.Parameters[name])); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	if symbol.Returns != "" {
		if _, err := fmt.Fprintf(w, "### Returns\n\n%s\n\n", markdown(symbol.Returns)); err != nil {
			return err
		}
	}
	for i, example := range symbol.Examples {
		heading := "Example"
		if len(symbol.Examples) > 1 {
			heading = fmt.Sprintf("Example %d", i+1)
		}
		if _, err := fmt.Fprintf(w, "### %s\n\n```pawn\n%s\n```\n\n", heading, codeBlock(example)); err != nil {
			return err
		}
	}
	if symbol.Since != "" {
		if _, err := fmt.Fprintf(w, "**Since:** %s\n\n", markdown(symbol.Since)); err != nil {
			return err
		}
	}
	if len(symbol.SeeAlso) > 0 {
		items := make([]string, len(symbol.SeeAlso))
		for i, item := range symbol.SeeAlso {
			items[i] = "`" + inlineMarkdown(item) + "`"
		}
		if _, err := fmt.Fprintf(w, "**See also:** %s\n\n", strings.Join(items, ", ")); err != nil {
			return err
		}
	}
	return nil
}

func markdown(value string) string {
	return strings.NewReplacer(
		"\\", "\\\\",
		"`", "\\`",
		"*", "\\*",
		"_", "\\_",
		"[", "\\[",
		"]", "\\]",
		"<", "&lt;",
		">", "&gt;",
	).Replace(value)
}

func inlineMarkdown(value string) string {
	return markdown(strings.NewReplacer("\r", " ", "\n", " ").Replace(value))
}

func codeBlock(value string) string {
	return strings.ReplaceAll(value, "```", "` ` `")
}

type parameter struct {
	Name string
	Text string
}

type htmlSymbol struct {
	doc.Symbol
	Parameters []parameter
}

type htmlPage struct {
	Name    string
	Symbols []htmlSymbol
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>{{.Name}}</title></head>
<body>
<main>
<h1>{{.Name}}</h1>
{{range .Symbols}}<section id="{{.ID}}">
<h2>{{.Name}}</h2>
<p><code>{{.Kind}}</code> · <code>{{.File}}:{{.Line}}</code></p>
{{if .Library}}<p><strong>Library:</strong> <code>{{.Library}}</code></p>{{end}}
{{if .Deprecated}}<p><strong>Deprecated:</strong> {{.Deprecated}}</p>{{end}}
{{if .Summary}}<p>{{.Summary}}</p>{{end}}
{{if .Remarks}}<h3>Remarks</h3><p>{{.Remarks}}</p>{{end}}
{{if .Parameters}}<h3>Parameters</h3><dl>{{range .Parameters}}<dt><code>{{.Name}}</code></dt><dd>{{.Text}}</dd>{{end}}</dl>{{end}}
{{if .Returns}}<h3>Returns</h3><p>{{.Returns}}</p>{{end}}
{{range .Examples}}<h3>Example</h3><pre><code>{{.}}</code></pre>{{end}}
{{if .Since}}<p><strong>Since:</strong> {{.Since}}</p>{{end}}
{{if .SeeAlso}}<p><strong>See also:</strong> {{range $index, $item := .SeeAlso}}{{if $index}}, {{end}}<code>{{$item}}</code>{{end}}</p>{{end}}
</section>{{end}}
</main>
</body>
</html>
`))

// HTML writes a standalone HTML document.
func HTML(w io.Writer, pkg doc.Package) error {
	page := htmlPage{Name: pkg.Name, Symbols: make([]htmlSymbol, 0, len(pkg.Symbols))}
	for _, symbol := range pkg.Symbols {
		view := htmlSymbol{Symbol: symbol}
		for _, name := range sortedKeys(symbol.Parameters) {
			view.Parameters = append(view.Parameters, parameter{Name: name, Text: symbol.Parameters[name]})
		}
		page.Symbols = append(page.Symbols, view)
	}
	return pageTemplate.Execute(w, page)
}

// SearchEntry is a compact symbol record for search indexes.
type SearchEntry struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Summary string `json:"summary,omitempty"`
}

// SearchIndex builds sorted search records.
func SearchIndex(pkg doc.Package) []SearchEntry {
	entries := make([]SearchEntry, 0, len(pkg.Symbols))
	for _, symbol := range pkg.Symbols {
		entries = append(entries, SearchEntry{ID: symbol.ID, Name: symbol.Name, Kind: symbol.Kind, Summary: symbol.Summary})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

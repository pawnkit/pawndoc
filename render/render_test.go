package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pawnkit/pawndoc/doc"
	"github.com/pawnkit/pawndoc/render"
)

func TestHTMLEscapesDocumentation(t *testing.T) {
	pkg := doc.Package{Name: "<pkg>", Symbols: []doc.Symbol{{
		ID:         "function:x",
		Name:       "x",
		Kind:       "function",
		File:       "main.pwn",
		Line:       3,
		Summary:    "a < b",
		Remarks:    "more <script>alert(1)</script>",
		Parameters: map[string]string{"value": "a & b"},
		Returns:    "the result",
		Examples:   []string{"x(<value>);"},
		SeeAlso:    []string{"y"},
	}}}
	var output bytes.Buffer
	if err := render.HTML(&output, pkg); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "a < b") || !strings.Contains(output.String(), "a &lt; b") {
		t.Fatalf("html = %q", output.String())
	}
	for _, expected := range []string{"Remarks", "Parameters", "Returns", "Example", "See also"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("html missing %q: %s", expected, output.String())
		}
	}
}

func TestMarkdownIncludesDocumentationFields(t *testing.T) {
	pkg := doc.Package{Name: "pkg", Symbols: []doc.Symbol{{
		ID:         "native:x",
		Name:       "x",
		Kind:       "native",
		File:       "api.inc",
		Line:       4,
		Summary:    "Read <value>.",
		Parameters: map[string]string{"value": "Input value."},
		Returns:    "The result.",
		Deprecated: "Use y.",
		Since:      "1.2",
	}}}
	var output bytes.Buffer
	if err := render.Markdown(&output, pkg); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"**Deprecated:**", "### Parameters", "### Returns", "**Since:**", "&lt;value&gt;"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("markdown missing %q: %s", expected, output.String())
		}
	}
}

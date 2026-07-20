package doc

import (
	"context"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	analysis "github.com/pawnkit/pawn-analysis"
	"github.com/pawnkit/pawnkit-core/source"
)

// Input is one Pawn source file.
type Input struct {
	Path string
	Text []byte
}

// Extract builds documentation from Pawn source files.
func Extract(ctx context.Context, name string, inputs []Input) (Package, error) {
	result := Package{SchemaVersion: 1, Name: name}
	ids := make(map[string]bool)
	for _, input := range inputs {
		if err := ctx.Err(); err != nil {
			return Package{}, err
		}
		result.Files = append(result.Files, input.Path)
		analysisResult, err := analysis.AnalyzeContext(ctx, input.Text, analysis.Options{URI: source.FileURI(input.Path)})
		if err != nil {
			return Package{}, err
		}
		lineStarts := starts(input.Text)
		for _, item := range analysisResult.Symbols.Symbols {
			if !documentable(item.Kind.String()) {
				continue
			}
			line := lineAt(lineStarts, int(item.Span.Start))
			comment := precedingComment(input.Text, int(item.Span.Start))
			parsed, parseErr := parseComment(comment)
			id := item.Kind.String() + ":" + item.Name
			symbol := Symbol{ID: id, Name: item.Name, Kind: item.Kind.String(), File: input.Path, Line: line}
			if parseErr != nil {
				result.Diagnostics = append(result.Diagnostics, Issue{Code: "pawndoc/malformed-comment", Message: parseErr.Error(), File: input.Path, Line: line})
			} else {
				applyComment(&symbol, parsed)
			}
			if comment == "" && (item.Kind.String() == "public" || item.Kind.String() == "native") {
				result.Diagnostics = append(result.Diagnostics, Issue{Code: "pawndoc/undocumented-public", Message: item.Name + " has no documentation", File: input.Path, Line: line})
			}
			if ids[id] {
				result.Diagnostics = append(result.Diagnostics, Issue{Code: "pawndoc/duplicate-id", Message: "duplicate symbol " + id, File: input.Path, Line: line})
				continue
			}
			ids[id] = true
			result.Symbols = append(result.Symbols, symbol)
		}
	}
	sort.Strings(result.Files)
	sort.Slice(result.Symbols, func(i, j int) bool { return result.Symbols[i].ID < result.Symbols[j].ID })
	return result, nil
}

type xmlDoc struct {
	Library    xmlText    `xml:"library"`
	Summary    xmlText    `xml:"summary"`
	Remarks    []xmlText  `xml:"remarks"`
	Returns    xmlText    `xml:"returns"`
	Return     xmlText    `xml:"return"`
	Deprecated xmlText    `xml:"deprecated"`
	Since      xmlText    `xml:"since"`
	Params     []xmlParam `xml:"param"`
	Examples   []xmlText  `xml:"example"`
	SeeAlso    []xmlRef   `xml:"seealso"`
}

type xmlParam struct {
	Name string `xml:"name,attr"`
	Text xmlText
}

func (p *xmlParam) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	for _, attribute := range start.Attr {
		if attribute.Name.Local == "name" {
			p.Name = attribute.Value
		}
	}
	text, err := readXMLText(decoder, start)
	p.Text = xmlText(text)
	return err
}

type xmlRef struct {
	Name string `xml:"name,attr"`
	CRef string `xml:"cref,attr"`
}

type xmlText string

func (t *xmlText) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	text, err := readXMLText(decoder, start)
	*t = xmlText(text)
	return err
}

func readXMLText(decoder *xml.Decoder, start xml.StartElement) (string, error) {
	var text strings.Builder
	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		switch value := token.(type) {
		case xml.StartElement:
			if blockXML(value.Name.Local) && text.Len() > 0 {
				text.WriteByte('\n')
			}
			depth++
		case xml.EndElement:
			depth--
			if depth > 0 && blockXML(value.Name.Local) {
				text.WriteByte('\n')
			}
		case xml.CharData:
			text.Write(value)
		}
	}
	return strings.TrimSpace(text.String()), nil
}

func blockXML(name string) bool {
	switch name {
	case "p", "para", "li", "item", "br":
		return true
	default:
		return false
	}
}

func parseComment(comment string) (xmlDoc, error) {
	if comment == "" {
		return xmlDoc{}, nil
	}
	var parsed xmlDoc
	if !strings.Contains(comment, "<") {
		parsed.Summary = xmlText(strings.TrimSpace(comment))
		return parsed, nil
	}
	if err := xml.Unmarshal([]byte("<doc>"+comment+"</doc>"), &parsed); err != nil {
		return xmlDoc{}, fmt.Errorf("invalid documentation XML: %w", err)
	}
	return parsed, nil
}

func applyComment(symbol *Symbol, parsed xmlDoc) {
	symbol.Library = strings.TrimSpace(string(parsed.Library))
	symbol.Summary = strings.TrimSpace(string(parsed.Summary))
	remarks := make([]string, 0, len(parsed.Remarks))
	for _, remark := range parsed.Remarks {
		if text := strings.TrimSpace(string(remark)); text != "" {
			remarks = append(remarks, text)
		}
	}
	symbol.Remarks = strings.Join(remarks, "\n\n")
	symbol.Returns = strings.TrimSpace(string(parsed.Returns))
	if symbol.Returns == "" {
		symbol.Returns = strings.TrimSpace(string(parsed.Return))
	}
	symbol.Deprecated = strings.TrimSpace(string(parsed.Deprecated))
	symbol.Since = strings.TrimSpace(string(parsed.Since))
	for _, parameter := range parsed.Params {
		if symbol.Parameters == nil {
			symbol.Parameters = make(map[string]string)
		}
		name := strings.TrimSpace(parameter.Name)
		if name != "" {
			symbol.Parameters[name] = strings.TrimSpace(string(parameter.Text))
		}
	}
	for _, example := range parsed.Examples {
		if text := strings.TrimSpace(string(example)); text != "" {
			symbol.Examples = append(symbol.Examples, text)
		}
	}
	for _, reference := range parsed.SeeAlso {
		name := strings.TrimSpace(reference.Name)
		if name == "" {
			name = strings.TrimSpace(reference.CRef)
		}
		if name != "" {
			symbol.SeeAlso = append(symbol.SeeAlso, name)
		}
	}
}

func documentable(kind string) bool { return kind != "parameter" && kind != "variable" }

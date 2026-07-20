// Package doc extracts Pawn documentation into a renderer-neutral model.
package doc

// Package is the documentation for one Pawn package.
type Package struct {
	SchemaVersion int      `json:"schemaVersion"`
	Name          string   `json:"name"`
	Files         []string `json:"files"`
	Symbols       []Symbol `json:"symbols"`
	Diagnostics   []Issue  `json:"diagnostics,omitempty"`
}

// Symbol describes one documented declaration.
type Symbol struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Kind       string            `json:"kind"`
	Library    string            `json:"library,omitempty"`
	File       string            `json:"file"`
	Line       int               `json:"line"`
	Summary    string            `json:"summary,omitempty"`
	Remarks    string            `json:"remarks,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Returns    string            `json:"returns,omitempty"`
	Deprecated string            `json:"deprecated,omitempty"`
	Since      string            `json:"since,omitempty"`
	Examples   []string          `json:"examples,omitempty"`
	SeeAlso    []string          `json:"seeAlso,omitempty"`
}

// Issue reports a documentation problem.
type Issue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	File    string `json:"file"`
	Line    int    `json:"line"`
}

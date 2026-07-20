package doc_test

import (
	"context"
	"testing"

	"github.com/pawnkit/pawndoc/doc"
)

func TestExtractXMLAndLineComments(t *testing.T) {
	source := `/// <summary>Starts the mode.</summary>
public OnGameModeInit() { return 1; }
/**
 * <summary>Add values.</summary>
 * <param name="left">First value.</param>
 * <returns>The sum.</returns>
 */
stock Add(left, right) { return left + right; }
`
	pkg, err := doc.Extract(context.Background(), "example", []doc.Input{{Path: "main.pwn", Text: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Symbols) != 2 {
		t.Fatalf("symbols = %#v", pkg.Symbols)
	}
	if pkg.Symbols[0].Name == "Add" && pkg.Symbols[0].Parameters["left"] != "First value." {
		t.Fatalf("parameters = %#v", pkg.Symbols[0].Parameters)
	}
}

func TestExtractDiagnosesMalformedXML(t *testing.T) {
	source := "/// <summary>broken\npublic Start() {}\n"
	pkg, err := doc.Extract(context.Background(), "example", []doc.Input{{Path: "main.pwn", Text: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Diagnostics) == 0 || pkg.Diagnostics[0].Code != "pawndoc/malformed-comment" {
		t.Fatalf("diagnostics = %#v", pkg.Diagnostics)
	}
}

func TestExtractPreservesNestedTextAndReferences(t *testing.T) {
	source := `/**
 * <summary>Read the <c>player</c> name.</summary>
 * <library>players</library>
 * <remarks>Uses the current player state.</remarks>
 * <remarks>Returns an empty value when unset.</remarks>
 * <param name="playerid">Player to read.</param>
 * <return>The <b>display</b> name.</return>
 * <seealso name="SetPlayerName"/>
 */
native GetPlayerName(playerid);
`
	pkg, err := doc.Extract(context.Background(), "example", []doc.Input{{Path: "names.inc", Text: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Symbols) != 1 {
		t.Fatalf("symbols = %#v", pkg.Symbols)
	}
	symbol := pkg.Symbols[0]
	if symbol.Summary != "Read the player name." {
		t.Fatalf("summary = %q", symbol.Summary)
	}
	if symbol.Library != "players" {
		t.Fatalf("library = %q", symbol.Library)
	}
	if symbol.Remarks != "Uses the current player state.\n\nReturns an empty value when unset." {
		t.Fatalf("remarks = %q", symbol.Remarks)
	}
	if symbol.Parameters["playerid"] != "Player to read." {
		t.Fatalf("parameters = %#v", symbol.Parameters)
	}
	if symbol.Returns != "The display name." {
		t.Fatalf("returns = %q", symbol.Returns)
	}
	if len(symbol.SeeAlso) != 1 || symbol.SeeAlso[0] != "SetPlayerName" {
		t.Fatalf("see also = %#v", symbol.SeeAlso)
	}
}

func TestExtractDoesNotCrossRegularBlockComment(t *testing.T) {
	source := `/** <summary>Wrong comment.</summary> */
/* an intervening comment */
native Undocumented();
`
	pkg, err := doc.Extract(context.Background(), "example", []doc.Input{{Path: "api.inc", Text: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Symbols) != 1 || pkg.Symbols[0].Summary != "" {
		t.Fatalf("symbols = %#v", pkg.Symbols)
	}
}

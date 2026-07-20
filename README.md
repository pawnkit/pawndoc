# pawndoc

`pawndoc` turns Pawn comments into Markdown, HTML, JSON, or a compact search
index. It understands the XML-style comments already used by Pawn libraries, so
projects do not need to adopt another documentation format.

## Install

```sh
go install github.com/pawnkit/pawndoc/cmd/pawndoc@latest
```

## Use it

Run it from a Pawn project:

```sh
pawndoc --format markdown > API.md
```

You can also point it at a directory or a single source file:

```sh
pawndoc --project pawno/include/time.inc --format html > time.html
pawndoc --project filterscripts --format json > docs.json
```

The supported formats are `markdown`, `html`, `json`, and `search`. Add
`--strict` in CI when malformed or missing public documentation should fail the
command.

Both `///` comments and `/** */` blocks are accepted. Common tags include
`summary`, `remarks`, `param`, `returns`, `example`, `deprecated`, `since`, and
`seealso`. Nested formatting tags are kept as readable text.

## Library use

The `doc` package extracts a renderer-neutral model. The `render` package writes
the formats used by the CLI. This split lets other PawnKit tools reuse extracted
documentation without parsing generated HTML.

See [docs/decision.md](docs/decision.md) for the compatibility decision and
[CONTRIBUTING.md](CONTRIBUTING.md) if you want to help.

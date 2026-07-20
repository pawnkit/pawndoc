# Contributing

PawnKit is maintained by volunteers, so reviews may take a little time.

Contributions are welcome. A parser or renderer fix should include a short Pawn
comment fixture and the expected model or output.

Run the local checks before opening a pull request:

```sh
go test ./...
go vet ./...
CGO_ENABLED=1 go test -race ./...
```

Keep extraction separate from rendering. Compatibility with existing Pawndoc
comments matters more than inventing a new documentation syntax here.

# Why pawndoc uses the PawnKit parser

The original `pawn-lang/pawndoc` workflow documents XML emitted by the compiler.
That remains useful compatibility material, but compiler output is awkward to
reuse in editors, package tools, and the PawnKit website. It also inherits known
compiler problems around enums, macros, and unused declarations.

This implementation reads source through the PawnKit parser and analysis
libraries. It keeps the established XML comment tags while producing a small Go
model that any renderer can consume.

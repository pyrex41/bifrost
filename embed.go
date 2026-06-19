package main

import "embed"

// Embedded defaults so the binary is self-contained for `go install` and
// release downloads. adapters.json holds machine-specific launcher paths, so it
// is only a fallback: resolution is $BIFROST_ADAPTERS -> ./adapters.json ->
// this embedded copy (see findAdaptersBytes). The corpus (cases/programs) and
// the worked examples ride along so the differential matrix runs out of the box.
//
//go:embed adapters.json
var embeddedAdapters []byte

//go:embed cases
var embeddedCases embed.FS

//go:embed programs
var embeddedPrograms embed.FS

//go:embed examples
var embeddedExamples embed.FS

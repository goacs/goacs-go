package scripts

import _ "embed"

// ReferenceDoc is the full Lua provisioning-script API reference (README.md), embedded at
// build time so it ships with the binary and can be fed as-is into an LLM system prompt -
// it stays in sync with the actual API automatically, since it's the same file documenting
// engine.go/functions.go/bridge.go for human readers.
//
//go:embed README.md
var ReferenceDoc string

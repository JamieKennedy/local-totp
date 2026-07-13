// Package apispec embeds the canonical OpenAPI interface document.
package apispec

import _ "embed"

//go:embed openapi.json
var document []byte

// Document returns an immutable copy of the OpenAPI document.
func Document() []byte { return append([]byte(nil), document...) }

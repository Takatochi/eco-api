package docs

import _ "embed"

//go:embed openapi.json
var openAPISpec []byte

func OpenAPISpec() []byte {
	return openAPISpec
}

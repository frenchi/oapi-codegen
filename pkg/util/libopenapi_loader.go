package util

import (
	"github.com/getkin/kin-openapi/openapi3"
)

// loadWithLibopenapi is a feature-flagged stub that currently delegates to the
// legacy kin-openapi loader. A future change will replace this with
// libopenapi-based parsing plus conversion to *openapi3.T while preserving
// backwards compatibility.
func loadWithLibopenapi(filePath string) (*openapi3.T, error) {
	return loadWithKin(filePath)
}

// Package parser provides a thin, version-aware adapter from an OpenAPI document
// (loaded via a chosen parser) into the internal IR. For illustration purposes,
// this stub only contains the entry-points and interfaces you would implement.
package parser

import (
	"fmt"
	"strings"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/ir"
)

// Detector peeks at the raw bytes to determine the OpenAPI version. In production,
// a tiny YAML decoder (yaml.v3) that extracts only the "openapi" field is sufficient.
func DetectVersion(raw []byte) ir.SpecVersion {
	// Very small heuristic for the stub: look for "openapi: 3.1" or "\"openapi\": "3.1"
	b := string(raw)
	if strings.Contains(b, "openapi: 3.1") || strings.Contains(b, "\"openapi\": \"3.1") {
		return ir.Spec3_1
	}
	if strings.Contains(b, "openapi: 3.0") || strings.Contains(b, "\"openapi\": \"3.0") {
		return ir.Spec3_0
	}
	return ir.SpecUnknown
}

// ParseToIR is the single entry-point the code generator will use. It dispatches to
// a version-specific parser but returns a unified IR either way.
func ParseToIR(raw []byte) (*ir.Document, error) {
	ver := DetectVersion(raw)
	switch ver {
	case ir.Spec3_0:
		return parse30(raw)
	case ir.Spec3_1:
		return parse31(raw)
	default:
		return nil, fmt.Errorf("unsupported or undetected OpenAPI version")
	}
}

// parse30 parses a 3.0.x spec into the IR. In the final design you would use a single
// parser (e.g., libopenapi) for both 3.0 and 3.1, then adapt into IR here.
func parse30(raw []byte) (*ir.Document, error) {
	// Stub: return a minimal document to demonstrate call flow.
	return &ir.Document{Version: ir.Spec3_0}, nil
}

// parse31 parses a 3.1.x spec into the IR. This is where 3.1-specific handling like
// nullability via type: [T, "null"], webhooks, and JSON Schema 2020-12 peculiarities
// would be normalized into the same IR consumed by templates.
func parse31(raw []byte) (*ir.Document, error) {
	// Stub: return a minimal document to demonstrate call flow.
	return &ir.Document{Version: ir.Spec3_1}, nil
}


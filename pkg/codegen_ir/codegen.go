// Package codegen_ir contains the minimal front door for the IR-driven generator.
// It exists alongside the current kin-openapi-based generator to illustrate how the
// final shape would look while maintaining backwards compatibility during migration.
package codegen_ir

import (
	"fmt"

	"github.com/oapi-codegen/oapi-codegen/v2/pkg/ir"
)

// Generate consumes the IR and produces code. For now, we stub the output just to
// show how it would be invoked.
func Generate(doc *ir.Document) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("nil IR document")
	}
	// In the real implementation, this package would:
	// - derive OperationDefinition, Schema, and other structures (replacing direct
	//   kin-openapi coupling), or translate IR directly to the existing templates via
	//   a small compatibility layer.
	// - render templates into the final code output.
	return fmt.Sprintf("// generated for version: %v\n", doc.Version), nil
}


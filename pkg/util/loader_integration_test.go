package util

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a small helper to create a file with content.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func TestLoadSwagger_File_StrictAndSoft(t *testing.T) {
	dir := t.TempDir()

	// ext.yaml provides a component referenced by main.yaml
	ext := `openapi: 3.0.0
info: {title: ext, version: v}
paths: {}
components:
  schemas:
    Bar:
      type: string
`
	_ = writeFile(t, dir, "ext.yaml", ext)

	// main.yaml references ext.yaml component using a relative $ref
	main := `openapi: 3.0.0
info: {title: main, version: v}
paths: {}
components:
  schemas:
    Foo:
      $ref: './ext.yaml#/components/schemas/Bar'
`
	mainPath := writeFile(t, dir, "main.yaml", main)

	// invalid.yaml is intentionally malformed YAML to trigger strict errors
	invalidPath := writeFile(t, dir, "invalid.yaml", "not: [valid")

	cases := []struct {
		name    string
		env     string // OAPI_CODEGEN_USE_LIBOPENAPI
		path    string
		wantErr bool // for soft+invalid we accept either path; set wantErr=true to require error
	}{
		{name: "strict valid file", env: "strict", path: mainPath, wantErr: false},
		{name: "strict invalid file", env: "strict", path: invalidPath, wantErr: true},
		{name: "soft valid file", env: "1", path: mainPath, wantErr: false},
		// For soft+invalid, behavior depends on whether kin can parse; we don't force a fallback here.
		// Use a clearly malformed YAML so both libopenapi and kin should error => expect error.
		{name: "soft invalid file", env: "1", path: invalidPath, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OAPI_CODEGEN_USE_LIBOPENAPI", tc.env)
			spec, err := LoadSwagger(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got spec=%v", spec != nil)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if spec == nil {
				t.Fatalf("got nil spec")
			}
		})
	}
}

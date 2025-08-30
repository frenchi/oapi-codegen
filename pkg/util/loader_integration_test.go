package util

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	libopenapi "github.com/pb33f/libopenapi"
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

func TestLoadSwagger_File(t *testing.T) {
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
		name        string
		env         string // OAPI_CODEGEN_USE_LIBOPENAPI
		path        string
		wantErr     bool   // for soft+invalid we accept either path; set wantErr=true to require error
		expStrategy string // expected strategy from onLoaderDecision (optional)
	}{
		// kin (unset/0)
		{name: "kin valid file (unset)", env: "", path: mainPath, wantErr: false, expStrategy: "kin:file"},
		{name: "kin invalid file (unset)", env: "", path: invalidPath, wantErr: true, expStrategy: "kin:file"},
		{name: "kin valid file (0)", env: "0", path: mainPath, wantErr: false, expStrategy: "kin:file"},
		{name: "kin invalid file (0)", env: "0", path: invalidPath, wantErr: true, expStrategy: "kin:file"},
		// libopenapi strict
		{name: "strict valid file", env: "strict", path: mainPath, wantErr: false, expStrategy: "libopenapi:file"},
		{name: "strict invalid file", env: "strict", path: invalidPath, wantErr: true, expStrategy: "libopenapi:file"},
		// libopenapi soft
		{name: "soft valid file", env: "1", path: mainPath, wantErr: false, expStrategy: "libopenapi:file"},
		// For fallback case, we test in TestLoadSwagger_File_ForcedFallback.
		// Use a clearly malformed YAML so both libopenapi and kin should error => expect error.
		{name: "soft invalid file", env: "1", path: invalidPath, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldHook := onLoaderDecision
			strategy := ""
			onLoaderDecision = func(d string) { strategy = d }
			t.Cleanup(func() { onLoaderDecision = oldHook })

			t.Setenv("OAPI_CODEGEN_USE_LIBOPENAPI", tc.env)
			spec, err := LoadSwagger(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got spec=%v", spec != nil)
				}
				if tc.expStrategy != "" && strategy != tc.expStrategy {
					t.Fatalf("expected strategy decision %q, got %q", tc.expStrategy, strategy)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expStrategy != "" && strategy != tc.expStrategy {
				t.Fatalf("expected strategy decision %q, got %q", tc.expStrategy, strategy)
			}
			if spec == nil || spec.OpenAPI == "" {
				t.Fatalf("expected non-nil spec with OpenAPI set")
			}
			if tc.path == mainPath {
				if spec.Components == nil || spec.Components.Schemas == nil || spec.Components.Schemas["Foo"] == nil {
					t.Fatalf("expected components.schemas[Foo] to exist for external ref test")
				}
			}
		})
	}
}

func TestLoadSwagger_URI(t *testing.T) {
	// valid minimal 3.0 spec
	spec := []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n")
	bad := []byte("not: [valid")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		switch r.URL.Path {
		case "/spec.yaml":
			_, _ = w.Write(spec)
		case "/bad.yaml":
			_, _ = w.Write(bad)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	cases := []struct {
		name        string
		env         string
		path        string
		wantErr     bool
		expStrategy string
	}{
		// kin (unset/0)
		{name: "kin valid uri (unset)", env: "", path: ts.URL + "/spec.yaml", wantErr: false, expStrategy: "kin:uri"},
		{name: "kin invalid uri (unset)", env: "", path: ts.URL + "/bad.yaml", wantErr: true, expStrategy: "kin:uri"},
		{name: "kin valid uri (0)", env: "0", path: ts.URL + "/spec.yaml", wantErr: false, expStrategy: "kin:uri"},
		{name: "kin invalid uri (0)", env: "0", path: ts.URL + "/bad.yaml", wantErr: true, expStrategy: "kin:uri"},
		// lib strict/soft
		{name: "strict valid uri", env: "strict", path: ts.URL + "/spec.yaml", wantErr: false, expStrategy: "libopenapi:uri"},
		{name: "strict invalid uri", env: "strict", path: ts.URL + "/bad.yaml", wantErr: true, expStrategy: "libopenapi:uri"},
		// libopenapi soft
		{name: "soft valid uri", env: "1", path: ts.URL + "/spec.yaml", wantErr: false, expStrategy: "libopenapi:uri"},
		// for soft+invalid, we force the fallback and test in TestLoadSwagger_URI_ForcedFallback
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			oldHook := onLoaderDecision
			strategy := ""
			onLoaderDecision = func(d string) { strategy = d }
			t.Cleanup(func() { onLoaderDecision = oldHook })

			t.Setenv("OAPI_CODEGEN_USE_LIBOPENAPI", tc.env)
			sw, err := LoadSwagger(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got spec=%v", sw != nil)
				}
				if tc.expStrategy != "" && strategy != tc.expStrategy {
					t.Fatalf("expected strategy decision %q, got %q", tc.expStrategy, strategy)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expStrategy != "" && strategy != tc.expStrategy {
				t.Fatalf("expected strategy decision %q, got %q", tc.expStrategy, strategy)
			}
			if sw == nil || sw.OpenAPI == "" {
				t.Fatalf("expected non-nil spec with OpenAPI set")
			}
		})
	}
}

func TestLoadSwagger_URI_ForcedFallback(t *testing.T) {
	old := newLibDoc
	newLibDoc = func(_ []byte) (libopenapi.Document, error) { return nil, fmt.Errorf("forced parse failure") }
	t.Cleanup(func() { newLibDoc = old })

	// capture only the last decision
	oldHook := onLoaderDecision
	last := ""
	onLoaderDecision = func(d string) { last = d }
	t.Cleanup(func() { onLoaderDecision = oldHook })

	spec := []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(spec)
	}))
	defer ts.Close()

	t.Setenv("OAPI_CODEGEN_USE_LIBOPENAPI", "1")
	sw, err := LoadSwagger(ts.URL + "/spec.yaml")
	if err != nil || sw == nil {
		t.Fatalf("expected soft fallback success, err=%v", err)
	}
	if last != "fallback:uri" {
		t.Fatalf("expected last decision to be fallback:uri, got %q", last)
	}
}

func TestLoadSwagger_File_ForcedFallback(t *testing.T) {
	old := newLibDoc
	newLibDoc = func(_ []byte) (libopenapi.Document, error) { return nil, fmt.Errorf("forced parse failure") }
	t.Cleanup(func() { newLibDoc = old })

	// capture only the last decision
	oldHook := onLoaderDecision
	last := ""
	onLoaderDecision = func(d string) { last = d }
	t.Cleanup(func() { onLoaderDecision = oldHook })

	dir := t.TempDir()
	p := filepath.Join(dir, "spec.yaml")
	if err := os.WriteFile(p, []byte("openapi: 3.0.0\ninfo: {title: t, version: v}\npaths: {}\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Setenv("OAPI_CODEGEN_USE_LIBOPENAPI", "1")
	sw, err := LoadSwagger(p)
	if err != nil || sw == nil {
		t.Fatalf("expected soft fallback success for file, err=%v", err)
	}
	if last != "fallback:file" {
		t.Fatalf("expected last decision to be fallback:file, got %q", last)
	}
}

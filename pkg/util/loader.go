package util

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	libopenapi "github.com/pb33f/libopenapi"
	"github.com/speakeasy-api/openapi-overlay/pkg/loader"
	"gopkg.in/yaml.v3"
)

func LoadSwagger(filePath string) (swagger *openapi3.T, err error) {
	// Feature-flagged libopenapi loader pat with safe fallback. Defaults to legacy kin-openapi loader.
	if v := os.Getenv("OAPI_CODEGEN_USE_LIBOPENAPI"); v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") {
		return loadWithLibopenapi(filePath)
	}
	// TODO: Add strict mode with no fallback to loadWithKin.
	// else if v := os.Getenv("OAPI_CODEGEN_USE_LIBOPENAPI_STRICT"); v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") {
	// 	return loadWithLibopenapiStrict(filePath)
	// }
	return loadWithKin(filePath)
}

func loadWithKin(filePath string) (swagger *openapi3.T, err error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	u, err := url.Parse(filePath)
	if err == nil && u.Scheme != "" && u.Host != "" {
		return loader.LoadFromURI(u)
	} else {
		return loader.LoadFromFile(filePath)
	}
}

// loadWithLibopenapi attempts to parse the spec using libopenapi, then returns a
// kin-openapi *openapi3.T loaded from the same bytes (with correct base path) to
// preserve public API and generator behavior. If libopenapi parsing fails, it
// falls back to the legacy kin-openapi loader.
func loadWithLibopenapi(filePath string) (swagger *openapi3.T, err error) {
	// Mirror loadWithKin URL parse and branching, with fallback to legacy kin-openapi loader.
	u, err := url.Parse(filePath)
	if err == nil && u.Scheme != "" && u.Host != "" {
		if swagger, err := libopenapiLoadFromURI(u); err == nil {
			return swagger, nil
		}
		return loadWithKin(filePath)
	}
	if swagger, err := libopenapiLoadFromFile(filePath); err == nil {
		return swagger, nil
	}
	return loadWithKin(filePath)
}

func libopenapiLoadFromURI(u *url.URL) (*openapi3.T, error) {
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	if resp != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if _, derr := libopenapi.NewDocument(b); derr != nil {
		return nil, derr
	}
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	return loader.LoadFromDataWithPath(b, u)
}

func libopenapiLoadFromFile(filePath string) (*openapi3.T, error) {
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	if _, derr := libopenapi.NewDocument(b); derr != nil {
		return nil, derr
	}
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	return loader.LoadFromDataWithPath(b, &url.URL{Path: filepath.ToSlash(filePath)})
}

// Deprecated: In kin-openapi v0.126.0 (https://github.com/getkin/kin-openapi/tree/v0.126.0?tab=readme-ov-file#v01260) the Circular Reference Counter functionality was removed, instead resolving all references with backtracking, to avoid needing to provide a limit to reference counts.
//
// This is now identital in method as `LoadSwagger`.
func LoadSwaggerWithCircularReferenceCount(filePath string, _ int) (swagger *openapi3.T, err error) {
	return LoadSwagger(filePath)
}

type LoadSwaggerWithOverlayOpts struct {
	Path   string
	Strict bool
}

func LoadSwaggerWithOverlay(filePath string, opts LoadSwaggerWithOverlayOpts) (swagger *openapi3.T, err error) {
	spec, err := LoadSwagger(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI specification: %w", err)
	}

	if opts.Path == "" {
		return spec, nil
	}

	// parse out the yaml.Node, which is required by the overlay library
	data, err := yaml.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec from %#v as YAML: %w", filePath, err)
	}

	var node yaml.Node
	err = yaml.NewDecoder(bytes.NewReader(data)).Decode(&node)
	if err != nil {
		return nil, fmt.Errorf("failed to parse spec from %#v: %w", filePath, err)
	}

	overlay, err := loader.LoadOverlay(opts.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load Overlay from %#v: %v", opts.Path, err)
	}

	err = overlay.Validate()
	if err != nil {
		return nil, fmt.Errorf("the Overlay in %#v was not valid: %v", opts.Path, err)
	}

	if opts.Strict {
		err, vs := overlay.ApplyToStrict(&node)
		if err != nil {
			return nil, fmt.Errorf("failed to apply Overlay %#v to specification %#v: %v\nAdditionally, the following validation errors were found:\n- %s", opts.Path, filePath, err, strings.Join(vs, "\n- "))
		}
	} else {
		err = overlay.ApplyTo(&node)
		if err != nil {
			return nil, fmt.Errorf("failed to apply Overlay %#v to specification %#v: %v", opts.Path, filePath, err)
		}
	}

	b, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize Overlay'd specification %#v: %v", opts.Path, err)
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	swagger, err = loader.LoadFromDataWithPath(b, &url.URL{
		Path: filepath.ToSlash(filePath),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize Overlay'd specification %#v: %v", opts.Path, err)
	}

	return swagger, nil
}

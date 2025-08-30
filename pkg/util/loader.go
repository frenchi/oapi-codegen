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

// test seams to improve testability of the fallback case
var (
	httpGet   = http.Get
	newLibDoc = libopenapi.NewDocument
	// onLoaderDecision, if set, is invoked with: "kin:uri", "kin:file", "libopenapi:uri", "libopenapi:file", "fallback:uri", "fallback:file".
	onLoaderDecision = func(string) {}
)

// LoaderStrategy abstracts how an OpenAPI document is loaded.
// more info: https://refactoring.guru/design-patterns/strategy
type LoaderStrategy interface {
	Load(filePath string) (*openapi3.T, error)
}

type kinLoader struct{}

func (kinLoader) Load(filePath string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	u, err := url.Parse(filePath)
	if err == nil && u.Scheme != "" && u.Host != "" {

		onLoaderDecision("kin:uri")
		return loader.LoadFromURI(u)
	}

	onLoaderDecision("kin:file")
	return loader.LoadFromFile(filePath)
}

type libopenapiLoaderStrict struct{}

// libopenapiLoaderStrict enforces the libopenapi path with no fallback.
func (libopenapiLoaderStrict) Load(filePath string) (*openapi3.T, error) {
	u, err := url.Parse(filePath)
	if err == nil && u.Scheme != "" && u.Host != "" {

		onLoaderDecision("libopenapi:uri")
		return libopenapiLoadFromURI(u)
	}

	onLoaderDecision("libopenapi:file")
	return libopenapiLoadFromFile(filePath)
}

type libopenapiLoader struct{}

// libopenapiLoader tries libopenapi first and falls back to kin-openapi on error, for backwards compatibility.
func (libopenapiLoader) Load(filePath string) (*openapi3.T, error) {
	u, err := url.Parse(filePath)
	if err == nil && u.Scheme != "" && u.Host != "" {

		onLoaderDecision("libopenapi:uri")
		if s, err := libopenapiLoadFromURI(u); err == nil {
			return s, nil
		}
		// fallback to kin loader for URI

		onLoaderDecision("fallback:uri")
		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = true
		return loader.LoadFromURI(u)
	}

	onLoaderDecision("libopenapi:file")
	if s, err := libopenapiLoadFromFile(filePath); err == nil {
		return s, nil
	}
	// fallback to kin loader for file

	onLoaderDecision("fallback:file")
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	return loader.LoadFromFile(filePath)
}

// getLoaderStrategy selects a loader strategy based on environment.
// OAPI_CODEGEN_USE_LIBOPENAPI values:
//   - "strict": use libopenapi with no fallback
//   - "1", "true", "yes": use libopenapi with fallback to kin on error
//   - otherwise: use kin-openapi
func getLoaderStrategy() LoaderStrategy {
	v := os.Getenv("OAPI_CODEGEN_USE_LIBOPENAPI")
	if strings.EqualFold(v, "strict") {
		return libopenapiLoaderStrict{}
	}
	if v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes") {
		return libopenapiLoader{}
	}
	return kinLoader{}
}

func LoadSwagger(filePath string) (swagger *openapi3.T, err error) {
	return getLoaderStrategy().Load(filePath)
}

func libopenapiLoadFromURI(u *url.URL) (*openapi3.T, error) {
	resp, err := httpGet(u.String())
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
	if _, derr := newLibDoc(b); derr != nil {
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
	if _, derr := newLibDoc(b); derr != nil {
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

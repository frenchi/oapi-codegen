// Package ir defines a minimal intermediate representation consumed by the codegen
// back-end. The goal is to decouple the generator from any specific OpenAPI
// parser/model (kin-openapi or libopenapi) and make maintenance easier.
package ir

// SpecVersion is a coarse indicator of the OpenAPI version detected.
type SpecVersion int

const (
	SpecUnknown SpecVersion = iota
	Spec3_0
	Spec3_1
)

// Document is the root IR for a parsed OpenAPI spec.
type Document struct {
	Version    SpecVersion
	Info       Info
	Servers    []Server
	Paths      []PathItem // flattened list of paths; each item carries the path string
	Components Components
}

// Components mirrors OpenAPI components in a minimal, generator-friendly form.
type Components struct {
	Schemas         map[string]SchemaRef
	Parameters      map[string]Parameter
	Responses       map[string]Response
	RequestBodies   map[string]RequestBody
	Headers         map[string]Header
	SecuritySchemes map[string]any // keep opaque for now
}

type Info struct {
	Title       string
	Description string
	Version     string
}

type Server struct {
	URL         string
	Description string
}

type PathItem struct {
	Path       string
	Operations []Operation
}

type Operation struct {
	Method      string // GET/POST/...
	OperationID string
	Summary     string
	Description string
	Parameters  []Parameter
	RequestBody *RequestBody
	Responses   []Response
}

type Parameter struct {
	In          string // path, query, header, cookie
	Name        string
	Required    bool
	Description string
	Schema      SchemaRef
}

type RequestBody struct {
	Required bool
	Content  []MediaType
}

type Response struct {
	StatusCode  string // "200", "default"
	Description string
	Headers     []Header
	Content     []MediaType
}

type Header struct {
	Name        string
	Description string
	Schema      SchemaRef
}

type MediaType struct {
	ContentType string // e.g. application/json
	Schema      SchemaRef
}

type SchemaRef struct {
	Ref   string  // if non-empty, points to a component
	Value *Schema // set when inlined
}

type Schema struct {
	// Core shape
	Type     string   // object, array, string, number, integer, boolean, "" (free-form)
	Format   string   // uuid, date-time, etc.
	Nullable bool     // effective nullability
	Required []string // required properties

	// Object
	Properties           map[string]SchemaRef
	AdditionalProperties *SchemaRef // if nil: unspecified; if set and Value==nil and Ref=="": true/any

	// Array
	Items *SchemaRef

	// Compositions
	AllOf         []SchemaRef
	AnyOf         []SchemaRef
	OneOf         []SchemaRef
	Discriminator *Discriminator

	// Enum
	Enum []string

	// Extensions (x-...)
	Extensions map[string]any

	// Description
	Description string
}

type Discriminator struct {
	PropertyName string
	Mapping      map[string]string // discriminator value -> ref/name
}

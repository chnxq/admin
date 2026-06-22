package modulehost

import (
	"fmt"
	"strings"

	hostassets "admin/cmd/server/assets"

	"github.com/getkin/kin-openapi/openapi3"
	"gopkg.in/yaml.v3"
)

const defaultOpenAPIFormat = "yaml"

func RegisteredOpenAPIDocuments() []OpenAPIDocument {
	docs := make([]OpenAPIDocument, 0, len(modules)+1)
	if len(hostassets.OpenApiData) > 0 {
		docs = append(docs, OpenAPIDocument{
			Name:     "host",
			FileName: "openapi.yaml",
			Format:   defaultOpenAPIFormat,
			Data:     hostassets.OpenApiData,
		})
	}
	for _, module := range GetRegisteredHostModules() {
		if module == nil {
			continue
		}
		docs = append(docs, module.OpenAPIDocuments()...)
	}
	return docs
}

func MergeRegisteredOpenAPIDocument() (*openapi3.T, error) {
	return MergeOpenAPIDocuments(RegisteredOpenAPIDocuments())
}

func MarshalRegisteredOpenAPIDocument(format string) ([]byte, error) {
	doc, err := MergeRegisteredOpenAPIDocument()
	if err != nil || doc == nil {
		return nil, err
	}
	return MarshalOpenAPIDocument(doc, format)
}

func MergeOpenAPIDocuments(docs []OpenAPIDocument) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	var merged *openapi3.T
	for _, item := range docs {
		if len(item.Data) == 0 {
			continue
		}
		doc, err := loader.LoadFromData(item.Data)
		if err != nil {
			return nil, fmt.Errorf("load OpenAPI document %q: %w", item.Name, err)
		}
		if doc == nil {
			continue
		}
		if merged == nil {
			merged = doc
			continue
		}
		mergeOpenAPIDocument(merged, doc)
	}
	return merged, nil
}

func MarshalOpenAPIDocument(doc *openapi3.T, format string) ([]byte, error) {
	if doc == nil {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "yaml", "yml":
		value, err := doc.MarshalYAML()
		if err != nil {
			return nil, err
		}
		return yaml.Marshal(value)
	case "json":
		return doc.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported OpenAPI format %q", format)
	}
}

func mergeOpenAPIDocument(dst, src *openapi3.T) {
	if dst == nil || src == nil {
		return
	}
	if dst.Paths == nil {
		dst.Paths = openapi3.NewPaths()
	}
	if src.Paths != nil {
		for path, pathItem := range src.Paths.Map() {
			if pathItem == nil {
				continue
			}
			target := dst.Paths.Value(path)
			if target == nil {
				dst.Paths.Set(path, pathItem)
				continue
			}
			mergeOpenAPIPathItem(target, pathItem)
		}
	}
	dst.Tags = mergeOpenAPITags(dst.Tags, src.Tags)
	if src.Components != nil {
		if dst.Components == nil {
			components := openapi3.NewComponents()
			dst.Components = &components
		}
		mergeOpenAPIComponents(dst.Components, src.Components)
	}
}

func mergeOpenAPIPathItem(dst, src *openapi3.PathItem) {
	if dst == nil || src == nil {
		return
	}
	for method, operation := range src.Operations() {
		if operation == nil || dst.GetOperation(method) != nil {
			continue
		}
		dst.SetOperation(method, operation)
	}
	if dst.Summary == "" {
		dst.Summary = src.Summary
	}
	if dst.Description == "" {
		dst.Description = src.Description
	}
	if len(dst.Servers) == 0 && len(src.Servers) > 0 {
		dst.Servers = src.Servers
	}
	if len(dst.Parameters) == 0 && len(src.Parameters) > 0 {
		dst.Parameters = src.Parameters
	}
}

func mergeOpenAPITags(dst, src openapi3.Tags) openapi3.Tags {
	if len(src) == 0 {
		return dst
	}
	known := make(map[string]struct{}, len(dst))
	for _, tag := range dst {
		if tag == nil {
			continue
		}
		known[tag.Name] = struct{}{}
	}
	for _, tag := range src {
		if tag == nil {
			continue
		}
		if _, ok := known[tag.Name]; ok {
			continue
		}
		dst = append(dst, tag)
		known[tag.Name] = struct{}{}
	}
	return dst
}

func mergeOpenAPIComponents(dst, src *openapi3.Components) {
	if dst == nil || src == nil {
		return
	}
	mergeOpenAPISchemas(dst, src)
	mergeOpenAPIParameters(dst, src)
	mergeOpenAPIHeaders(dst, src)
	mergeOpenAPIRequestBodies(dst, src)
	mergeOpenAPIResponses(dst, src)
	mergeOpenAPISecuritySchemes(dst, src)
	mergeOpenAPIExamples(dst, src)
	mergeOpenAPILinks(dst, src)
	mergeOpenAPICallbacks(dst, src)
}

func mergeOpenAPISchemas(dst, src *openapi3.Components) {
	if len(src.Schemas) == 0 {
		return
	}
	if dst.Schemas == nil {
		dst.Schemas = openapi3.Schemas{}
	}
	for key, value := range src.Schemas {
		if _, ok := dst.Schemas[key]; ok {
			continue
		}
		dst.Schemas[key] = value
	}
}

func mergeOpenAPIParameters(dst, src *openapi3.Components) {
	if len(src.Parameters) == 0 {
		return
	}
	if dst.Parameters == nil {
		dst.Parameters = openapi3.ParametersMap{}
	}
	for key, value := range src.Parameters {
		if _, ok := dst.Parameters[key]; ok {
			continue
		}
		dst.Parameters[key] = value
	}
}

func mergeOpenAPIHeaders(dst, src *openapi3.Components) {
	if len(src.Headers) == 0 {
		return
	}
	if dst.Headers == nil {
		dst.Headers = openapi3.Headers{}
	}
	for key, value := range src.Headers {
		if _, ok := dst.Headers[key]; ok {
			continue
		}
		dst.Headers[key] = value
	}
}

func mergeOpenAPIRequestBodies(dst, src *openapi3.Components) {
	if len(src.RequestBodies) == 0 {
		return
	}
	if dst.RequestBodies == nil {
		dst.RequestBodies = openapi3.RequestBodies{}
	}
	for key, value := range src.RequestBodies {
		if _, ok := dst.RequestBodies[key]; ok {
			continue
		}
		dst.RequestBodies[key] = value
	}
}

func mergeOpenAPIResponses(dst, src *openapi3.Components) {
	if len(src.Responses) == 0 {
		return
	}
	if dst.Responses == nil {
		dst.Responses = openapi3.ResponseBodies{}
	}
	for key, value := range src.Responses {
		if _, ok := dst.Responses[key]; ok {
			continue
		}
		dst.Responses[key] = value
	}
}

func mergeOpenAPISecuritySchemes(dst, src *openapi3.Components) {
	if len(src.SecuritySchemes) == 0 {
		return
	}
	if dst.SecuritySchemes == nil {
		dst.SecuritySchemes = openapi3.SecuritySchemes{}
	}
	for key, value := range src.SecuritySchemes {
		if _, ok := dst.SecuritySchemes[key]; ok {
			continue
		}
		dst.SecuritySchemes[key] = value
	}
}

func mergeOpenAPIExamples(dst, src *openapi3.Components) {
	if len(src.Examples) == 0 {
		return
	}
	if dst.Examples == nil {
		dst.Examples = openapi3.Examples{}
	}
	for key, value := range src.Examples {
		if _, ok := dst.Examples[key]; ok {
			continue
		}
		dst.Examples[key] = value
	}
}

func mergeOpenAPILinks(dst, src *openapi3.Components) {
	if len(src.Links) == 0 {
		return
	}
	if dst.Links == nil {
		dst.Links = openapi3.Links{}
	}
	for key, value := range src.Links {
		if _, ok := dst.Links[key]; ok {
			continue
		}
		dst.Links[key] = value
	}
}

func mergeOpenAPICallbacks(dst, src *openapi3.Components) {
	if len(src.Callbacks) == 0 {
		return
	}
	if dst.Callbacks == nil {
		dst.Callbacks = openapi3.Callbacks{}
	}
	for key, value := range src.Callbacks {
		if _, ok := dst.Callbacks[key]; ok {
			continue
		}
		dst.Callbacks[key] = value
	}
}

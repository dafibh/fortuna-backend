package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dafibh/fortuna/fortuna-backend/docs"
	"github.com/labstack/echo/v4"
	"github.com/swaggo/swag"
)

// OpenAPI3Spec represents an OpenAPI 3.0 spec structure
type OpenAPI3Spec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       map[string]interface{} `json:"info"`
	Servers    []Server               `json:"servers"`
	Paths      map[string]interface{} `json:"paths"`
	Components map[string]interface{} `json:"components,omitempty"`
}

// Server represents an OpenAPI 3.0 server
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// transformRefs recursively transforms $ref from #/definitions/ to #/components/schemas/
// and converts Swagger 2.0 parameters to OpenAPI 3.0 format
func transformRefs(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})

		// Check if this is a parameter object (has "in" and "name" fields)
		if _, hasIn := v["in"]; hasIn {
			if _, hasName := v["name"]; hasName {
				return transformParameter(v)
			}
		}

		for key, value := range v {
			if key == "$ref" {
				if ref, ok := value.(string); ok {
					result[key] = strings.Replace(ref, "#/definitions/", "#/components/schemas/", 1)
				} else {
					result[key] = value
				}
			} else {
				result[key] = transformRefs(value)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = transformRefs(item)
		}
		return result
	default:
		return data
	}
}

// transformParameter converts a Swagger 2.0 parameter to OpenAPI 3.0 format
func transformParameter(param map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy standard fields
	for _, field := range []string{"name", "in", "description", "required"} {
		if val, ok := param[field]; ok {
			result[field] = val
		}
	}

	// Check if it's a body parameter (OpenAPI 3.0 handles these differently via requestBody)
	if param["in"] == "body" {
		// Keep as-is for now, body params need special handling
		return param
	}

	// Build schema object from type-related fields
	schema := make(map[string]interface{})
	for _, field := range []string{"type", "format", "enum", "default", "minimum", "maximum", "items"} {
		if val, ok := param[field]; ok {
			if field == "items" {
				// Transform $ref in items
				schema[field] = transformRefs(val)
			} else {
				schema[field] = val
			}
		}
	}

	if len(schema) > 0 {
		result["schema"] = schema
	}

	return result
}

// ServeOpenAPI3Spec serves the swagger spec converted to OpenAPI 3.0 with multiple servers
func ServeOpenAPI3Spec(c echo.Context) error {
	// Get the swagger 2.0 spec
	doc, err := swag.ReadDoc(docs.SwaggerInfo.InstanceName())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read swagger doc"})
	}

	// Parse swagger 2.0
	var swagger2 map[string]interface{}
	if err := json.Unmarshal([]byte(doc), &swagger2); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to parse swagger doc"})
	}

	// Extract info
	info, _ := swagger2["info"].(map[string]interface{})

	// Extract and transform paths (convert $ref from definitions to components/schemas)
	paths, _ := swagger2["paths"].(map[string]interface{})
	transformedPaths := transformRefs(paths).(map[string]interface{})

	// Convert securityDefinitions to components/securitySchemes
	components := make(map[string]interface{})
	if secDefs, ok := swagger2["securityDefinitions"].(map[string]interface{}); ok {
		components["securitySchemes"] = secDefs
	}
	if definitions, ok := swagger2["definitions"].(map[string]interface{}); ok {
		// Also transform $refs within definitions themselves
		components["schemas"] = transformRefs(definitions)
	}

	// Build OpenAPI 3.0 spec
	openapi3 := OpenAPI3Spec{
		OpenAPI: "3.0.3",
		Info:    info,
		Servers: []Server{
			{
				URL:         "http://localhost:18080/api/v1",
				Description: "Local Development",
			},
			{
				URL:         "https://fortunaapi.ghadafi.com/api/v1",
				Description: "Production",
			},
		},
		Paths:      transformedPaths,
		Components: components,
	}

	return c.JSON(http.StatusOK, openapi3)
}

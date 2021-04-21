package autodocs

// SwaggerConfig config of docs
type SwaggerConfig struct {
	Swagger             string                    `json:"swagger,omitempty"`
	OpenAPI             string                    `json:"openapi,omitempty"`
	Info                DocsInfo                  `json:"info"`
	Host                string                    `json:"host"`
	BasePath            string                    `json:"basePath"`
	Tags                []TagInfo                 `json:"tags"`
	Schemes             []string                  `json:"schemes"`
	Paths               map[string]PathInfo       `json:"paths"`
	SecurityDefinitions SecurityDefinitions       `json:"securityDefinitions"`
	Definitions         map[string]DefinitionInfo `json:"definitions"`
	ExternalDocs        ExternalDocsInfo          `json:"externalDocs,omitempty"`
}

// DocsInfo document information object
type DocsInfo struct {
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Version        string      `json:"version"`
	TermsOfService string      `json:"termsOfService"`
	Contact        ContactInfo `json:"contact"`
	License        LicenseInfo `json:"license"`
}

// ContactInfo contains email etc
type ContactInfo struct {
	Email     string `json:"email,omitempty"`
	Telephone string `json:"telephone,omitempty"`
}

// LicenseInfo license information
type LicenseInfo struct {
	Name string `json:"name,omitempty"`
	Url  string `json:"url,omitempty"`
}

// TagInfo tag information
type TagInfo struct {
	Name         string           `json:"name,omitempty"`
	Description  string           `json:"description,omitempty"`
	ExternalDocs ExternalDocsInfo `json:"externalDocs,omitempty"`
}

// ExternalDocsInfo external docs information
type ExternalDocsInfo struct {
	Description string `json:"description,omitempty"`
	Url         string `json:"url,omitempty"`
}

// PathInfo path information
type PathInfo struct {
	Post   *QueryInfo `json:"post,omitempty"`
	Get    *QueryInfo `json:"get,omitempty"`
	Put    *QueryInfo `json:"put,omitempty"`
	Delete *QueryInfo `json:"delete,omitempty"`
	Patch  *QueryInfo `json:"patch,omitempty"`
}

// QueryInfo query information
type QueryInfo struct {
	Tags        []string              `json:"tags,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Consumes    []string              `json:"consumes,omitempty"`
	Produces    []string              `json:"produces,omitempty"`
	Parameters  []ParameterInfo       `json:"parameters,omitempty"`
	RequestBody RequestBodyInfo       `json:"requestBody,omitempty"`
	Responses   map[string]SchemaInfo `json:"responses"`
	Security    []interface{}         `json:"security,omitempty"`
}

// ParameterInfo parameter information
type ParameterInfo struct {
	Type             string          `json:"type"`
	Name             string          `json:"name"`
	In               string          `json:"in"`
	Description      string          `json:"description,omitempty"`
	Required         bool            `json:"required,omitempty"`
	Format           string          `json:"format,omitempty"`
	Items            *PropertyInfo   `json:"items,omitempty"`
	CollectionFormat string          `json:"collectionFormat,omitempty"`
	Schema           *DefinitionInfo `json:"schema,omitempty"`
}

// RequestBodyInfo parameter in body information
type RequestBodyInfo struct {
	Description string                `json:"description,omitempty"`
	Required    bool                  `json:"required,omitempty"`
	Content     map[string]SchemaInfo `json:"content,omitempty"`
}

// SchemaInfo response information
type SchemaInfo struct {
	Description       string          `json:"description,omitempty"`
	Schema            *DefinitionInfo `json:"schema,omitempty"`
	SerializationType string          `json:"serializationType,omitempty"`
}

// DefinitionInfo model definition information
type DefinitionInfo struct {
	Type       string                  `json:"type,omitempty"`
	Properties map[string]PropertyInfo `json:"properties"`
	Required   []string                `json:"required,omitempty"`
	XML        XMLInfo                 `json:"xml,omitempty"`
}

// XMLInfo struct
type XMLInfo struct {
	Name    string `json:"name,omitempty"`
	Wrapped bool   `json:"wrapped,omitempty"`
}

// PropertyInfo property information
type PropertyInfo struct {
	Type        string                  `json:"type,omitempty"`
	Format      string                  `json:"format,omitempty"`
	Description string                  `json:"description,omitempty"`
	Enum        []string                `json:"enum,omitempty"`
	Reference   string                  `json:"$ref,omitempty"`
	Example     string                  `json:"example,omitempty"`
	Default     string                  `json:"default,omitempty"`
	XML         *XMLInfo                `json:"xml,omitempty"`
	Items       *PropertyInfo           `json:"items,omitempty"`
	Properties  map[string]PropertyInfo `json:"properties,omitempty"`
}

// QuerySecurityAuthInfo struct
type QuerySecurityAuthInfo struct {
	APIKey       []string `json:"jwt,omitempty"`
	PrestoreAuth []string `json:"petstoreAuth,omitempty"`
}

// SecurityDefinitions security definition
type SecurityDefinitions struct {
	APIKey       *APIKeyInfo       `json:"jwt,omitempty"`
	BasicAuth    *APIKeyInfo       `json:"basicAuth,omitempty"`
	PrestoreAuth *PrestoreAuthInfo `json:"petstoreAuth,omitempty"`
}

// APIKeyInfo struct
type APIKeyInfo struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	In   string `json:"in,omitempty"`
}

// PrestoreAuthInfo struct
type PrestoreAuthInfo struct {
	Type             string            `json:"type"`
	AuthorizationUrl string            `json:"authorizationUrl"`
	Flow             string            `json:"flow"`
	Scopes           map[string]string `json:"scopes,omitempty"`
}

// BearerAuthInfo struct
type BearerAuthInfo struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	In           string `json:"in"`
	Scheme       string `json:"scheme"`
	BearerFormat string `json:"bearerFormat,omitempty"`
}
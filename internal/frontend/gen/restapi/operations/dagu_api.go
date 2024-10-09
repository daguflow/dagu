// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/runtime/security"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"

	"github.com/dagu-org/dagu/internal/frontend/gen/restapi/operations/dags"
)

// NewDaguAPI creates a new Dagu instance
func NewDaguAPI(spec *loads.Document) *DaguAPI {
	return &DaguAPI{
		handlers:            make(map[string]map[string]http.Handler),
		formats:             strfmt.Default,
		defaultConsumes:     "application/json",
		defaultProduces:     "application/json",
		customConsumers:     make(map[string]runtime.Consumer),
		customProducers:     make(map[string]runtime.Producer),
		PreServerShutdown:   func() {},
		ServerShutdown:      func() {},
		spec:                spec,
		useSwaggerUI:        false,
		ServeError:          errors.ServeError,
		BasicAuthenticator:  security.BasicAuth,
		APIKeyAuthenticator: security.APIKeyAuth,
		BearerAuthenticator: security.BearerAuth,

		JSONConsumer: runtime.JSONConsumer(),

		JSONProducer: runtime.JSONProducer(),

		DagsCreateDagHandler: dags.CreateDagHandlerFunc(func(params dags.CreateDagParams) middleware.Responder {
			return middleware.NotImplemented("operation dags.CreateDag has not yet been implemented")
		}),
		DagsDeleteDagHandler: dags.DeleteDagHandlerFunc(func(params dags.DeleteDagParams) middleware.Responder {
			return middleware.NotImplemented("operation dags.DeleteDag has not yet been implemented")
		}),
		DagsGetDagDetailsHandler: dags.GetDagDetailsHandlerFunc(func(params dags.GetDagDetailsParams) middleware.Responder {
			return middleware.NotImplemented("operation dags.GetDagDetails has not yet been implemented")
		}),
		DagsListDagsHandler: dags.ListDagsHandlerFunc(func(params dags.ListDagsParams) middleware.Responder {
			return middleware.NotImplemented("operation dags.ListDags has not yet been implemented")
		}),
		DagsListHistoryHandler: dags.ListHistoryHandlerFunc(func(params dags.ListHistoryParams) middleware.Responder {
			return middleware.NotImplemented("operation dags.ListHistory has not yet been implemented")
		}),
		DagsListTagsHandler: dags.ListTagsHandlerFunc(func(params dags.ListTagsParams) middleware.Responder {
			return middleware.NotImplemented("operation dags.ListTags has not yet been implemented")
		}),
		DagsPostDagActionHandler: dags.PostDagActionHandlerFunc(func(params dags.PostDagActionParams) middleware.Responder {
			return middleware.NotImplemented("operation dags.PostDagAction has not yet been implemented")
		}),
		DagsSearchDagsHandler: dags.SearchDagsHandlerFunc(func(params dags.SearchDagsParams) middleware.Responder {
			return middleware.NotImplemented("operation dags.SearchDags has not yet been implemented")
		}),
	}
}

/*
DaguAPI Dagu is a simple DAG (Directed Acyclic Graph) runner.
It is a simple tool to run a series of tasks in a specific order
*/
type DaguAPI struct {
	spec            *loads.Document
	context         *middleware.Context
	handlers        map[string]map[string]http.Handler
	formats         strfmt.Registry
	customConsumers map[string]runtime.Consumer
	customProducers map[string]runtime.Producer
	defaultConsumes string
	defaultProduces string
	Middleware      func(middleware.Builder) http.Handler
	useSwaggerUI    bool

	// BasicAuthenticator generates a runtime.Authenticator from the supplied basic auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	BasicAuthenticator func(security.UserPassAuthentication) runtime.Authenticator

	// APIKeyAuthenticator generates a runtime.Authenticator from the supplied token auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	APIKeyAuthenticator func(string, string, security.TokenAuthentication) runtime.Authenticator

	// BearerAuthenticator generates a runtime.Authenticator from the supplied bearer token auth function.
	// It has a default implementation in the security package, however you can replace it for your particular usage.
	BearerAuthenticator func(string, security.ScopedTokenAuthentication) runtime.Authenticator

	// JSONConsumer registers a consumer for the following mime types:
	//   - application/json
	JSONConsumer runtime.Consumer

	// JSONProducer registers a producer for the following mime types:
	//   - application/json
	JSONProducer runtime.Producer

	// DagsCreateDagHandler sets the operation handler for the create dag operation
	DagsCreateDagHandler dags.CreateDagHandler
	// DagsDeleteDagHandler sets the operation handler for the delete dag operation
	DagsDeleteDagHandler dags.DeleteDagHandler
	// DagsGetDagDetailsHandler sets the operation handler for the get dag details operation
	DagsGetDagDetailsHandler dags.GetDagDetailsHandler
	// DagsListDagsHandler sets the operation handler for the list dags operation
	DagsListDagsHandler dags.ListDagsHandler
	// DagsListHistoryHandler sets the operation handler for the list history operation
	DagsListHistoryHandler dags.ListHistoryHandler
	// DagsListTagsHandler sets the operation handler for the list tags operation
	DagsListTagsHandler dags.ListTagsHandler
	// DagsPostDagActionHandler sets the operation handler for the post dag action operation
	DagsPostDagActionHandler dags.PostDagActionHandler
	// DagsSearchDagsHandler sets the operation handler for the search dags operation
	DagsSearchDagsHandler dags.SearchDagsHandler

	// ServeError is called when an error is received, there is a default handler
	// but you can set your own with this
	ServeError func(http.ResponseWriter, *http.Request, error)

	// PreServerShutdown is called before the HTTP(S) server is shutdown
	// This allows for custom functions to get executed before the HTTP(S) server stops accepting traffic
	PreServerShutdown func()

	// ServerShutdown is called when the HTTP(S) server is shut down and done
	// handling all active connections and does not accept connections any more
	ServerShutdown func()

	// Custom command line argument groups with their descriptions
	CommandLineOptionsGroups []swag.CommandLineOptionsGroup

	// User defined logger function.
	Logger func(string, ...interface{})
}

// UseRedoc for documentation at /docs
func (o *DaguAPI) UseRedoc() {
	o.useSwaggerUI = false
}

// UseSwaggerUI for documentation at /docs
func (o *DaguAPI) UseSwaggerUI() {
	o.useSwaggerUI = true
}

// SetDefaultProduces sets the default produces media type
func (o *DaguAPI) SetDefaultProduces(mediaType string) {
	o.defaultProduces = mediaType
}

// SetDefaultConsumes returns the default consumes media type
func (o *DaguAPI) SetDefaultConsumes(mediaType string) {
	o.defaultConsumes = mediaType
}

// SetSpec sets a spec that will be served for the clients.
func (o *DaguAPI) SetSpec(spec *loads.Document) {
	o.spec = spec
}

// DefaultProduces returns the default produces media type
func (o *DaguAPI) DefaultProduces() string {
	return o.defaultProduces
}

// DefaultConsumes returns the default consumes media type
func (o *DaguAPI) DefaultConsumes() string {
	return o.defaultConsumes
}

// Formats returns the registered string formats
func (o *DaguAPI) Formats() strfmt.Registry {
	return o.formats
}

// RegisterFormat registers a custom format validator
func (o *DaguAPI) RegisterFormat(name string, format strfmt.Format, validator strfmt.Validator) {
	o.formats.Add(name, format, validator)
}

// Validate validates the registrations in the DaguAPI
func (o *DaguAPI) Validate() error {
	var unregistered []string

	if o.JSONConsumer == nil {
		unregistered = append(unregistered, "JSONConsumer")
	}

	if o.JSONProducer == nil {
		unregistered = append(unregistered, "JSONProducer")
	}

	if o.DagsCreateDagHandler == nil {
		unregistered = append(unregistered, "dags.CreateDagHandler")
	}
	if o.DagsDeleteDagHandler == nil {
		unregistered = append(unregistered, "dags.DeleteDagHandler")
	}
	if o.DagsGetDagDetailsHandler == nil {
		unregistered = append(unregistered, "dags.GetDagDetailsHandler")
	}
	if o.DagsListDagsHandler == nil {
		unregistered = append(unregistered, "dags.ListDagsHandler")
	}
	if o.DagsListHistoryHandler == nil {
		unregistered = append(unregistered, "dags.ListHistoryHandler")
	}
	if o.DagsListTagsHandler == nil {
		unregistered = append(unregistered, "dags.ListTagsHandler")
	}
	if o.DagsPostDagActionHandler == nil {
		unregistered = append(unregistered, "dags.PostDagActionHandler")
	}
	if o.DagsSearchDagsHandler == nil {
		unregistered = append(unregistered, "dags.SearchDagsHandler")
	}

	if len(unregistered) > 0 {
		return fmt.Errorf("missing registration: %s", strings.Join(unregistered, ", "))
	}

	return nil
}

// ServeErrorFor gets a error handler for a given operation id
func (o *DaguAPI) ServeErrorFor(operationID string) func(http.ResponseWriter, *http.Request, error) {
	return o.ServeError
}

// AuthenticatorsFor gets the authenticators for the specified security schemes
func (o *DaguAPI) AuthenticatorsFor(schemes map[string]spec.SecurityScheme) map[string]runtime.Authenticator {
	return nil
}

// Authorizer returns the registered authorizer
func (o *DaguAPI) Authorizer() runtime.Authorizer {
	return nil
}

// ConsumersFor gets the consumers for the specified media types.
// MIME type parameters are ignored here.
func (o *DaguAPI) ConsumersFor(mediaTypes []string) map[string]runtime.Consumer {
	result := make(map[string]runtime.Consumer, len(mediaTypes))
	for _, mt := range mediaTypes {
		switch mt {
		case "application/json":
			result["application/json"] = o.JSONConsumer
		}

		if c, ok := o.customConsumers[mt]; ok {
			result[mt] = c
		}
	}
	return result
}

// ProducersFor gets the producers for the specified media types.
// MIME type parameters are ignored here.
func (o *DaguAPI) ProducersFor(mediaTypes []string) map[string]runtime.Producer {
	result := make(map[string]runtime.Producer, len(mediaTypes))
	for _, mt := range mediaTypes {
		switch mt {
		case "application/json":
			result["application/json"] = o.JSONProducer
		}

		if p, ok := o.customProducers[mt]; ok {
			result[mt] = p
		}
	}
	return result
}

// HandlerFor gets a http.Handler for the provided operation method and path
func (o *DaguAPI) HandlerFor(method, path string) (http.Handler, bool) {
	if o.handlers == nil {
		return nil, false
	}
	um := strings.ToUpper(method)
	if _, ok := o.handlers[um]; !ok {
		return nil, false
	}
	if path == "/" {
		path = ""
	}
	h, ok := o.handlers[um][path]
	return h, ok
}

// Context returns the middleware context for the dagu API
func (o *DaguAPI) Context() *middleware.Context {
	if o.context == nil {
		o.context = middleware.NewRoutableContext(o.spec, o, nil)
	}

	return o.context
}

func (o *DaguAPI) initHandlerCache() {
	o.Context() // don't care about the result, just that the initialization happened
	if o.handlers == nil {
		o.handlers = make(map[string]map[string]http.Handler)
	}

	if o.handlers["POST"] == nil {
		o.handlers["POST"] = make(map[string]http.Handler)
	}
	o.handlers["POST"]["/dags"] = dags.NewCreateDag(o.context, o.DagsCreateDagHandler)
	if o.handlers["DELETE"] == nil {
		o.handlers["DELETE"] = make(map[string]http.Handler)
	}
	o.handlers["DELETE"]["/dags/{dagId}"] = dags.NewDeleteDag(o.context, o.DagsDeleteDagHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/dags/{dagId}"] = dags.NewGetDagDetails(o.context, o.DagsGetDagDetailsHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/dags"] = dags.NewListDags(o.context, o.DagsListDagsHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/history"] = dags.NewListHistory(o.context, o.DagsListHistoryHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/tags"] = dags.NewListTags(o.context, o.DagsListTagsHandler)
	if o.handlers["POST"] == nil {
		o.handlers["POST"] = make(map[string]http.Handler)
	}
	o.handlers["POST"]["/dags/{dagId}"] = dags.NewPostDagAction(o.context, o.DagsPostDagActionHandler)
	if o.handlers["GET"] == nil {
		o.handlers["GET"] = make(map[string]http.Handler)
	}
	o.handlers["GET"]["/search"] = dags.NewSearchDags(o.context, o.DagsSearchDagsHandler)
}

// Serve creates a http handler to serve the API over HTTP
// can be used directly in http.ListenAndServe(":8000", api.Serve(nil))
func (o *DaguAPI) Serve(builder middleware.Builder) http.Handler {
	o.Init()

	if o.Middleware != nil {
		return o.Middleware(builder)
	}
	if o.useSwaggerUI {
		return o.context.APIHandlerSwaggerUI(builder)
	}
	return o.context.APIHandler(builder)
}

// Init allows you to just initialize the handler cache, you can then recompose the middleware as you see fit
func (o *DaguAPI) Init() {
	if len(o.handlers) == 0 {
		o.initHandlerCache()
	}
}

// RegisterConsumer allows you to add (or override) a consumer for a media type.
func (o *DaguAPI) RegisterConsumer(mediaType string, consumer runtime.Consumer) {
	o.customConsumers[mediaType] = consumer
}

// RegisterProducer allows you to add (or override) a producer for a media type.
func (o *DaguAPI) RegisterProducer(mediaType string, producer runtime.Producer) {
	o.customProducers[mediaType] = producer
}

// AddMiddlewareFor adds a http middleware to existing handler
func (o *DaguAPI) AddMiddlewareFor(method, path string, builder middleware.Builder) {
	um := strings.ToUpper(method)
	if path == "/" {
		path = ""
	}
	o.Init()
	if h, ok := o.handlers[um][path]; ok {
		o.handlers[um][path] = builder(h)
	}
}

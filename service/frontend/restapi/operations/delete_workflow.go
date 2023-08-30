// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// DeleteWorkflowHandlerFunc turns a function with the right signature into a delete workflow handler
type DeleteWorkflowHandlerFunc func(DeleteWorkflowParams) middleware.Responder

// Handle executing the request and returning a response
func (fn DeleteWorkflowHandlerFunc) Handle(params DeleteWorkflowParams) middleware.Responder {
	return fn(params)
}

// DeleteWorkflowHandler interface for that can handle valid delete workflow params
type DeleteWorkflowHandler interface {
	Handle(DeleteWorkflowParams) middleware.Responder
}

// NewDeleteWorkflow creates a new http.Handler for the delete workflow operation
func NewDeleteWorkflow(ctx *middleware.Context, handler DeleteWorkflowHandler) *DeleteWorkflow {
	return &DeleteWorkflow{Context: ctx, Handler: handler}
}

/*
	DeleteWorkflow swagger:route DELETE /workflows/{workflowId} deleteWorkflow

DeleteWorkflow delete workflow API
*/
type DeleteWorkflow struct {
	Context *middleware.Context
	Handler DeleteWorkflowHandler
}

func (o *DeleteWorkflow) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewDeleteWorkflowParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}

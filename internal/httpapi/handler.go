package httpapi

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

const contentType = "application/json; charset=utf-8"

type responseBody struct {
	Service   string `json:"service"`
	Message   string `json:"message"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	RequestID string `json:"requestId,omitempty"`
}

// NewHandler creates an API Gateway HTTP API (payload v2) Lambda handler.
func NewHandler(service string) func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return func(_ context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
		body, err := json.Marshal(responseBody{
			Service:   service,
			Message:   service + " service is running",
			Method:    request.RequestContext.HTTP.Method,
			Path:      request.RawPath,
			RequestID: request.RequestContext.RequestID,
		})
		if err != nil {
			return events.APIGatewayV2HTTPResponse{}, err
		}

		return events.APIGatewayV2HTTPResponse{
			StatusCode: 200,
			Headers: map[string]string{
				"content-type": contentType,
			},
			Body: string(body),
		}, nil
	}
}

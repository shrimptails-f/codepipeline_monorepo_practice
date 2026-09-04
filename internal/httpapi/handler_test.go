package httpapi_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"

	"backend/internal/httpapi"
)

func TestHandlerReturnsRequestDetails(t *testing.T) {
	t.Parallel()

	handler := httpapi.NewHandler("user")
	request := events.APIGatewayV2HTTPRequest{
		RawPath: "/users/42",
		RequestContext: events.APIGatewayV2HTTPRequestContext{
			RequestID: "request-123",
			HTTP: events.APIGatewayV2HTTPRequestContextHTTPDescription{
				Method: "GET",
			},
		},
	}

	response, err := handler(context.Background(), request)
	if err != nil {
		t.Fatalf("handler returned an error: %v", err)
	}
	if response.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", response.StatusCode)
	}
	if got := response.Headers["content-type"]; got != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q, want application/json; charset=utf-8", got)
	}

	var body struct {
		Service   string `json:"service"`
		Message   string `json:"message"`
		Method    string `json:"method"`
		Path      string `json:"path"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal([]byte(response.Body), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v", err)
	}

	if body.Service != "user" {
		t.Errorf("service = %q, want user", body.Service)
	}
	if body.Message != "user service is running" {
		t.Errorf("message = %q, want user service is running", body.Message)
	}
	if body.Method != "GET" {
		t.Errorf("method = %q, want GET", body.Method)
	}
	if body.Path != "/users/42" {
		t.Errorf("path = %q, want /users/42", body.Path)
	}
	if body.RequestID != "request-123" {
		t.Errorf("requestId = %q, want request-123", body.RequestID)
	}
}

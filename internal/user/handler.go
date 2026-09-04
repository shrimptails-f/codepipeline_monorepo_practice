package user

import (
	"context"

	"github.com/aws/aws-lambda-go/events"

	"backend/internal/httpapi"
)

func NewHandler() func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return httpapi.NewHandler("user")
}

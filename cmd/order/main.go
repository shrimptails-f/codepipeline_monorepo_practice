package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"backend/internal/order"
)

func main() {
	lambda.Start(order.NewHandler())
}

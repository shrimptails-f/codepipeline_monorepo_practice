package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"backend/internal/user"
)

func main() {
	lambda.Start(user.NewHandler())
}

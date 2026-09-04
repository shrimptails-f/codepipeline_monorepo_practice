package main

import (
	"github.com/aws/aws-lambda-go/lambda"

	"backend/internal/post"
)

func main() {
	lambda.Start(post.NewHandler())
}

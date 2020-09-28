package main

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Returning the error will result in "message": "Internal server error" with 502 Bad Gateway, so do not return the error if you want a custom display.
func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Load information from environment variables and make it available on a global basis
	env, err := loadEnvConfig()
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}

	// Output debug log
	debug.Printf("request: %+v\n", request)

	// Parse redirect ID from path
	id := request.PathParameters["proxy"]
	log.Printf("id: %+v\n", id)

	// Initialize the aws client
	aws := newAwsClient()

	result := `{"result": "ok"}`

	// Acquiring URL information of the redirected destination from Dynamo DB
	urlItem, err := aws.getURLItem(env.URLTableName, id)
	if err != nil {
		result = `{"Error": "No redirect found for id [` + id + `]"}`
		return events.APIGatewayProxyResponse{Body: result, StatusCode: 400}, nil
	}
	log.Printf("urlItem: %+v\n", urlItem)

	// If there is no redirection destination, there is an error
	location := strings.TrimSpace(urlItem.URL)
	if location == "" {
		result = `{"Error": "No redirect found for location [` + location + `]"}`
		return events.APIGatewayProxyResponse{Body: result, StatusCode: 400}, nil
	}

	// The URL itself has an expiration date, so display it
	urlExpiry := strings.TrimSpace(strconv.FormatInt(urlItem.TTL, 10))
	dateStr, err := getHTTPDateString(urlExpiry)
	if err == nil {
		urlExpiry = dateStr
	}

	// Setting the response header
	headers := map[string]string{
		"Location":      location,
		"x-url-expires": urlExpiry,
	}

	// In case of success, it's a temporary redirect and returns 302
	return events.APIGatewayProxyResponse{Headers: headers, Body: result, StatusCode: 302}, nil
}

func main() {
	lambda.Start(handleRequest)
}

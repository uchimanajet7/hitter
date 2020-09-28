package main

import (
	"context"
	"log"

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

	// Initialize the slack client
	sc := newSlackClient(env.SlackOAuthAccessToken)

	// Parsing JSON of events sent from slack
	se, result, err := sc.parseEvent(request.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}
	if result != "" {
		return events.APIGatewayProxyResponse{Body: result, StatusCode: 200}, nil
	}

	// Initialize the aws client
	aws := newAwsClient()

	// Do not process the same event multiple times
	if aws.checkAndPutMutexItem(env.MutexTableName, se.EventID) {
		result = `{"message": "[REJECTED] Already running under the same slack event ID"}`
		return events.APIGatewayProxyResponse{Body: result, StatusCode: 200}, nil
	}

	// Parse the command to be executed
	cmd := parseCommand(se)

	// Actually execute the command
	err = cmd.runCommand(sc, aws)
	if err != nil {
		log.Println("[ERROR] Processing failed: ", err)
		return events.APIGatewayProxyResponse{Body: err.Error(), StatusCode: 500}, nil
	}
	result = `{"result": "ok"}`

	return events.APIGatewayProxyResponse{Body: result, StatusCode: 200}, nil
}

func main() {
	lambda.Start(handleRequest)
}

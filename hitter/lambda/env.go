package main

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// Keep information obtained from environment variables in global variables.
var envconf *envConfig

type envConfig struct {
	SlackOAuthAccessToken  string `envconfig:"SLACK_OAUTH_ACCESS_TOKEN" required:"true"`
	SlackVerificationToken string `envconfig:"SLACK_VERIFICATION_TOKEN" required:"true"`
	MutexTableName         string `envconfig:"MUTEX_TABLE_NAME" required:"true"`
	URLTableName           string `envconfig:"URL_TABLE_NAME" required:"true"`
	S3BucketName           string `envconfig:"S3_BUCKET_NAME" required:"true"`
	APIBaseURL             string `envconfig:"API_BASE_URL" required:"true"`
	SlackChannelID         string `envconfig:"SLACK_CHANNEL_ID"`
	DebugLog               bool   `envconfig:"DEBUG_LOG"`
}

func loadEnvConfig() (*envConfig, error) {
	// Load environment variables
	env := &envConfig{}

	err := envconfig.Process("", env)
	if err != nil {
		log.Println("[ERROR] Failed to load environment variables: ", err)
		return env, err
	}

	// Keep information obtained from environment variables in global variables.
	envconf = env

	// Debug log output settings
	debug = debugT(env.DebugLog)

	// Output debug log
	debug.Printf("env: %+v\n", env)

	return env, err
}

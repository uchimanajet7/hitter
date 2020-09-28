package main

import (
	"log"

	"github.com/kelseyhightower/envconfig"
)

// Keep information obtained from environment variables in global variables.
var envconf *envConfig

// Be careful!
// The required specification is different from the one used in the slack bot's Lambda mission
type envConfig struct {
	SlackOAuthAccessToken  string `envconfig:"SLACK_OAUTH_ACCESS_TOKEN"`
	SlackVerificationToken string `envconfig:"SLACK_VERIFICATION_TOKEN"`
	MutexTableName         string `envconfig:"MUTEX_TABLE_NAME"`
	URLTableName           string `envconfig:"URL_TABLE_NAME" required:"true"`
	S3BucketName           string `envconfig:"S3_BUCKET_NAME"`
	APIBaseURL             string `envconfig:"API_BASE_URL"`
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

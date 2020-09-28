# hitter

hitter is a bot of slack, which can randomly select members and perform translations.

![screenshot](https://user-images.githubusercontent.com/6448792/94336534-e40ce000-001e-11eb-9de5-149f8437b99c.png)

- Use the following icon for the bot
	- https://stampo.fun

## Description
- Works as a slack bot
- It works by mentoring the bot and entering the default commands.
- Add a bot to the slack channel and use it
- May be subject to slack and AWS Lambda limitations

## Features
The following five commands are currently available

1. **hit**
	- Randomly selected from the members of the channel
1. **translate**
	- Translate the text as you type it.
1. **link**
	- Generate a pre-signed URL to get access the attachments
1. **short**
	- Generate a shortened URL with expiration date
1. **help**
	- Displays help for the command

## Requirement
- Go 1.15+
- Packages in use
	- aws / aws-lambda-go: ALibraries, samples and tools to help Go developers develop AWS Lambda functions.
		- https://github.com/aws/aws-lambda-go
	- slack-go / slack: Slack API in Go - community-maintained fork created by the original author, @nlopes
		- https://github.com/slack-go/slack
	- guregu / dynamo: expressive DynamoDB library for Go
		- https://github.com/guregu/dynamo
	- google / uuid: Go package for UUIDs based on RFC 4122 and DCE 1.1: Authentication and Security Services.
		- https://github.com/google/uuid
	- araddon / dateparse: GoLang Parse many date strings without knowing format in advance.
		- https://github.com/araddon/dateparse
	- kelseyhightower / envconfig: Golang library for managing configuration data from environment variables
		- https://github.com/kelseyhightower/envconfig

- Python 3.8+
- Node.js 14+
- aws / aws-cdk: The AWS Cloud Development Kit is a framework for defining cloud infrastructure in code
	- https://github.com/aws/aws-cdk
	
## Usage
For details on how to use each command, see below

- **hit**
	- Synopsis
		- `@hitter hit <number of selections> [<options> ...] `
			- Please specify the number of people to select.
			- It is an error to specify how many people to select for a channel
			- It may take longer to run if there are too many participants in the channel
	- Options
		- `--ex <@channel participant>`
			- You can specify which members you want to exclude from the selection
			- Multiple options can be configured
			- Be sure to specify the format in which you want to mention
	- Examples
		- `@hitter hit 2`
			- I will select two participants from the channel
		- `@hitter hit 3 --ex @userA --ex @userB`
			- We will select three participants from the channel
			- There are options, so @userA and @userB are not available

- **translate**
	- Synopsis
		- `@hitter translate <input text>`
			- Input text is recognized even if it is on a new line
			- The language you entered is automatically determined
			- If there is a mixture of languages, it may not be determined correctly.
			- The upper limit of the input string depends on the maximum input value of slack
			- Whenever Japanese is entered, it will be translated into English.
			- If you enter a language other than Japanese, it will always be translated into Japanese.
	- Options
		- None
	- Examples
		- `@hitter translate AWS is the world's most comprehensive and broadly adopted cloud platform`
			- It's an English input, so it will be translated into Japanese
		- `@hitter translate AWS は、世界で最も包括的で広く採用されているクラウドプラットフォームです`
			- It's a Japanese input, so it will be translated into English

- **link**
	- Synopsis
		- `@hitter link <expired minutes> <attachments>`
			- Please specify the number of minutes of validity
			- Pre-Signed URLs will become invalid after the expiry minutes
			- Expiration minutes is dependent on AWS limitations
				- https://aws.amazon.com/premiumsupport/knowledge-center/presigned-url-s3-bucket-expiration/
				- AWS Security Token Service (STS): corresponds to a maximum of 36 hours of availability in this case
			- The default expiration minutes is set to 15 minutes
			- Multiple attachments are allowed.
			- Pre-Signed URLs are generated per attachment
			- The attachment size limit depends on the memory size of AWS Lambda
	- Options
		- None
	- Examples
		- `@hitter link attachmentA`
			- Generate a pre-signed URL for attachment A
		- `@hitter link attachmentA attachmentB`
			- Generates pre-signed URLs for attachment A and attachment B, respectively

- **short**
	- Synopsis
		- `@hitter short <URL> <option>`
			- Please specify the URL to be shortened
	- Options
		- `--ttl <expiration date>`
			- Shortened URLs will not be available after the expiration date
			- The default expiration date is set to 1 day
			- The number of days of validity depends on the TTL feature of Amazon DynamoDB
				- https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/howitworks-ttl.html
			- Depending on the timing of the deletion, the specified expiration date may be exceeded.
	- Examples
		- `@hitter short https://www.example.com/`
			- Generate a shortened URL for https://www.example.com/
		- `@hitter short https://www.example.net/ --ttl 4`
			- Generates a shortened URL of https://www.example.net/ 
			- expires in 4 days

- **help**
	- Synopsis
		- `@hitter help`
	- Options
		- None
	- Examples
		- `@hitter help`
			- Show brief help

## Limitations
- About AWS Lambda

	- Dependent on memory capacity
		- Maximum file size of the attachment
		- Current setting is 2GB.
		- Maximum memory capacity of 3GB.
		- Files over the memory capacity will fail to be attached.
		- Architectural changes are required to break through this limitation
		- Consider using the Amazon Elastic File System (Amazon EFS)
		- https://docs.aws.amazon.com/lambda/latest/dg/services-efs.html

	- Dependent on execution time
		- Current setting is 15 minutes.
		- Maximum run time is 15 minutes.
		- If there are many members participating in the slack channel, the run time may be exceeded
		- If the attachment is large, the execution time may be exceeded.

- About Amazon DynamoDB

	- Depends on the capacity unit
		- Currently in On-Demand Mode
		- Note that it is automatically expanded, but the price is proportional

	- Depends on TTL specification
		- Item deletion is done through the TTL features.
		- This could be longer than the specified period depending on the TTL specification
		- https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/howitworks-ttl.html

- About AWS Resources

	- About the start of use
		- Basically, we use AWS CDK to build it
		- AWS CloudFormation is executed from CDK
		- However, the zone to be used as a domain must be created in advance
		- The zone name and zone ID will be required during the build process.

	- About deletion
		- Basically, the deletion is done by AWS CDK
		- AWS CloudFormation is executed from CDK
		- Zones for pre-created domains will not be deleted
		- Logs output to CloudWatch Logs will not be deleted
		- S3 cannot be deleted unless the bucket is empty, so it will not be deleted
		- All other resources will be deleted automatically.
		- If you need it, please make a backup in advance
		- To remove it completely, you will need to remove the above resources manually

## Resources
- Amazon S3
	- https://aws.amazon.com/s3/
- Amazon API Gateway
	- https://aws.amazon.com/api-gateway/
- AWS Lambda
	- https://aws.amazon.com/lambda/
- Amazon Route 53
	- https://aws.amazon.com/route53/
- AWS Certificate Manager
	- https://aws.amazon.com/certificate-manager/
- Amazon DynamoDB
	- https://aws.amazon.com/dynamodb/
- Amazon CloudWatch Logs
	- https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/WhatIsCloudWatchLogs.html
- AWS CloudFormation
	- https://aws.amazon.com/cloudformation/
- AWS Identity and Access Management (IAM)
	- https://aws.amazon.com/iam/


## Deployment

Get the sources yourself and do the build and deployment.

#### 1. Getting the Source
```	console
$ git clone github.com/uchimanajet7/hitter
```

#### 2. Building Lambda Functions
```	console
cd ./hitter/hitter/lambda
go get -v -t -d ./...
GOOS=linux go build -o main

cd ./hitter/hitter/lambda_api
go get -v -t -d ./...
GOOS=linux go build -o main
```

#### 3. Deployment with AWS CDK
```	console
$ cd ./hitter/hitter
$ source .env/bin/activate
$ cdk synth
$ cdk deploy
```

### You need to set the environment variables in order to deploy

You need to install the tools and set the environment variables in advance.

#### Setting Environment Variables in AWS CDK
```	console
export AWS_ACCESS_KEY_ID=<YOUR ACCESS KEY>
export AWS_SECRET_ACCESS_KEY=<YOUR SECRET KEY>
export AWS_DEFAULT_REGION=ap-northeast-1
export AWS_DEFAULT_OUTPUT=json
```

#### Setting Environment Variables in Amazon Route 53
```	console
export HITTER_ZONE_NAME=example.com
export HITTER_ZONE_ID=<YOUR ZONE ID>
```

#### Setting Environment Variables in Slack
```	console
export HITTER_SLACK_OAUTH_ACCESS_TOKEN=<YOUR SLACK OAUTH TOKEN>
export HITTER_SLACK_VERIFICATION_TOKEN=<YOUR SLACK VERIFICATION TOKEN>
```

If you don't have the time to set up the tools, you can use the Remote - Containers extension and Docker in Visual Studio Code to help you.

- Visual Studio Code
	- https://code.visualstudio.com
- Visual Studio Code Remote - Containers
	- https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers
- Developing inside a Container
	- https://code.visualstudio.com/docs/remote/containers
- Docker
	- https://www.docker.com


## Author
[uchimanajet7](https://github.com/uchimanajet7)

## Licence
[Apache License 2.0](https://github.com/uchimanajet7/hitter/blob/master/LICENSE)

## As reference information
- Create a Slack Bot for random selection using AWS CDK #aws #slack #cdk
	- https://medium.com/@uchimanajet7/create-a-slack-bot-for-random-selection-using-aws-cdk-aws-slack-cdk-e90b8a1c3536
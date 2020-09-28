import os

from aws_cdk import (
    core,
    aws_lambda,
    aws_apigateway,
    aws_dynamodb,
    aws_iam,
    aws_s3,
    aws_route53,
    aws_route53_targets,
    aws_certificatemanager,
)

# Retrieving Information from Environment Variables
SLACK_OAUTH_ACCESS_TOKEN = os.environ.get('HITTER_SLACK_OAUTH_ACCESS_TOKEN')
SLACK_VERIFICATION_TOKEN = os.environ.get('HITTER_SLACK_VERIFICATION_TOKEN')
ZONE_NAME = os.environ.get('HITTER_ZONE_NAME')
ZONE_ID = os.environ.get('HITTER_ZONE_ID')


class HitterStack(core.Stack):

    def __init__(self, scope: core.Construct, id: str, **kwargs) -> None:
        super().__init__(scope, id, **kwargs)

        # Setting up a salck bot Lambda function
        bot_handler = aws_lambda.Function(
            self, "HitterBot",
            runtime=aws_lambda.Runtime.GO_1_X,
            handler="main",
            timeout=core.Duration.seconds(900),
            memory_size=2048,
            code=aws_lambda.AssetCode(path="./lambda")
        )

        # Creating Mutex Table in DynamoDB
        mutex_table = aws_dynamodb.Table(self, "HitterMutexTable",
                                         partition_key=aws_dynamodb.Attribute(
                                             name="ID",
                                             type=aws_dynamodb.AttributeType.STRING),
                                         billing_mode=aws_dynamodb.BillingMode.PAY_PER_REQUEST,
                                         time_to_live_attribute="TTL",
                                         removal_policy=core.RemovalPolicy.DESTROY,
                                         )

        # Creating URL Table in DynamoDB
        url_table = aws_dynamodb.Table(self, "HitterURLTable",
                                       partition_key=aws_dynamodb.Attribute(
                                           name="ID",
                                           type=aws_dynamodb.AttributeType.STRING),
                                       billing_mode=aws_dynamodb.BillingMode.PAY_PER_REQUEST,
                                       time_to_live_attribute="TTL",
                                       removal_policy=core.RemovalPolicy.DESTROY,
                                       )

        # Creating a bucket to be used in a Pre-Signed URL
        bucket = aws_s3.Bucket(self, "HitterS3",
                               removal_policy=core.RemovalPolicy.RETAIN,
                               block_public_access=aws_s3.BlockPublicAccess.BLOCK_ALL,
                               )

        # Configuring the S3 Lifecycle
        bucket.add_lifecycle_rule(expiration=core.Duration.days(2))

        # Setting permission to the salck bot Lambda function
        mutex_table.grant_read_write_data(bot_handler)
        url_table.grant_read_write_data(bot_handler)
        bucket.grant_put(bot_handler)
        bucket.grant_read(bot_handler)
        bot_handler.add_to_role_policy(aws_iam.PolicyStatement(
            resources=["*"], actions=["comprehend:BatchDetectDominantLanguage", "translate:TranslateText"]))

        # Setting environment variables to the salck bot Lambda function
        bot_handler.add_environment(
            'SLACK_OAUTH_ACCESS_TOKEN', SLACK_OAUTH_ACCESS_TOKEN)
        bot_handler.add_environment(
            'SLACK_VERIFICATION_TOKEN', SLACK_VERIFICATION_TOKEN)
        bot_handler.add_environment('MUTEX_TABLE_NAME', mutex_table.table_name)
        bot_handler.add_environment('URL_TABLE_NAME', url_table.table_name)
        bot_handler.add_environment('S3_BUCKET_NAME', bucket.bucket_name)
        bot_handler.add_environment('DEBUG_LOG', "false")

        # Creating an API Gateway for a slack bot
        bot_api = aws_apigateway.LambdaRestApi(
            self, "HitterBotAPI", handler=bot_handler)

        # Only once for the hosted zone.
        hosted_zone = aws_route53.HostedZone.from_hosted_zone_attributes(
            self, 'HitterHostedZone', hosted_zone_id=ZONE_ID, zone_name=ZONE_NAME)

        # Set the domain for a bot
        bot_subdomain = "hitter"
        bot_domain_name = bot_subdomain + '.' + ZONE_NAME

        # Using AWS ACM to create a certificate for a bot
        bot_cert = aws_certificatemanager.DnsValidatedCertificate(
            self, 'HitterBotCertificate', domain_name=bot_domain_name, hosted_zone=hosted_zone)

        # Add the domain name to the api and the A record to our hosted zone for a bot
        bot_domain = bot_api.add_domain_name(
            'HitterBotDomain', certificate=bot_cert, domain_name=bot_domain_name)

        # Set the A record for a bot
        aws_route53.ARecord(
            self, 'HitterBotARecord',
            record_name=bot_subdomain,
            zone=hosted_zone,
            target=aws_route53.RecordTarget.from_alias(aws_route53_targets.ApiGatewayDomain(bot_domain)))

        # Setting up Short URL Lambda function
        short_url_handler = aws_lambda.Function(
            self, "HitterShortURL",
            runtime=aws_lambda.Runtime.GO_1_X,
            handler="main",
            timeout=core.Duration.seconds(900),
            memory_size=128,
            code=aws_lambda.AssetCode(path="./lambda_api")
        )

        # Setting environment variables to the Short URL Lambda function
        url_table.grant_read_data(short_url_handler)

        # Setting environment variables to the salck bot Lambda function
        short_url_handler.add_environment(
            'URL_TABLE_NAME', url_table.table_name)
        short_url_handler.add_environment('DEBUG_LOG', "false")

        # Creating an API Gateway for a Short URL
        short_url_api = aws_apigateway.LambdaRestApi(
            self, "HitterShortURLAPI", handler=short_url_handler)

        # Set the domain for a Short URL
        short_url_subdomain = "sl"
        short_url_domain_name = short_url_subdomain + '.' + ZONE_NAME
        short_url = 'https://' + short_url_domain_name

        # Using AWS ACM to create a certificate for a Short URL
        short_url_cert = aws_certificatemanager.DnsValidatedCertificate(
            self, 'HitterShortURLCertificate', domain_name=short_url_domain_name, hosted_zone=hosted_zone)

        # Add the domain name to the api and the A record to our hosted zone for a Short URL
        short_url_domain = short_url_api.add_domain_name(
            'HitterShortURLDomain', certificate=short_url_cert, domain_name=short_url_domain_name)

        # Set the A record for a Short URL
        aws_route53.ARecord(
            self, 'HitterShortURLARecord',
            record_name=short_url_subdomain,
            zone=hosted_zone,
            target=aws_route53.RecordTarget.from_alias(aws_route53_targets.ApiGatewayDomain(short_url_domain)))

        # Set the URL of the API Gateway to receive the shortened URL to an environment variable.
        bot_handler.add_environment('API_BASE_URL', short_url)

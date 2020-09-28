package main

import (
	"bytes"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/comprehend"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/translate"
	"github.com/guregu/dynamo"
)

type awsClient struct {
	session          *session.Session
	dynamoDBClient   *dynamo.DB
	comprehendClient *comprehend.Comprehend
	translateClient  *translate.Translate
	s3Client         *s3.S3
}

type mutexItem struct {
	ID   string
	TTL  int64
	Time time.Time
}

type urlItem struct {
	ID   string
	URL  string
	TTL  int64
	Time time.Time
}

type s3Item struct {
	bucket       string
	key          string
	objectExpiry string
	preSignedURL string
	urlExpiry    string
}

func newAwsClient() *awsClient {
	ac := &awsClient{}
	ac.session = session.Must(session.NewSession())
	ac.comprehendClient = comprehend.New(ac.session)
	ac.translateClient = translate.New(ac.session)
	ac.dynamoDBClient = dynamo.New(ac.session)
	ac.s3Client = s3.New(ac.session)

	return ac
}

func (c *awsClient) putURLItem(tableName string, id string, urlStr string, days int) (int64, error) {
	table := c.dynamoDBClient.Table(tableName)

	// put item
	now := time.Now()
	i := &urlItem{}
	i.ID = id
	i.URL = urlStr
	i.Time = now
	// TTL is per day.
	i.TTL = now.AddDate(0, 0, days).Unix()

	return i.TTL, table.Put(i).Run()
}

func (c *awsClient) getURLItem(tableName string, id string) (*urlItem, error) {
	table := c.dynamoDBClient.Table(tableName)

	// get item
	var result urlItem
	err := table.Get("ID", id).One(&result)

	return &result, err
}

func (c *awsClient) checkAndPutMutexItem(tableName string, id string) bool {
	// Check to see if there is anything running under the same ID.
	// If it exists, it returns true.
	item, err := c.getMutexItem(tableName, id)
	if err == nil {
		log.Println("[REJECTED] Already running under the same slack event ID: ", item.ID)
		return true
	}

	// Register the slack event ID before execution
	c.putMutexItem(tableName, id)

	return false
}

func (c *awsClient) putMutexItem(tableName string, id string) error {
	table := c.dynamoDBClient.Table(tableName)

	// put item
	now := time.Now()
	i := &mutexItem{}
	i.ID = id
	i.Time = now
	// Deleted after 24 hours.
	i.TTL = now.Add(24 * time.Hour).Unix()

	return table.Put(i).Run()
}

func (c *awsClient) getMutexItem(tableName string, id string) (*mutexItem, error) {
	table := c.dynamoDBClient.Table(tableName)

	// get item
	var result mutexItem
	err := table.Get("ID", id).One(&result)

	return &result, err
}

func (c *awsClient) detectLanguageCode(text string) (string, error) {
	input := &comprehend.BatchDetectDominantLanguageInput{}
	input.SetTextList([]*string{&text})

	output, err := c.comprehendClient.BatchDetectDominantLanguage(input)
	if err != nil {
		log.Println("[ERROR] Failed to aws comprehend detect language: ", err)
		return "", err
	}

	code := ""
	for _, i := range output.ResultList {
		for _, j := range i.Languages {
			if *j.LanguageCode != "" {
				code = *j.LanguageCode
			}
			break
		}
	}

	return code, nil
}

func (c *awsClient) translate(text string, source string, target string) (string, error) {
	input := &translate.TextInput{}
	input.SetSourceLanguageCode(source)
	input.SetTargetLanguageCode(target)
	input.SetText(text)

	output, err := c.translateClient.Text(input)
	if err != nil {
		log.Println("[ERROR] Failed to aws translate translation message: ", err)
		return "", err
	}

	return *output.TranslatedText, nil
}

func (c *awsClient) uploadAndPreSignedURL(bucket string, key string, body []byte, min int) (*s3Item, error) {
	result := &s3Item{}

	// Upload file to S3
	objectExpiry, err := c.uploadS3(bucket, key, body)
	if err != nil {
		return result, err
	}

	// Create pre-signed URL
	preURL, urlExpiry, err := c.createPreSignedURL(bucket, key, min)
	if err != nil {
		return result, err
	}

	// Set the result
	result.bucket = bucket
	result.key = key
	result.objectExpiry = objectExpiry
	result.preSignedURL = preURL
	result.urlExpiry = urlExpiry

	// Output debug log
	debug.Printf("s3Item: %+v\n", result)

	return result, nil
}

func (c *awsClient) uploadS3(bucket string, key string, body []byte) (string, error) {
	// Set the information of the object to be put
	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(bytes.NewReader(body)),
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	// Output debug log
	debug.Printf("input: %+v\n", input)

	// Uploading Objects to S3
	result, err := c.s3Client.PutObject(input)
	if err != nil {
		log.Println("[ERROR] Failed to upload object to S3: ", err)
		return "", err
	}

	// Output debug log
	debug.Printf("result: %+v\n", result)

	// Extract the expiry date
	// ex.) expiry-date="Tue, 15 Sep 2020 00:00:00 GMT", rule-id="YzhmY2RkZTUtYmM0OS00NTE5LWE3NjctODNjM2QwMTU2MDFm"
	text := *result.Expiration
	i := strings.Index(text, `"`)
	text = text[i+1:]
	i = strings.Index(text, `"`)
	text = text[:i]

	// Output debug log
	debug.Printf("expiry-date: %+v\n", text)

	dispDate, _ := getDisplayDateString(text, "")

	return dispDate, nil
}

func (c *awsClient) createPreSignedURL(bucket string, key string, min int) (string, string, error) {
	// Set the information of the object to be get
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	// Output debug log
	debug.Printf("input: %+v\n", input)

	// The default lifetime is 15 minutes.
	if min <= 0 {
		min = 15
	}

	// Output debug log
	debug.Printf("min: %+v\n", min)

	// Requesting a get object
	req, _ := c.s3Client.GetObjectRequest(input)
	if req.Error != nil {
		log.Println("[ERROR] Failed to get object request: ", req.Error)
		return "", "", req.Error
	}

	// Get pre-signed URL
	urlStr, err := req.Presign(time.Duration(min) * time.Minute)
	if err != nil {
		log.Println("[ERROR] Failed to sign request: ", err)
		return "", "", err
	}

	// Output debug log
	debug.Printf("urlStr: %+v\n", urlStr)

	u, err := url.Parse(urlStr)
	if err != nil {
		log.Println("[ERROR] Failed to parse pre-signed URL: ", err)
		return "", "", err
	}

	// Output debug log
	debug.Printf("RawQuery: %s\n", u.RawQuery)
	debug.Printf("RawQuery: %s\n", u.Query()["X-Amz-Date"][0])
	debug.Printf("RawQuery: %s\n", u.Query()["X-Amz-Expires"][0])

	tt := u.Query()["X-Amz-Date"][0]
	tt = strings.Replace(tt, "T", "", -1)
	tt = strings.Replace(tt, "Z", "", -1)

	dispDate, _ := getDisplayDateString(tt, u.Query()["X-Amz-Expires"][0])

	return urlStr, dispDate, nil
}

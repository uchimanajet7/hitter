package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type commandParameter struct {
	channel  string
	eventTs  string
	text     string
	to       string
	from     string
	command  string
	argument string
	files    map[string]string
	options  map[string][]string
}

func parseCommand(se *slackEvent) *commandParameter {
	cmdParam := &commandParameter{}
	cmdParam.channel = se.Event.Channel
	cmdParam.eventTs = se.Event.EventTs
	cmdParam.text = se.Event.Text
	cmdParam.from = se.Event.User
	cmdParam.options = make(map[string][]string)
	cmdParam.files = make(map[string]string)

	// If there are attachments, get the download URL.
	for _, f := range se.Event.Files {
		cmdParam.files[f.URLPrivateDownload] = f.Name
	}

	// Parse Text
	// <@W017HPXHDF0> hit 3 --ex <@W018217962V> --ex <@W017HPXHDF0> --ex <@W018217962V>
	// "Mention to bot" "command" "argument" "options"
	// You can only parse the specified format.
	// If the command cannot be executed because the parsing fails, the input string is posted to Slack as an error message.
	// URLs are enclosed in "<>" like this: "<https://docs.aws.amazon.com/cdk/api/latest/>"
	r := regexp.MustCompile(`<@([A-Z0-9]{11,11})>`)
	prevKey := ""

	items := strings.Split(cmdParam.text, " ")
	for i, str := range items {
		str = strings.TrimSpace(str)

		// Output debug log
		debug.Printf("i: %+v\n", i)
		debug.Printf("str: %+v\n", str)
		debug.Printf("id match: %+v\n", r.MatchString(str))

		if str == "" {
			continue
		}
		// A regular expression user ID match
		if r.MatchString(str) {
			// <@W017HPXHDF0>
			// Delete the first two characters and the last one
			str = str[2 : len(str)-1]
		}

		// The first string is a mension to a bot
		if i == 0 {
			cmdParam.to = str
			continue
		}

		// The second string is the command
		if i == 1 {
			cmdParam.command = str
			continue
		}

		// The third string is the argument to the command
		if i == 2 {
			// A specific command is input as an argument for all of the following
			if cmdParam.command == "translate" {
				cmdParam.argument = strings.Join(items[i:], " ")
				break
			}

			// URL string is escaped, reconfigure
			if cmdParam.command == "short" {
				// Retrieve the entered URL
				for j, b := range se.Event.Blocks {
					// Output debug log
					debug.Printf("index : %+v\n", j)
					debug.Printf("value: %+v\n", b)

					for k, e := range b.Elements {
						// Output debug log
						debug.Printf("index: %+v\n", k)
						debug.Printf("value: %+v\n", e)

						for l, t := range e.Elements {
							// Output debug log
							debug.Printf("index: %+v\n", l)
							debug.Printf("value: %+v\n", t)

							if t.Type == "link" {
								debug.Printf("Type: %+v\n", t.Type)
								debug.Printf("URL: %+v\n", t.URL)

								cmdParam.argument = t.URL
								break
							}
						}
					}
				}
				continue
			}

			cmdParam.argument = str
			continue
		}

		// Optional strings after the fourth are
		// Options are strings that begin with "--"
		if strings.HasPrefix(str, "--") {
			prevKey = str
			continue
		}

		// The value of the option is a string beginning with "--" followed by
		if strings.HasPrefix(prevKey, "--") {
			if _, ok := cmdParam.options[prevKey]; ok {
				cmdParam.options[prevKey] = append(cmdParam.options[prevKey], str)
			} else {
				cmdParam.options[prevKey] = []string{str}
			}
			prevKey = ""

			continue
		}
	}

	// Output debug log
	debug.Printf("cmdParam: %+v\n", cmdParam)

	return cmdParam
}

func (c *commandParameter) runCommand(sc *slackClient, aws *awsClient) error {
	var err error

	// Determine which commands are entered and execute them individually.
	switch c.command {
	case "help":
		log.Println("[COMMAND] Run help command")
		err = sc.notifyHelpSuccess(c)
	case "hit":
		log.Println("[COMMAND] Run hit command")
		err = c.runHitCommand(sc)
	case "translate":
		log.Println("[COMMAND] Run translate command")
		err = c.runTranslateCommand(sc, aws)
	case "link":
		log.Println("[COMMAND] Run link command")
		err = c.runLinkCommand(sc, aws)
	case "short":
		log.Println("[COMMAND] Run short command")
		err = c.runShortCommand(sc, aws)
	default:
		log.Println("[COMMAND] The target command was not available:", c.command)
		err = sc.notifyHelpSuccess(c)
	}

	if err != nil {
		err = sc.notifyError(c, fmt.Sprintf("Command execution failed. *[%s]*", err))
	}

	return err
}

func (c *commandParameter) runShortCommand(sc *slackClient, aws *awsClient) error {
	// Getting information on environment variables
	table := envconf.URLTableName
	baseURL := envconf.APIBaseURL

	// Generate a random UUID V4
	uuid4, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	// Create a unique id (take first 8 chars)
	// https://github.com/aws-samples/aws-cdk-examples/blob/bb182579313dca6a91c630a116fc66cc3921f412/python/url-shortener/lambda/handler.py#L40
	urlID := uuid4.String()[:8]

	// Parse the URLs for the API and create shortened URLs
	u, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, urlID)

	// Output debug log
	debug.Printf("u: %+v\n", u.String())

	// Get the value of a command option
	val, ok := c.options["--ttl"]

	// TTL is 1 day by default.
	ttl := 1
	if ok {
		num, err := strconv.Atoi(strings.TrimSpace(val[0]))
		if err == nil {
			ttl = num
		}
	}

	// Output debug log
	debug.Printf("ttl: %+v\n", ttl)

	// Setting Information in the Mapping Table
	unixTime, err := aws.putURLItem(table, urlID, strings.TrimSpace(c.argument), ttl)
	if err != nil {
		return err
	}

	// Get an expiration date for display
	dateStr := strconv.FormatInt(unixTime, 10)
	dateStr, _ = getDisplayDateString(dateStr, "")

	// Notify your slack of the results
	return sc.notifyShortSuccess(c, u.String(), dateStr)
}

func (c *commandParameter) runLinkCommand(sc *slackClient, aws *awsClient) error {
	// Getting information on environment variables
	bucket := envconf.S3BucketName

	// Generate a random UUID V4
	uuid4, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	// Get a validity period
	// The default is 15 minutes.
	min := 15
	if c.argument != "" {
		num, err := strconv.Atoi(strings.TrimSpace(c.argument))
		if err == nil {
			min = num
		}
	}

	var results []*s3Item
	for k, v := range c.files {
		// Output debug log
		debug.Printf("downlodURL: %+v\n", k)

		// Downloading files from slack
		wb, err := sc.downloadFile(k)
		if err != nil {
			return err
		}

		key := path.Join(uuid4.String(), v)

		// Output debug log
		debug.Printf("bucket: %+v\n", bucket)
		debug.Printf("key: %+v\n", key)

		// Upload file and create pre-signed URL
		s3Item, err := aws.uploadAndPreSignedURL(bucket, key, wb, min)
		if err != nil {
			return err
		}
		results = append(results, s3Item)
	}

	// Notify your slack of the results
	return sc.notifyLinkSuccess(c, results)
}

func (c *commandParameter) runTranslateCommand(sc *slackClient, aws *awsClient) error {
	// Get the language code of the input text.
	source, err := aws.detectLanguageCode(c.argument)
	if err != nil {
		return err
	}

	// Determine the language code to translate
	// https://docs.aws.amazon.com/ja_jp/translate/latest/dg/what-is.html#what-is-languages
	target := "ja"
	if source == "ja" {
		target = "en"
	}

	// Translate the text
	translated, err := aws.translate(c.argument, source, target)
	if err != nil {
		return err
	}

	// Notify your slack of the results
	return sc.notifyTranslateSuccess(c, c.argument, translated, source, target)
}

func (c *commandParameter) runHitCommand(sc *slackClient) error {
	// Get the value of a command option
	val, _ := c.options["--ex"]

	// Get the target users
	users, err := sc.getTargetUsers(c.channel, val)
	if err != nil {
		return err
	}

	// Minimum number of draws is 1
	num := 1

	// If the number of lots is specified in the argument
	if c.argument != "" {
		i, err := strconv.Atoi(strings.TrimSpace(c.argument))
		if err == nil {
			num = i
		}
	}

	// There are more choices than options.
	if len(users) < num {
		text := fmt.Sprintf("There are too many choices: %d/%d", num, len(users))
		log.Println("[ERROR] " + text)
		return errors.New(text)
	}

	// If there is a tie, return as is.
	if len(users) == num {
		log.Println("[SUCCESS] It worked, as there were an equal number of options.")
		return sc.notifyHitSuccess(c, users)
	}

	// Select the specified number of choices at random
	rand.Seed(time.Now().UnixNano())
	selectedMap := map[string]struct{}{}
	for {
		selectedMap[users[rand.Intn(len(users))]] = struct{}{}
		if len(selectedMap) == num {
			break
		}
	}

	// Output debug log
	debug.Printf("selectedMap: %+v\n", selectedMap)

	results := []string{}
	for k, _ := range selectedMap {
		results = append(results, k)
	}

	// Notify your slack of the results
	return sc.notifyHitSuccess(c, results)
}

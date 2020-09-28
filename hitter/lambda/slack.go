package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/slack-go/slack"
)

type slackClient struct {
	client *slack.Client
}

// automatically generated using the following
// https://mholt.github.io/json-to-go/
type slackEvent struct {
	// only url_verification event
	// https://api.slack.com/events/url_verification
	Challenge string `json:"challenge"`

	Token        string `json:"token"`
	TeamID       string `json:"team_id"`
	EnterpriseID string `json:"enterprise_id"`
	APIAppID     string `json:"api_app_id"`

	Event struct {
		ClientMsgID  string `json:"client_msg_id"`
		Type         string `json:"type"`
		Text         string `json:"text"`
		User         string `json:"user"`
		Ts           string `json:"ts"`
		Team         string `json:"team"`
		DisplayAsBot bool   `json:"display_as_bot"`
		Channel      string `json:"channel"`
		EventTs      string `json:"event_ts"`
		Upload       bool   `json:"upload"`
		Subtype      string `json:"subtype"`
		Username     string `json:"username"`
		BotID        string `json:"bot_id"`

		Files []struct {
			ID                 string `json:"id"`
			Created            int    `json:"created"`
			Timestamp          int    `json:"timestamp"`
			Name               string `json:"name"`
			Title              string `json:"title"`
			Mimetype           string `json:"mimetype"`
			Filetype           string `json:"filetype"`
			PrettyType         string `json:"pretty_type"`
			User               string `json:"user"`
			Editable           bool   `json:"editable"`
			Size               int    `json:"size"`
			Mode               string `json:"mode"`
			IsExternal         bool   `json:"is_external"`
			ExternalType       string `json:"external_type"`
			IsPublic           bool   `json:"is_public"`
			PublicURLShared    bool   `json:"public_url_shared"`
			DisplayAsBot       bool   `json:"display_as_bot"`
			Username           string `json:"username"`
			URLPrivate         string `json:"url_private"`
			URLPrivateDownload string `json:"url_private_download"`
			Thumb64            string `json:"thumb_64"`
			Thumb80            string `json:"thumb_80"`
			Thumb360           string `json:"thumb_360"`
			Thumb360W          int    `json:"thumb_360_w"`
			Thumb360H          int    `json:"thumb_360_h"`
			Thumb480           string `json:"thumb_480"`
			Thumb480W          int    `json:"thumb_480_w"`
			Thumb480H          int    `json:"thumb_480_h"`
			Thumb160           string `json:"thumb_160"`
			OriginalW          int    `json:"original_w"`
			OriginalH          int    `json:"original_h"`
			ThumbTiny          string `json:"thumb_tiny"`
			Permalink          string `json:"permalink"`
			PermalinkPublic    string `json:"permalink_public"`
			IsStarred          bool   `json:"is_starred"`
			HasRichPreview     bool   `json:"has_rich_preview"`
		} `json:"files"`

		Blocks []struct {
			Type     string `json:"type"`
			BlockID  string `json:"block_id"`
			Elements []struct {
				Type     string `json:"type"`
				Elements []struct {
					Type   string `json:"type"`
					UserID string `json:"user_id"`
					Text   string `json:"text"`
					URL    string `json:"url"`
				} `json:"elements"`
			} `json:"elements"`
		} `json:"blocks"`
	} `json:"event"`

	Type        string   `json:"type"`
	EventID     string   `json:"event_id"`
	EventTime   int      `json:"event_time"`
	AuthedUsers []string `json:"authed_users"`
}

// https://api.slack.com/events/url_verification
/*
{
    "token": "Jhj5dZrVaK7ZwHHjRyZWjbDl",
    "challenge": "3eZbrw1aBm2rZgRNFdxV2595E9CY3gmdALWMmHkvFXO7tYXAYM8P",
    "type": "url_verification"
}
*/

// https://api.slack.com/events-api#receiving_events
// https://api.slack.com/types/event
// https://api.slack.com/events/app_mention
/*
{
    "token": "ZZZZZZWSxiZZZ2yIvs3peJ",
    "team_id": "T061EG9R6",
    "api_app_id": "A0MDYCDME",
    "event": {
        "type": "app_mention",
        "user": "U061F7AUR",
        "text": "What is the hour of the pearl, <@U0LAN0Z89>?",
        "ts": "1515449522.000016",
        "channel": "C0LAN2Q65",
        "event_ts": "1515449522000016"
    },
    "type": "event_callback",
    "event_id": "Ev0LAN670R",
    "event_time": 1515449522000016,
    "authed_users": [
        "U0LAN0Z89"
    ]
}
*/

type stateEnum int

const (
	successState stateEnum = iota
	failureState
	helpState
)

func newSlackClient(token string) *slackClient {
	sc := &slackClient{}
	sc.client = slack.New(token)

	return sc
}

func (c *slackClient) parseEvent(eventJSON string) (*slackEvent, string, error) {
	text := ""
	se := &slackEvent{}

	// Output debug log
	debug.Printf("eventJSON: %+v\n", eventJSON)

	err := json.Unmarshal([]byte(eventJSON), se)
	if err != nil {
		log.Println("[ERROR] Failed to parse the slack event JSON.: ", err)
		return se, text, err
	}

	// Use the parsed information to perform a check.
	// Check slack verification token
	if se.Token != envconf.SlackVerificationToken {
		log.Println("[REJECTED] The token received does not match the verification token: ", se.Token)
		text = `{"message": "[REJECTED] The token received does not match the verification token"}`
		return se, text, err
	}

	// Accept slack url_verification event
	// https://api.slack.com/events/url_verification
	if se.Type == "url_verification" {
		log.Println("[ACCEPTED] Slack url_verification event")
		text = fmt.Sprintf(`{"challenge": %s}`, se.Challenge)
		return se, text, err
	}

	// Respond only to specific events
	if se.Event.Type != "app_mention" {
		log.Println("[REJECTED] Slack event type do not 'app_mention': ", se.Event.Type)
		text = `{"message": "[REJECTED] Slack event type do not 'app_mention'"}`
		return se, text, err
	}

	// filter the channel?
	if envconf.SlackChannelID != "" {
		if envconf.SlackChannelID != se.Event.Channel {
			log.Println("[REJECTED] Slack channel ID do not match: ", se.Event.Channel)
			text = `{"message": "[REJECTED] Slack channel ID do not match"}`
			return se, text, err
		}
	}

	return se, text, err
}

func (c *slackClient) getTargetUsers(channelID string, exclusions []string) ([]string, error) {
	// Get a list of users who have joined the channel
	users, err := c.getUsers(channelID, 1000)
	if err != nil {
		return nil, err
	}

	// Separate the user list into bots and people.
	userIds, _, err := c.classifyUsers(users...)
	if err != nil {
		return nil, err
	}

	// Do you have a list of users not eligible for the lottery?
	choices := append([]string{}, userIds...)
	for _, x := range exclusions {
		ret := make([]string, len(choices))
		i := 0
		for _, t := range choices {
			if x != t {
				ret[i] = t
				i++
			}
		}
		choices = ret[:i]
	}

	// Output debug log
	debug.Printf("exclusions: %+v\n", exclusions)
	debug.Printf("choices: %+v\n", choices)

	return choices, nil
}

func (c *slackClient) uploadFile(channel string, body []byte, filename string, comment string, ts string) error {
	params := slack.FileUploadParameters{}
	params.Reader = bytes.NewReader(body)
	params.Channels = []string{channel}
	params.Filetype = "text"
	params.Filename = filename
	params.InitialComment = comment
	params.ThreadTimestamp = ts

	// Output debug log
	debug.Printf("params: %+v\n", params)

	// Uploading files to a thread
	f, err := c.client.UploadFile(params)

	// Output debug log
	debug.Printf("file: %+v\n", f)

	return err
}

func (c *slackClient) downloadFile(url string) ([]byte, error) {
	var wb bytes.Buffer

	// Downloading files from slack
	err := c.client.GetFile(url, &wb)

	return wb.Bytes(), err
}

func (c *slackClient) getUsers(channelID string, limit int) ([]string, error) {
	// Get all users involved in the conversation
	// https://api.slack.com/methods/conversations.members
	param := &slack.GetUsersInConversationParameters{}
	param.ChannelID = channelID
	param.Cursor = ""
	// If the limit is greater than zero, set it.
	if limit > 0 {
		param.Limit = limit
	}

	var users []string
	for {
		list, next, err := c.client.GetUsersInConversation(param)

		// Output debug log
		debug.Printf("list: %+v\n", list)
		debug.Printf("next: %+v\n", next)
		debug.Printf("err: %+v\n", err)
		debug.Printf("users: %+v\n", users)

		if err != nil {
			log.Println("[ERROR] Failed to retrieve the user list: ", err)
			return users, err
		}

		users = append(users, list...)

		if next == "" {
			break
		}

		param.Cursor = next
	}

	// Output debug log
	debug.Printf("users: %+v\n", users)

	return users, nil
}

func (c *slackClient) classifyUsers(ids ...string) ([]string, []string, error) {
	var botIds []string
	var userIds []string

	// Retrieving User Information from a User ID
	list, err := c.client.GetUsersInfo(ids...)
	if err != nil {
		log.Println("[ERROR] Failed to retrieve user information: ", err)
		return userIds, botIds, err
	}

	// Classify Bot and User IDs
	for _, item := range *list {
		if item.IsBot {
			botIds = append(botIds, item.ID)
		} else {
			userIds = append(userIds, item.ID)
		}
	}

	// Output debug log
	debug.Printf("botIds: %+v\n", botIds)
	debug.Printf("userIds: %+v\n", userIds)

	return userIds, botIds, nil
}

func (c *slackClient) createSummarySection(mention string, state stateEnum) *slack.SectionBlock {
	// Mentions to the commander and results summary section
	var text string

	switch state {
	case successState:
		text = ":confetti_ball: I successfully executed the requested command."
	case failureState:
		text = ":rotating_light: I failed to execute the requested command."
	case helpState:
		text = ":thinking_face: Please check the following command help."
	default:
		// unidentified
		text = ":eyes: An exempt designation has been made."
	}

	if mention != "" {
		// Adding Mentions to a Target
		text = "<@" + mention + "> \n" + text
	}

	summaryText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	summarySection := slack.NewSectionBlock(summaryText, nil, nil)

	return summarySection
}

func (c *slackClient) createInfoSection(cmd string, eventTs string) *slack.SectionBlock {
	// Input information Section
	// ex.)1595673533.002100
	dispDate, _ := getDisplayDateString(strings.Replace(eventTs, ".", "", -1), "")

	// They say the maximum character count is about 4,000 characters.
	// https://www.cotegg.com/blog/?p=1951#result
	// The input and output strings are 4,000 characters in total, so we'll omit them in about half.
	text := cmd
	if utf8.RuneCountInString(text) > 2000 {
		r := []rune(text)
		text = string(r[:1950]) + "...(omitted)"
	}

	text = "*Command:*\n```" + text + "```\n:clock8: " + dispDate
	infoText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	infoSection := slack.NewSectionBlock(infoText, nil, nil)

	return infoSection
}

func (c *slackClient) notifyMessage(channel string, option slack.MsgOption) (string, string, error) {
	// Sending a message to slack
	// https://api.slack.com/methods/chat.postMessage
	channelID, timestamp, err := c.client.PostMessage(channel, option)
	if err != nil {
		log.Println("[ERROR] The notification to slack failed.: ", channelID, timestamp, err)
		return "", "", err
	}

	return channelID, timestamp, err
}

func (c *slackClient) notifyError(cp *commandParameter, message string) error {
	// dividing line section
	divSection := slack.NewDividerBlock()

	// Get summary section
	summarySection := c.createSummarySection(cp.from, failureState)

	// Get input information Section
	infoSection := c.createInfoSection(cp.text, cp.eventTs)

	// Command Execution Error Result Section
	text := "*Results:*\n:name_badge: " + message + "\n> :warning: _Be sure to check the help if you want to rerun the command._"
	resultText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	resultSection := slack.NewSectionBlock(resultText, nil, nil)

	// Build Message with blocks created above
	msgOption := slack.MsgOptionBlocks(
		summarySection,
		divSection,
		infoSection,
		divSection,
		resultSection,
		divSection,
	)

	// Notify your slack of the results
	_, _, err := c.notifyMessage(cp.channel, msgOption)
	if err == nil {
		log.Println("[NOTICE] Notify slack of a command execution error.")
	}

	return err
}

func (c *slackClient) notifyHelpSuccess(cp *commandParameter) error {
	// dividing line section
	divSection := slack.NewDividerBlock()

	// Get summary section
	summarySection := c.createSummarySection(cp.from, helpState)

	// Create Help Details
	text := ""

	// hit command help
	text = ":book: *hit*\n"
	text = text + "```"
	text = text + "DESCRIPTION: \n"
	text = text + " • Randomly select from the members in the channel\n"
	text = text + "SYNOPSIS: \n"
	text = text + " • @hitter hit <Number> [<Options> ...]\n"
	text = text + "OPTIONS: \n"
	text = text + " • --ex <User>\n"
	text = text + "EXAMPLES: \n"
	text = text + " • @hitter hit 2\n"
	text = text + " • @hitter hit 3 --ex @userA --ex @userB\n"
	text = text + "```\n"

	// translate command help
	text = text + "\n:book: *translate*\n"
	text = text + "```"
	text = text + "DESCRIPTION: \n"
	text = text + " • Translates the input text\n"
	text = text + "SYNOPSIS: \n"
	text = text + " • @hitter translate <Text>\n"
	text = text + "EXAMPLES: \n"
	text = text + " • @hitter translate AWS is the world’s most comprehensive and broadly adopted cloud platform\n"
	text = text + " • @hitter translate AWS は、世界で最も包括的で広く採用されているクラウドプラットフォームです\n"
	text = text + "```\n"

	// link command help
	text = text + "\n:book: *link*\n"
	text = text + "```"
	text = text + "DESCRIPTION: \n"
	text = text + " • Upload the attached file to Amazon S3 and generate a pre-signed URL\n"
	text = text + "SYNOPSIS: \n"
	text = text + " • @hitter link <Expiry Minutes> <Files>\n"
	text = text + "EXAMPLES: \n"
	text = text + " • @hitter link <file1>\n"
	text = text + " • @hitter link 15 <fileA, fileB>\n"
	text = text + "```\n"

	// short command help
	text = text + "\n:book: *short*\n"
	text = text + "```"
	text = text + "DESCRIPTION: \n"
	text = text + " • Generate a shortened URL\n"
	text = text + "SYNOPSIS: \n"
	text = text + " • @hitter short <URL> <Options>\n"
	text = text + "OPTIONS: \n"
	text = text + " • --ttl <Expiry Days>\n"
	text = text + "EXAMPLES: \n"
	text = text + " • @hitter short https://aws.amazon.com/jp/\n"
	text = text + " • @hitter short https://aws.amazon.com/jp/ --ttl 7\n"
	text = text + "```\n"

	text = "*Commands:*\n" + text + "\n> :information_source: _See the documentation if you need more details._"
	detailText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	detailSection := slack.NewSectionBlock(detailText, nil, nil)

	// Build Message with blocks created above
	msgOption := slack.MsgOptionBlocks(
		summarySection,
		divSection,
		detailSection,
		divSection,
	)

	// Notify your slack of the results
	_, _, err := c.notifyMessage(cp.channel, msgOption)
	if err == nil {
		log.Println("[NOTICE] Notify slack of the result of the hit command.")
	}

	return err
}

func (c *slackClient) notifyHitSuccess(cp *commandParameter, result []string) error {
	// dividing line section
	divSection := slack.NewDividerBlock()

	// Get summary section
	summarySection := c.createSummarySection(cp.from, successState)

	// Get input information Section
	infoSection := c.createInfoSection(cp.text, cp.eventTs)

	// Command Execution Result Section
	text := ""
	for i, v := range result {
		num := strconv.Itoa(i + 1)
		text = text + ":tada: *[" + num + "]:*  <@" + v + "> You are the *" + num + "th* choice.\n\n"
	}
	text = "*Results:*\n" + text + "\n> :zap: _If you have a problem with your choice, please try again._"
	resultText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	resultSection := slack.NewSectionBlock(resultText, nil, nil)

	// Build Message with blocks created above
	msgOption := slack.MsgOptionBlocks(
		summarySection,
		divSection,
		infoSection,
		divSection,
		resultSection,
		divSection,
	)

	// Notify your slack of the results
	_, _, err := c.notifyMessage(cp.channel, msgOption)
	if err == nil {
		log.Println("[NOTICE] Notify slack of the result of the hit command.")
	}

	return err
}

func (c *slackClient) notifyTranslateSuccess(cp *commandParameter, source string, translated string, sourceLangCode string, translatedLangCode string) error {
	// dividing line section
	divSection := slack.NewDividerBlock()

	// Get summary section
	summarySection := c.createSummarySection(cp.from, successState)

	// Get input information Section
	infoSection := c.createInfoSection(cp.text, cp.eventTs)

	// Command Execution Result Section
	text := "*Results:*\n:dart: Translated the text from *[" + sourceLangCode + "]* to *[" + translatedLangCode + "]*\n\n`Please check the file attached to the thread for details of the translation command results.`\n\n> :zap: _If there is a problem with the translation, please check the input text and try again._"
	resultText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	resultSection := slack.NewSectionBlock(resultText, nil, nil)

	// Build Message with blocks created above
	msgOption := slack.MsgOptionBlocks(
		summarySection,
		divSection,
		infoSection,
		divSection,
		resultSection,
		divSection,
	)

	// Notify your slack of the results
	ch, ts, err := c.notifyMessage(cp.channel, msgOption)
	if err != nil {
		return err
	}

	// Organize the output to a file
	body := "• Source text: [" + sourceLangCode + "]\n\n"
	body = body + source + "\n\n\n\n"
	body = body + "• Translated text: [" + translatedLangCode + "]\n\n"
	body = body + translated + "\n"

	// Organize file names
	dateStr, _ := getFileNameDateString(strings.Replace(ts, ".", "", -1))
	filename := dateStr + "_translate_command_result.text"

	// Organize file comment
	comment := ":dart: This file is the result of the translation command.\n"

	// Return results in an attachment, taking into account the character limit.
	err = c.uploadFile(ch, []byte(body), filename, comment, ts)
	if err == nil {
		log.Println("[NOTICE] Notify and upload file slack of the result of the translate command.")
	}

	return err
}

func (c *slackClient) notifyLinkSuccess(cp *commandParameter, results []*s3Item) error {
	// dividing line section
	divSection := slack.NewDividerBlock()

	// Get summary section
	summarySection := c.createSummarySection(cp.from, successState)

	// Get input information Section
	values := []string{}
	for _, v := range cp.files {
		values = append(values, v)
	}
	infoSection := c.createInfoSection(cp.text+" <"+strings.Join(values, ", ")+">", cp.eventTs)

	// Command Execution Result Section
	text := ""
	if len(results) > 1 {
		text = strconv.Itoa(len(results)) + " files "
	}
	text = "*Results:*\n:linked_paperclips: " + text + "S3 Object information and Pre-Signed URL\n\n`Please check the file attached to the thread for details of the link command results.`\n\n> :satellite_antenna: _If you want to change the expiry date, please try again._"
	resultText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	resultSection := slack.NewSectionBlock(resultText, nil, nil)

	// Build Message with blocks created above
	msgOption := slack.MsgOptionBlocks(
		summarySection,
		divSection,
		infoSection,
		divSection,
		resultSection,
		divSection,
	)

	// Notify your slack of the results
	ch, ts, err := c.notifyMessage(cp.channel, msgOption)

	// Organize the output to a file
	body := ""
	for i, v := range results {
		num := strconv.Itoa(i + 1)
		body = body + "• [" + num + "]: S3 Object information and Pre-Signed URL\n\n\n"
		body = body + "S3 Object: \n"
		body = body + " • bucket: \n"
		body = body + "      " + v.bucket + "\n\n"
		body = body + " • key: \n"
		body = body + "      " + v.key + "\n\n"
		body = body + " • expiry date: \n"
		body = body + "      " + v.objectExpiry + "\n\n\n"
		body = body + "Pre-Signed URL: \n"
		body = body + " • URL: \n"
		body = body + "      " + v.preSignedURL + "\n\n"
		body = body + " • expiry date: \n"
		body = body + "      " + v.urlExpiry + "\n\n\n\n"
	}

	// Organize file names
	dateStr, _ := getFileNameDateString(strings.Replace(ts, ".", "", -1))
	filename := dateStr + "_link_command_result.text"

	// Organize file comment
	comment := ":linked_paperclips: This file is the result of the link command.\n"

	// Return results in an attachment, taking into account the character limit.
	err = c.uploadFile(ch, []byte(body), filename, comment, ts)
	if err == nil {
		log.Println("[NOTICE] Notify and upload file slack of the result of the link command.")
	}

	return err
}

func (c *slackClient) notifyShortSuccess(cp *commandParameter, urlStr string, dateStr string) error {
	// dividing line section
	divSection := slack.NewDividerBlock()

	// Get summary section
	summarySection := c.createSummarySection(cp.from, successState)

	// Get input information Section
	infoSection := c.createInfoSection(cp.text, cp.eventTs)

	// Command Execution Result Section
	text := ":link: " + urlStr + "\n\n"
	text = text + ":clock930: " + dateStr + "\n"
	text = "*Results:*\n" + text + "\n> :globe_with_meridians: _If you want to change the expiry date, please try again._"
	resultText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	resultSection := slack.NewSectionBlock(resultText, nil, nil)

	// Build Message with blocks created above
	msgOption := slack.MsgOptionBlocks(
		summarySection,
		divSection,
		infoSection,
		divSection,
		resultSection,
		divSection,
	)

	// Notify your slack of the results
	_, _, err := c.notifyMessage(cp.channel, msgOption)
	if err == nil {
		log.Println("[NOTICE] Notify slack of the result of the short command.")
	}

	return err
}

package function

import (
	"encoding/json"
	"fmt"
	"os"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/bwmarrin/discordgo"
	handler "github.com/openfaas-incubator/go-function-sdk"
	"go.uber.org/zap"
	"gopkg.in/go-playground/webhooks.v5/github"
)

var (
	ColorRed   = 0xd40b30
	ColorGreen = 0x4fa96a
)

// Handle a function invocation
func Handle(req handler.Request) (handler.Response, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return httpError(500, err)
	}

	// check for the github event header
	event := req.Header.Get("X-GitHub-Event")
	if event == "" {
		logger.Error("missing X-GitHub-Event header")
		return httpError(400, fmt.Errorf("missing X-GitHub-Event header"))
	}

	logger.Info("got X-GitHub-Event header", zap.String("event", event))
	gitHubEvent := github.Event(event)

	// get the channel's ID
	channelID := os.Getenv("DISCORD_CHANNEL_ID")

	// create a new Discord session using the provided bot token
	ds, err := discordgo.New("Bot " + os.Getenv("DISCORD_TOKEN"))
	if err != nil {
		return httpError(500, err)
	}
	defer ds.Close()

	// open a websocket connection to Discord and begin listening
	if err := ds.Open(); err != nil {
		return httpError(500, err)
	}

	// figure out what to do based on the event
	switch gitHubEvent {
	case github.IssuesEvent:
		payload := github.IssuesPayload{}
		if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
			return httpError(500, err)
		}

		color := ColorGreen

		if payload.Action == "edited" && payload.Issue.State == "closed" {
			payload.Action = "closed"
			color = ColorRed
		}

		embed := NewEmbed().
			SetTitle(
				"Issue %s by %s",
				payload.Action,
				payload.Issue.User.Login,
			).
			SetAuthor(
				payload.Issue.User.Login,
				payload.Issue.User.AvatarURL,
				payload.Issue.User.URL,
			).
			AddField("State", payload.Issue.State).
			SetColor(color)

		if payload.Issue.Assignee != nil {
			embed = embed.AddField(
				"Assignee",
				payload.Issue.Assignee.Login,
			)
		}

		if payload.Action != "closed" {
			embed = embed.SetDescription(
				"__[%s](%s)__\n\n%s",
				payload.Issue.Title,
				payload.Issue.URL,
				mdConvert(payload.Issue.Body),
			)
		}

		embed = embed.InlineAllFields()
		embed = embed.Truncate()

		if _, err := ds.ChannelMessageSendEmbed(
			channelID,
			embed.MessageEmbed,
		); err != nil {
			return httpError(500, err)
		}

	case github.PullRequestEvent:
		payload := github.PullRequestPayload{}
		if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
			return httpError(500, err)
		}

		color := ColorGreen

		if payload.Action == "edited" && payload.PullRequest.State == "closed" {
			payload.Action = "closed"
			color = ColorRed
		}

		embed := NewEmbed().
			SetTitle(
				"Pull request %s by %s",
				payload.Action,
				payload.PullRequest.User.Login,
			).
			SetAuthor(
				payload.PullRequest.User.Login,
				payload.PullRequest.User.AvatarURL,
				payload.PullRequest.User.URL,
			).
			AddField("State", payload.PullRequest.State).
			SetColor(color)

		if payload.PullRequest.Assignee != nil {
			embed = embed.AddField(
				"Assignee",
				payload.PullRequest.Assignee.Login,
			)
		}

		if payload.Action != "closed" {
			embed = embed.SetDescription(
				"__[%s](%s)__\n\n%s",
				payload.PullRequest.Title,
				payload.PullRequest.URL,
				mdConvert(payload.PullRequest.Body),
			)
		}

		embed = embed.InlineAllFields()
		embed = embed.Truncate()

		if _, err := ds.ChannelMessageSendEmbed(
			channelID,
			embed.MessageEmbed,
		); err != nil {
			return httpError(500, err)
		}

	default:
		return httpOk("We don't care about this event, all is good.")
	}

	return httpOk("Ok!")
}

func httpOk(body string) (handler.Response, error) {
	return handler.Response{
		Body:       []byte(body),
		StatusCode: 200,
	}, nil
}

func httpError(code int, err error) (handler.Response, error) {
	return handler.Response{
		Body:       []byte(err.Error()),
		StatusCode: code,
	}, err
}

func mdConvert(body string) string {
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(body)
	if err != nil {
		return body
	}
	return markdown
}

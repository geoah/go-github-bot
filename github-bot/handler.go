package function

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/bwmarrin/discordgo"
	handler "github.com/openfaas/templates-sdk/go-http"
	"go.uber.org/zap"
	"gopkg.in/go-playground/webhooks.v5/github"
)

var (
	ColorRed    = 0xd72b2b
	ColorYellow = 0xf2c359
	ColorBlue   = 0x5798d2
	ColorGreen  = 0x31b967
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

	// parse query string
	query, err := url.ParseQuery(req.QueryString)
	if err != nil {
		return httpError(400, err)
	}

	// get the bot and channel info
	channelID := query.Get("discordChannelID")
	botToken := query.Get("discordBotToken")

	// create a new Discord session using the provided bot token
	ds, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return httpError(500, err)
	}
	defer ds.Close()

	// TODO this is supposed to only be needed the first time around?
	// open a websocket connection to Discord and begin listening
	// if err := ds.Open(); err != nil {
	// 	return httpError(500, err)
	// }

	// figure out what to do based on the event
	switch gitHubEvent {

	case github.IssuesEvent:
		payload := github.IssuesPayload{}
		if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
			return httpError(500, err)
		}

		color := ColorGreen
		switch payload.Action {
		case "opened":
		case "closed":
			color = ColorRed
		case "edited":
			color = ColorYellow
			if payload.Issue.State == "closed" {
				payload.Action = "closed"
				color = ColorRed
			}
		default:
			return httpOk("Don't care about this action.")
		}

		header := fmt.Sprintf(
			"Issue %s by %s",
			payload.Action,
			payload.Issue.User.Login,
		)

		embed := NewEmbed().
			SetTitle(
				"#%d %s",
				payload.Issue.Number,
				payload.Issue.Title,
			).
			SetURL(
				payload.Issue.HTMLURL,
			).
			SetFooter(
				payload.Issue.User.Login,
				fmt.Sprintf(
					"https://github.com/%s.png?size=40",
					payload.Issue.User.Login,
				),
			).
			SetColor(color)

		if payload.Action != "closed" {
			embed = embed.SetDescription(
				mdConvert(payload.Issue.Body),
			)
		}

		embed = embed.Truncate()

		if _, err := ds.ChannelMessageSendComplex(
			channelID,
			&discordgo.MessageSend{
				Content: header,
				Embed:   embed.MessageEmbed,
			},
		); err != nil {
			return httpError(500, err)
		}

	case github.PullRequestEvent:
		payload := github.PullRequestPayload{}
		if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
			return httpError(500, err)
		}

		header := ""

		color := ColorGreen
		switch payload.Action {
		case "opened":
		case "closed":
			color = ColorRed
			if payload.PullRequest.Merged {
				payload.Action = "merged"
				color = ColorBlue
			}
		case "synchronize":
			color = ColorYellow
			header = fmt.Sprintf(
				"Pull request updated by %s",
				payload.PullRequest.User.Login,
			)
		case "edited":
			color = ColorYellow
			if payload.PullRequest.State == "closed" {
				header = fmt.Sprintf(
					"Pull request closed by %s",
					payload.PullRequest.User.Login,
				)
				color = ColorRed
			}
		default:
			return httpOk("Don't care about this action.")
		}

		if header == "" {
			header = fmt.Sprintf(
				"Pull request %s by %s",
				payload.Action,
				payload.PullRequest.User.Login,
			)
		}

		embed := NewEmbed().
			SetTitle(
				"#%d %s",
				payload.PullRequest.Number,
				payload.PullRequest.Title,
			).
			SetURL(
				payload.PullRequest.HTMLURL,
			).
			SetFooter(
				payload.PullRequest.User.Login,
				fmt.Sprintf(
					"https://github.com/%s.png?size=40",
					payload.PullRequest.User.Login,
				),
			).
			SetColor(color)

		switch payload.Action {
		case "opened", "edited":
			embed = embed.SetDescription(
				mdConvert(payload.PullRequest.Body),
			)
			fallthrough
		case "synchronize", "merged":
			if commits, err := fetchCommits(
				payload.PullRequest.CommitsURL,
			); err != nil {
				logger.Error("error fetching commits", zap.Error(err))
			} else {
				body := ""
				for _, commit := range commits {
					body += fmt.Sprintf(
						"• %s [[%s](%s)]\n",
						commit.Commit.Message,
						commit.Sha[:8],
						commit.HTMLURL,
					)
				}
				embed = embed.AddField(
					"Commits",
					body,
				)
			}
		}

		embed = embed.Truncate()

		if _, err := ds.ChannelMessageSendComplex(
			channelID,
			&discordgo.MessageSend{
				Content: header,
				Embed:   embed.MessageEmbed,
			},
		); err != nil {
			return httpError(500, err)
		}

	case github.PushEvent:
		payload := github.PushPayload{}
		if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
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
	if strings.Contains(body, "<li>") {
		converter := md.NewConverter("", true, nil)
		markdown, err := converter.ConvertString(body)
		if err == nil {
			body = markdown
		}
	}
	return body
}

func fetchCommits(url string) ([]GithubCommit, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	cs := []GithubCommit{}
	if err := json.Unmarshal(body, &cs); err != nil {
		return nil, err
	}

	return cs, nil
}

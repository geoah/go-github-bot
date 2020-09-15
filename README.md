# Github Bot

This is a very simple Github bot that posts Pull Request and Issues to a Discord
channel.

## Log

```sh
faas template pull https://github.com/openfaas-incubator/golang-http-template
faas-cli new --lang golang-http --prefix geoah github-bot
cd github-bot
go mod init handler/function
cd ..
```

## Literature

* OpenFAAS Go template, https://github.com/openfaas-incubator/golang-http-template
* OpenFAAS YAML, https://docs.openfaas.com/reference/yaml/
* DiscordGo package, https://github.com/bwmarrin/discordgo
* Discord embed visualizer, https://leovoel.github.io/embed-visualizer/

## Setup

Assuming you have an open instance setup and configured:

* Create a Discord app, https://discord.com/developers/applications
* Create a Discord bot, and get its token
* Invite the bot to your server, https://discordapi.com/permissions.html
* Modify the `gateway` and `image` in the `stack.yaml`
* Note: Remember to set your `OPENFAAS_URL` env var.
* Run `faas-cli up` to deploy the bot
* Go to your Github repo settings, and add a new web hook that points to your function.
  You need to add the following query params to your function URL:
  * `discordBotToken` your bot token
  * `discordChannelID` your discord channel id
  ie. `https://foo/functions/github-bot?discordBotToken=foo&discordChannelID=bar`

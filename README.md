# Github Bot

## Log

```sh
faas template pull https://github.com/openfaas-incubator/golang-http-template
faas-cli new --lang golang-http --prefix geoah github-bot
cd github-bot
go mod init handler/function
cd ..
```

## Setup

* Create app and bot
* Invite to server, https://github.com/v0idp/Mellow#invite-bot

## Literature

* OpenFAAS Go template, https://github.com/openfaas-incubator/golang-http-template
* OpenFAAS YAML, https://docs.openfaas.com/reference/yaml/
* DiscordGo package, https://github.com/bwmarrin/discordgo

## Deployment

```sh
faas-cli up -f github-bot.yml
```

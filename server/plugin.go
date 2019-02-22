package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

const (
	pandaIconURL = "https://cdn.clipart.email/046e3b0eb751d06352bf62b52b808014_fist-bump-vinyl-decal-t34_570-460.jpeg"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	PandaBotID string

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

func (p *Plugin) OnActivate() error {
	user, err := p.API.GetUserByUsername(p.configuration.Username)
	if err != nil && err.StatusCode == http.StatusNotFound {
		p.API.LogInfo(err.Error())
		pandaUser := &model.User{
			Username: p.configuration.Username,
			Password: p.configuration.Password,
			Email:    p.configuration.Email,
		}
		pandaUserCreated, errUser := p.API.CreateUser(pandaUser)
		if errUser != nil {
			return fmt.Errorf("Error creating user panda: %v", errUser.Error())
		}
		p.PandaBotID = pandaUserCreated.Id
		return nil
	} else if err != nil {
		return fmt.Errorf("Error finding user panda: %v", err.Error())
	}

	p.PandaBotID = user.Id
	return nil
}

func (p *Plugin) MessageHasBeenPosted(c *plugin.Context, post *model.Post) {

	teamsToValidate := strings.Split(p.configuration.Teams, ",")
	foundTeam := false
	for _, teamToCheck := range teamsToValidate {
		team, err := p.API.GetTeamByName(teamToCheck)
		if err != nil {
			return
		}
		if teamToCheck == team.Name {
			foundTeam = true
			break
		}
	}
	if !foundTeam {
		return
	}

	channel, err := p.API.GetChannel(post.ChannelId)
	if err != nil {
		return
	}
	if channel.IsGroupOrDirect() {
		return
	}
	channelsToValidate := strings.Split(p.configuration.Channels, ",")
	foundChannel := false
	for _, channelToCheck := range channelsToValidate {
		if channelToCheck == channel.Name {
			foundChannel = true
			break
		}
	}

	if !foundChannel {
		return
	}

	isCompleteMatch, _ := p.checkSubstrings(post.Message, "anytime", "anything")
	if isCompleteMatch {
		rootId := post.RootId
		if rootId == "" {
			rootId = post.Id
		}
		fistPost := &model.Post{
			RootId:    rootId,
			ChannelId: post.ChannelId,
			UserId:    p.PandaBotID,
			Message:   ":fist_oncoming:",
			Props: map[string]interface{}{
				"from_webhook":      "true",
				"override_icon_url": pandaIconURL,
			},
		}
		_, err := p.API.CreatePost(fistPost)
		if err != nil {
			p.API.LogError("Panda Plugin", "err=", err.Error())
			return
		}
		return
	}

	if strings.Contains(post.Message, "anytime") || strings.Contains(post.Message, "anything") {
		fistReaction := &model.Reaction{
			UserId:    p.PandaBotID,
			PostId:    post.Id,
			EmojiName: "fist_oncoming",
		}
		_, err := p.API.AddReaction(fistReaction)
		if err != nil {
			p.API.LogError("Panda Plugin", "err=", err.Error())
			return
		}
		return
	}
}

func (p *Plugin) checkSubstrings(str string, subs ...string) (bool, int) {

	matches := 0
	isCompleteMatch := true

	p.API.LogDebug("Panda Plugin", str, subs)

	for _, sub := range subs {
		if strings.Contains(str, sub) {
			matches++
		} else {
			isCompleteMatch = false
		}
	}

	return isCompleteMatch, matches
}

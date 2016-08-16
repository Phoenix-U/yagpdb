package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/fzzy/radix/redis"
	"github.com/jonas747/dutil/commandsystem"
	"github.com/jonas747/yagpdb/bot"
	"github.com/jonas747/yagpdb/common"
	"github.com/jonas747/yagpdb/web"
)

type Plugin struct{}

func RegisterPlugin() {
	plugin := &Plugin{}
	web.RegisterPlugin(plugin)
	bot.RegisterPlugin(plugin)
}

func (p *Plugin) Name() string {
	return "Commands"
}

type ChannelCommandSetting struct {
	Cmd            string `json:"cmd"`
	CommandEnabled bool   `json:"enabled"`
	AutoDelete     bool   `json:"autodelete"`
}

type ChannelOverride struct {
	Settings        []*ChannelCommandSetting `json:"settings"`
	OverrideEnabled bool                     `json:"enabled"`
	Channel         string                   `json:"channel"`
	ChannelName     string                   `json:"-"` // Used for the template rendering
}

type CommandsConfig struct {
	Prefix string `json:"-"` // Stored in a seperate key for speed

	Global           []*ChannelCommandSetting `json:"gloabl"`
	ChannelOverrides []*ChannelOverride       `json:"overrides"`
}

// Fills in the defaults for missing data, for when users create channels or commands are added
func CheckChannelsConfig(conf *CommandsConfig, channels []*discordgo.Channel) {
	commands := bot.CommandSystem.Commands

	if conf.Global == nil {
		conf.Global = []*ChannelCommandSetting{}
	}

	if conf.ChannelOverrides == nil {
		conf.ChannelOverrides = []*ChannelOverride{}
	}

ROOT:
	for _, channel := range channels {
		// Look for an existing override
		for _, override := range conf.ChannelOverrides {
			// Found an existing override, check if it has all the commands
			if channel.ID == override.Channel {
				override.Settings = checkCommandSettings(override.Settings, commands, false)
				override.ChannelName = channel.Name // Update name if changed
				continue ROOT
			}
		}

		// Not found, create a default override
		override := &ChannelOverride{
			Settings:        []*ChannelCommandSetting{},
			OverrideEnabled: false,
			Channel:         channel.ID,
			ChannelName:     channel.Name,
		}

		// Fill in default command settings
		override.Settings = checkCommandSettings(override.Settings, commands, false)
		conf.ChannelOverrides = append(conf.ChannelOverrides, override)
	}

	// Check the global settings
	conf.Global = checkCommandSettings(conf.Global, commands, true)
}

// Checks a single list of ChannelCommandSettings and applies defaults if not found
func checkCommandSettings(settings []*ChannelCommandSetting, commands []commandsystem.CommandHandler, defaultEnabled bool) []*ChannelCommandSetting {

ROOT:
	for _, cmdDef := range commands {
		cast, ok := cmdDef.(*bot.CustomCommand)
		if !ok {
			continue
		}

		for _, settingsCmd := range settings {
			if cast.Name == settingsCmd.Cmd {
				// Bingo
				continue ROOT
			}
		}

		// Not found, add it to the list of overrides
		settingsCmd := &ChannelCommandSetting{
			Cmd:            cast.Name,
			CommandEnabled: defaultEnabled,
			AutoDelete:     false,
		}
		settings = append(settings, settingsCmd)
	}
	return settings
}

func GetConfig(client *redis.Client, guild string, channels []*discordgo.Channel) *CommandsConfig {
	var config *CommandsConfig
	err := common.GetRedisJson(client, "commands:"+guild, &config)
	if err != nil {
		config = &CommandsConfig{}
	}

	prefix := client.Cmd("GET", "command_prefix:")

	// Fill in defaults
	CheckChannelsConfig(config, channels)

	return config
}

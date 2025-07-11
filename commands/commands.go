package commands

//go:generate sqlboiler --no-hooks psql
//REMOVED: generate easyjson commands.go

import (
	"context"

	"github.com/ThatBathroom/yagpdb/bot/eventsystem"
	"github.com/ThatBathroom/yagpdb/commands/models"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/common/config"
	"github.com/ThatBathroom/yagpdb/common/featureflags"
	prfx "github.com/ThatBathroom/yagpdb/common/prefix"
	"github.com/ThatBathroom/yagpdb/lib/dcmd"
	"github.com/ThatBathroom/yagpdb/lib/discordgo"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

var logger = common.GetPluginLogger(&Plugin{})

type CtxKey int

const (
	CtxKeyCmdSettings CtxKey = iota
	CtxKeyChannelOverride
	CtxKeyExecutedByCC
	CtxKeyExecutedByCommandTemplate
	CtxKeyExecutedByNestedCommandTemplate
)

type MessageFilterFunc func(evt *eventsystem.EventData, msg *discordgo.Message) bool

var (
	confSetTyping = config.RegisterOption("yagpdb.commands.typing", "Wether to set typing or not when running commands", true)
)

// These functions are called on every message, and should return true if the message should be checked for commands, false otherwise
var MessageFilterFuncs []MessageFilterFunc

type Plugin struct{}

func (p *Plugin) PluginInfo() *common.PluginInfo {
	return &common.PluginInfo{
		Name:     "Commands",
		SysName:  "commands",
		Category: common.PluginCategoryCore,
	}
}

func RegisterPlugin() {
	plugin := &Plugin{}
	common.RegisterPlugin(plugin)

	common.InitSchemas("commands", DBSchemas...)
}

type CommandProvider interface {
	// This is where you should register your commands, called on both the webserver and the bot
	AddCommands()
}

func InitCommands() {
	// Setup the command system
	CommandSystem = &dcmd.System{
		Root: &dcmd.Container{
			HelpTitleEmoji: "ℹ️",
			HelpColor:      0xbeff7a,
			RunInDM:        true,
			IgnoreBots:     true,
		},

		ResponseSender: &dcmd.StdResponseSender{LogErrors: true},
		Prefix:         &Plugin{},
	}

	// We have our own middleware before the argument parsing, this is to check for things such as whether or not the command is enabled at all
	CommandSystem.Root.AddMidlewares(YAGCommandMiddleware)
	CommandSystem.Root.AddCommand(cmdHelp, cmdHelp.GetTrigger())
	CommandSystem.Root.AddCommand(cmdPrefix, cmdPrefix.GetTrigger())

	for _, v := range common.Plugins {
		if adder, ok := v.(CommandProvider); ok {
			adder.AddCommands()
		}
	}
}

var _ featureflags.PluginWithFeatureFlags = (*Plugin)(nil)

const (
	featureFlagHasCustomPrefix    = "commands_has_custom_prefix"
	featureFlagHasCustomOverrides = "commands_has_custom_overrides"
)

func (p *Plugin) UpdateFeatureFlags(guildID int64) ([]string, error) {

	prefix, err := prfx.GetCommandPrefixRedis(guildID)
	if err != nil {
		return nil, err
	}

	var flags []string
	if prfx.DefaultCommandPrefix() != prefix {
		flags = append(flags, featureFlagHasCustomPrefix)
	}

	channelOverrides, err := models.CommandsChannelsOverrides(qm.Where("guild_id=?", guildID), qm.Load("CommandsCommandOverrides")).AllG(context.Background())
	if err != nil {
		return nil, err
	}

	if isCustomOverrides(channelOverrides) {
		flags = append(flags, featureFlagHasCustomOverrides)
	}

	return flags, nil
}

func isCustomOverrides(overrides []*models.CommandsChannelsOverride) bool {
	if len(overrides) == 0 {
		return false
	}

	if len(overrides) == 1 && overrides[0].Global {
		// check if this is default
		g := overrides[0]
		if !g.AutodeleteResponse &&
			!g.AutodeleteTrigger &&
			g.CommandsEnabled &&
			len(g.RequireRoles) == 0 &&
			len(g.IgnoreRoles) == 0 &&
			len(g.R.CommandsCommandOverrides) == 0 {
			return false
		}
	}

	return true
}

func (p *Plugin) AllFeatureFlags() []string {
	return []string{
		featureFlagHasCustomPrefix,    // Set if the server has a custom command prefix
		featureFlagHasCustomOverrides, // set if the server has custom command and/or channel overrides
	}
}

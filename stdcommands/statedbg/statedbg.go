package statedbg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/ThatBathroom/yagpdb/bot"
	"github.com/ThatBathroom/yagpdb/commands"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/lib/dcmd"
	"github.com/ThatBathroom/yagpdb/lib/discordgo"
	"github.com/ThatBathroom/yagpdb/stdcommands/util"
)

func Commands() *dcmd.Container {
	container, _ := commands.CommandSystem.Root.Sub("state")
	container.Description = "utilities for debugging state stuff. Bot Admin Only"
	container.AddMidlewares(util.RequireBotAdmin)
	container.AddCommand(getGuild, getGuild.GetTrigger())
	container.AddCommand(getMember, getMember.GetTrigger())
	container.AddCommand(botMember, botMember.GetTrigger())

	return container
}

var getGuild = &commands.YAGCommand{
	CmdCategory:  commands.CategoryDebug,
	Name:         "guild",
	Description:  "Responds with state debug info. Bot Owner Only",
	HideFromHelp: true,
	RunFunc:      util.RequireOwner(cmdFuncGetGuild),
}

func cmdFuncGetGuild(data *dcmd.Data) (interface{}, error) {
	serialized, err := json.MarshalIndent(data.GuildData.GS, "", "  ")
	if err != nil {
		return nil, err
	}

	send := &discordgo.MessageSend{
		File: &discordgo.File{
			Name:        fmt.Sprintf("guild_%d.json", data.GuildData.GS.ID),
			ContentType: "application/json",
			Reader:      bytes.NewReader(serialized),
		},
	}

	return send, nil
}

var getMember = &commands.YAGCommand{
	CmdCategory: commands.CategoryDebug,
	Name:        "member",
	Description: "Responds with state debug info. Bot Owner Only",
	Arguments: []*dcmd.ArgDef{
		{Name: "Target", Type: dcmd.BigInt},
	},
	ArgSwitches: []*dcmd.ArgDef{
		{Name: "fetch", Help: "fetch the member if not in state"},
	},
	RequiredArgs: 1,
	HideFromHelp: true,
	RunFunc:     util.RequireOwner(cmdFuncGetMember),
}

func cmdFuncGetMember(data *dcmd.Data) (interface{}, error) {

	targetID := data.Args[0].Int64()

	ms := bot.State.GetMember(data.GuildData.GS.ID, targetID)
	didFetch := false
	if ms == nil && data.Switch("fetch").Bool() {
		didFetch = true
		fms, err := bot.GetMember(data.GuildData.GS.ID, targetID)
		if err != nil {
			return nil, err
		}

		ms = fms
	} else if ms == nil {
		return "Member not in state :(", nil
	}

	serialized, err := json.MarshalIndent(ms, "", "  ")
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("Fetched: %v, ```json\n%s\n```", didFetch, string(serialized)), nil
}

var botMember = &commands.YAGCommand{
	CmdCategory:  commands.CategoryDebug,
	Name:         "botmember",
	Description:  "Responds with state debug info. Bot Owner Only",
	HideFromHelp: true,
	RunFunc:      util.RequireOwner(cmdFuncBotMember),
}

func cmdFuncBotMember(data *dcmd.Data) (interface{}, error) {
	shards := bot.ReadyTracker.GetProcessShards()

	numFound := 0
	numNotFound := 0
	for _, v := range shards {
		guilds := bot.State.GetShardGuilds(int64(v))
		for _, g := range guilds {
			if ms := bot.State.GetMember(g.ID, common.BotUser.ID); ms != nil {
				numFound++
			} else {
				numNotFound++
			}
		}
	}

	return fmt.Sprintf("Bot member found on %d/%d guilds", numFound, numFound+numNotFound), nil
}

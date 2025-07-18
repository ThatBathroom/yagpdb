package banserver

import (
	"github.com/ThatBathroom/yagpdb/commands"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/lib/dcmd"
	"github.com/ThatBathroom/yagpdb/stdcommands/util"
	"github.com/mediocregopher/radix/v3"
)

var Command = &commands.YAGCommand{
	Cooldown:             2,
	CmdCategory:          commands.CategoryDebug,
	HideFromCommandsPage: true,
	Name:                 "banserver",
	Description:          "Bans the specified server from using the bot. YAGPDB will leave the server, and leave whenever invited back. Bot Owner Only",
	HideFromHelp:         true,
	RequiredArgs:         1,
	Arguments: []*dcmd.ArgDef{
		{Name: "server", Type: dcmd.BigInt},
	},
	RunFunc: util.RequireOwner(func(data *dcmd.Data) (interface{}, error) {
		err := common.BotSession.GuildLeave(data.Args[0].Int64())
		if err == nil {

			common.RedisPool.Do(radix.FlatCmd(nil, "SADD", "banned_servers", data.Args[0].Int64()))

			return "Banned " + data.Args[0].Str(), nil
		}
		return nil, err
	}),
}

package guildunavailable

import (
	"fmt"

	"github.com/ThatBathroom/yagpdb/bot/botrest"
	"github.com/ThatBathroom/yagpdb/commands"
	"github.com/ThatBathroom/yagpdb/lib/dcmd"
)

var Command = &commands.YAGCommand{
	CmdCategory:  commands.CategoryDebug,
	Name:         "IsGuildUnavailable",
	Description:  "Returns whether the specified guild is unavailable or not",
	RequiredArgs: 1,
	Arguments: []*dcmd.ArgDef{
		{Name: "guildid", Type: dcmd.BigInt, Default: int64(0)},
	},
	RunFunc: func(data *dcmd.Data) (interface{}, error) {
		gID := data.Args[0].Int64()
		guild, err := botrest.GetGuild(gID)
		if err != nil {
			return "Uh oh", err
		}

		return fmt.Sprintf("Guild (%d) unavailable: %v", guild.ID, !guild.Available), nil
	},
}

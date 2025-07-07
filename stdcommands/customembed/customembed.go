package customembed

import (
	"github.com/ThatBathroom/yagpdb/commands"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/lib/dcmd"
	"github.com/ThatBathroom/yagpdb/lib/discordgo"
	"github.com/goccy/go-yaml"
)

var Command = &commands.YAGCommand{
	CmdCategory:         commands.CategoryTool,
	Name:                "CustomEmbed",
	Aliases:             []string{"ce"},
	Description:         "Creates an embed from what you give it in json form: https://help.yagpdb.xyz/docs/reference/custom-embeds/",
	LongDescription:     "Example: `-ce {\"title\": \"hello\", \"description\": \"wew\"}`",
	RequiredArgs:        1,
	RequireDiscordPerms: []int64{discordgo.PermissionManageMessages},
	Arguments: []*dcmd.ArgDef{
		{Name: "Json", Type: dcmd.String},
	},
	RunFunc: func(data *dcmd.Data) (interface{}, error) {
		j := common.ParseCodeblock(data.Args[0].Str())
		var parsed discordgo.MessageEmbed

		// yaml.Unmarshal also works with JSON, as that is a subset of YAML.
		err := yaml.Unmarshal([]byte(j), &parsed)
		if err != nil {
			return err, err
		}

		if discordgo.IsEmbedEmpty(&parsed) {
			return "Cannot send an empty embed", nil
		}

		return &parsed, nil
	},
}

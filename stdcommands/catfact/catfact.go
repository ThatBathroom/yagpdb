package catfact

import (
	"math/rand"

	"github.com/ThatBathroom/yagpdb/commands"
	"github.com/ThatBathroom/yagpdb/lib/dcmd"
)

var Command = &commands.YAGCommand{
	CmdCategory:         commands.CategoryFun,
	Name:                "CatFact",
	Aliases:             []string{"cf", "cat", "catfacts"},
	Description:         "Cat Facts",
	DefaultEnabled:      true,
	SlashCommandEnabled: true,

	RunFunc: func(data *dcmd.Data) (interface{}, error) {
		cf := Catfacts[rand.Intn(len(Catfacts))]
		return cf, nil
	},
}

package stdcommands

import (
	"github.com/ThatBathroom/yagpdb/v2/bot"
	"github.com/ThatBathroom/yagpdb/v2/bot/eventsystem"
	"github.com/ThatBathroom/yagpdb/v2/commands"
	"github.com/ThatBathroom/yagpdb/v2/common"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/advice"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/allocstat"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/banserver"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/calc"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/catfact"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/ccreqs"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/cleardm"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/createinvite"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/currentshard"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/currenttime"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/customembed"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/dadjoke"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/dcallvoice"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/define"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/dictionary"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/dogfact"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/eightball"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/findserver"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/forex"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/globalrl"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/guildunavailable"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/howlongtobeat"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/info"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/inspire"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/invite"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/leaveserver"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/listflags"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/listroles"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/memstats"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/ping"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/poll"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/roast"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/roll"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/setstatus"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/simpleembed"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/sleep"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/statedbg"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/stateinfo"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/throw"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/toggledbg"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/topcommands"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/topevents"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/topgames"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/topic"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/topservers"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/unbanserver"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/undelete"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/viewperms"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/weather"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/wouldyourather"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/xkcd"
	"github.com/ThatBathroom/yagpdb/v2/stdcommands/yagstatus"
)

var (
	_ bot.BotInitHandler       = (*Plugin)(nil)
	_ commands.CommandProvider = (*Plugin)(nil)
)

type Plugin struct{}

func (p *Plugin) PluginInfo() *common.PluginInfo {
	return &common.PluginInfo{
		Name:     "Standard Commands",
		SysName:  "standard_commands",
		Category: common.PluginCategoryCore,
	}
}

func (p *Plugin) AddCommands() {
	commands.AddRootCommands(p,
		// Info
		info.Command,
		invite.Command,

		// Standard
		define.Command,
		weather.Command,
		calc.Command,
		topic.Command,
		catfact.Command,
		dadjoke.Command,
		dogfact.Command,
		advice.Command,
		ping.Command,
		throw.Command,
		roll.Command,
		customembed.Command,
		simpleembed.Command,
		currenttime.Command,
		listroles.Command,
		memstats.Command,
		wouldyourather.Command,
		poll.Command,
		undelete.Command,
		viewperms.Command,
		topgames.Command,
		xkcd.Command,
		howlongtobeat.Command,
		inspire.Command,
		forex.Command,
		roast.Command,
		eightball.Command,

		// Maintenance
		stateinfo.Command,
		leaveserver.Command,
		banserver.Command,
		cleardm.Command,
		allocstat.Command,
		unbanserver.Command,
		topservers.Command,
		topcommands.Command,
		topevents.Command,
		currentshard.Command,
		guildunavailable.Command,
		yagstatus.Command,
		setstatus.Command,
		createinvite.Command,
		findserver.Command,
		dcallvoice.Command,
		ccreqs.Command,
		sleep.Command,
		toggledbg.Command,
		globalrl.Command,
		listflags.Command,
	)

	statedbg.Commands()
	commands.AddRootCommands(p, dictionary.Command)
}

func (p *Plugin) BotInit() {
	eventsystem.AddHandlerAsyncLastLegacy(p, ping.HandleMessageCreate, eventsystem.EventMessageCreate)
}

func RegisterPlugin() {
	common.RegisterPlugin(&Plugin{})
}

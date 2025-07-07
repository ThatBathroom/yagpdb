package stdcommands

import (
	"github.com/ThatBathroom/yagpdb/bot"
	"github.com/ThatBathroom/yagpdb/bot/eventsystem"
	"github.com/ThatBathroom/yagpdb/commands"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/stdcommands/advice"
	"github.com/ThatBathroom/yagpdb/stdcommands/allocstat"
	"github.com/ThatBathroom/yagpdb/stdcommands/banserver"
	"github.com/ThatBathroom/yagpdb/stdcommands/calc"
	"github.com/ThatBathroom/yagpdb/stdcommands/catfact"
	"github.com/ThatBathroom/yagpdb/stdcommands/ccreqs"
	"github.com/ThatBathroom/yagpdb/stdcommands/cleardm"
	"github.com/ThatBathroom/yagpdb/stdcommands/createinvite"
	"github.com/ThatBathroom/yagpdb/stdcommands/currentshard"
	"github.com/ThatBathroom/yagpdb/stdcommands/currenttime"
	"github.com/ThatBathroom/yagpdb/stdcommands/customembed"
	"github.com/ThatBathroom/yagpdb/stdcommands/dadjoke"
	"github.com/ThatBathroom/yagpdb/stdcommands/dcallvoice"
	"github.com/ThatBathroom/yagpdb/stdcommands/define"
	"github.com/ThatBathroom/yagpdb/stdcommands/dictionary"
	"github.com/ThatBathroom/yagpdb/stdcommands/dogfact"
	"github.com/ThatBathroom/yagpdb/stdcommands/eightball"
	"github.com/ThatBathroom/yagpdb/stdcommands/findserver"
	"github.com/ThatBathroom/yagpdb/stdcommands/forex"
	"github.com/ThatBathroom/yagpdb/stdcommands/globalrl"
	"github.com/ThatBathroom/yagpdb/stdcommands/guildunavailable"
	"github.com/ThatBathroom/yagpdb/stdcommands/howlongtobeat"
	"github.com/ThatBathroom/yagpdb/stdcommands/info"
	"github.com/ThatBathroom/yagpdb/stdcommands/inspire"
	"github.com/ThatBathroom/yagpdb/stdcommands/invite"
	"github.com/ThatBathroom/yagpdb/stdcommands/leaveserver"
	"github.com/ThatBathroom/yagpdb/stdcommands/listflags"
	"github.com/ThatBathroom/yagpdb/stdcommands/listroles"
	"github.com/ThatBathroom/yagpdb/stdcommands/memstats"
	"github.com/ThatBathroom/yagpdb/stdcommands/ping"
	"github.com/ThatBathroom/yagpdb/stdcommands/poll"
	"github.com/ThatBathroom/yagpdb/stdcommands/roast"
	"github.com/ThatBathroom/yagpdb/stdcommands/roll"
	"github.com/ThatBathroom/yagpdb/stdcommands/setstatus"
	"github.com/ThatBathroom/yagpdb/stdcommands/simpleembed"
	"github.com/ThatBathroom/yagpdb/stdcommands/sleep"
	"github.com/ThatBathroom/yagpdb/stdcommands/statedbg"
	"github.com/ThatBathroom/yagpdb/stdcommands/stateinfo"
	"github.com/ThatBathroom/yagpdb/stdcommands/throw"
	"github.com/ThatBathroom/yagpdb/stdcommands/toggledbg"
	"github.com/ThatBathroom/yagpdb/stdcommands/topcommands"
	"github.com/ThatBathroom/yagpdb/stdcommands/topevents"
	"github.com/ThatBathroom/yagpdb/stdcommands/topgames"
	"github.com/ThatBathroom/yagpdb/stdcommands/topic"
	"github.com/ThatBathroom/yagpdb/stdcommands/topservers"
	"github.com/ThatBathroom/yagpdb/stdcommands/unbanserver"
	"github.com/ThatBathroom/yagpdb/stdcommands/undelete"
	"github.com/ThatBathroom/yagpdb/stdcommands/viewperms"
	"github.com/ThatBathroom/yagpdb/stdcommands/weather"
	"github.com/ThatBathroom/yagpdb/stdcommands/wouldyourather"
	"github.com/ThatBathroom/yagpdb/stdcommands/xkcd"
	"github.com/ThatBathroom/yagpdb/stdcommands/yagstatus"
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

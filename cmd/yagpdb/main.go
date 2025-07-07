package main

import (
	"github.com/ThatBathroom/yagpdb/analytics"
	"github.com/ThatBathroom/yagpdb/antiphishing"
	"github.com/ThatBathroom/yagpdb/common/featureflags"
	"github.com/ThatBathroom/yagpdb/common/prom"
	"github.com/ThatBathroom/yagpdb/common/run"
	"github.com/ThatBathroom/yagpdb/lib/confusables"
	"github.com/ThatBathroom/yagpdb/trivia"
	"github.com/ThatBathroom/yagpdb/web/discorddata"

	// Core yagpdb packages

	"github.com/ThatBathroom/yagpdb/admin"
	"github.com/ThatBathroom/yagpdb/bot/paginatedmessages"
	"github.com/ThatBathroom/yagpdb/common/internalapi"
	"github.com/ThatBathroom/yagpdb/common/scheduledevents2"

	// Plugin imports
	"github.com/ThatBathroom/yagpdb/automod"
	"github.com/ThatBathroom/yagpdb/automod_legacy"
	"github.com/ThatBathroom/yagpdb/autorole"
	"github.com/ThatBathroom/yagpdb/cah"
	"github.com/ThatBathroom/yagpdb/commands"
	"github.com/ThatBathroom/yagpdb/customcommands"
	"github.com/ThatBathroom/yagpdb/discordlogger"
	"github.com/ThatBathroom/yagpdb/logs"
	"github.com/ThatBathroom/yagpdb/moderation"
	"github.com/ThatBathroom/yagpdb/notifications"
	"github.com/ThatBathroom/yagpdb/premium"
	"github.com/ThatBathroom/yagpdb/premium/discordpremiumsource"
	"github.com/ThatBathroom/yagpdb/premium/patreonpremiumsource"
	"github.com/ThatBathroom/yagpdb/reddit"
	"github.com/ThatBathroom/yagpdb/reminders"
	"github.com/ThatBathroom/yagpdb/reputation"
	"github.com/ThatBathroom/yagpdb/rolecommands"
	"github.com/ThatBathroom/yagpdb/rsvp"
	"github.com/ThatBathroom/yagpdb/safebrowsing"
	"github.com/ThatBathroom/yagpdb/serverstats"
	"github.com/ThatBathroom/yagpdb/soundboard"
	"github.com/ThatBathroom/yagpdb/stdcommands"
	"github.com/ThatBathroom/yagpdb/streaming"
	"github.com/ThatBathroom/yagpdb/tickets"
	"github.com/ThatBathroom/yagpdb/timezonecompanion"
	"github.com/ThatBathroom/yagpdb/twitter"
	"github.com/ThatBathroom/yagpdb/verification"
	"github.com/ThatBathroom/yagpdb/youtube"
	// External plugins
)

func main() {

	run.Init()

	//BotSession.LogLevel = discordgo.LogInformational
	paginatedmessages.RegisterPlugin()
	discorddata.RegisterPlugin()

	// Setup plugins
	analytics.RegisterPlugin()
	safebrowsing.RegisterPlugin()
	antiphishing.RegisterPlugin()
	discordlogger.Register()
	commands.RegisterPlugin()
	stdcommands.RegisterPlugin()
	serverstats.RegisterPlugin()
	notifications.RegisterPlugin()
	customcommands.RegisterPlugin()
	reddit.RegisterPlugin()
	moderation.RegisterPlugin()
	reputation.RegisterPlugin()
	streaming.RegisterPlugin()
	automod_legacy.RegisterPlugin()
	automod.RegisterPlugin()
	logs.RegisterPlugin()
	autorole.RegisterPlugin()
	reminders.RegisterPlugin()
	soundboard.RegisterPlugin()
	youtube.RegisterPlugin()
	rolecommands.RegisterPlugin()
	cah.RegisterPlugin()
	tickets.RegisterPlugin()
	verification.RegisterPlugin()
	premium.RegisterPlugin()
	patreonpremiumsource.RegisterPlugin()
	discordpremiumsource.RegisterPlugin()
	scheduledevents2.RegisterPlugin()
	twitter.RegisterPlugin()
	rsvp.RegisterPlugin()
	timezonecompanion.RegisterPlugin()
	admin.RegisterPlugin()
	internalapi.RegisterPlugin()
	prom.RegisterPlugin()
	featureflags.RegisterPlugin()
	trivia.RegisterPlugin()

	// Register confusables replacer
	confusables.Init()

	run.Run()
}

package paginatedmessages

import (
	"errors"
	"sync"
	"time"

	"github.com/ThatBathroom/yagpdb/bot"
	"github.com/ThatBathroom/yagpdb/bot/eventsystem"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/common/pubsub"
	"github.com/ThatBathroom/yagpdb/lib/discordgo"
)

var (
	logger                  = common.GetPluginLogger(&Plugin{})
	activePaginatedMessages []*PaginatedMessage
	menusLock               sync.Mutex
)

var ErrNoResults = errors.New("no results")

type Plugin struct{}

func (p *Plugin) PluginInfo() *common.PluginInfo {
	return &common.PluginInfo{
		Name:     "Paginated Messages",
		SysName:  "paginatedmessages",
		Category: common.PluginCategoryMisc,
	}
}

func RegisterPlugin() {
	common.RegisterPlugin(&Plugin{})
}

var _ bot.BotInitHandler = (*Plugin)(nil)

func (p *Plugin) BotInit() {
	eventsystem.AddHandlerAsyncLastLegacy(p, handleInteractionCreate, eventsystem.EventInteractionCreate)

	// this just handles interaction events from DMS
	pubsub.AddHandler("dm_interaction", func(evt *pubsub.Event) {
		dataCast := evt.Data.(*discordgo.InteractionCreate)
		if dataCast.Type != discordgo.InteractionMessageComponent {
			return
		}
		switch dataCast.MessageComponentData().CustomID {
		case paginationNext:
			handlePageChange(dataCast, 1)
		case paginationPrev:
			handlePageChange(dataCast, -1)
		}
	}, discordgo.InteractionCreate{})
}

type PaginatedMessage struct {
	// immutable fields, safe to access without a lock, don't write to these, i dont see why you would need to either...
	MessageID int64
	ChannelID int64
	GuildID   int64

	// mutable fields
	CurrentPage  int
	MaxPage      int
	LastResponse *discordgo.MessageEmbed
	Navigate     func(p *PaginatedMessage, newPage int) (*discordgo.MessageEmbed, error)
	Broken       bool

	stopped        bool
	stopCh         chan bool
	lastUpdateTime time.Time
	mu             sync.Mutex
}

type PagerFunc func(p *PaginatedMessage, page int) (*discordgo.MessageEmbed, error)

func (p *PaginatedMessage) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}

	p.stopped = true
	close(p.stopCh)
}

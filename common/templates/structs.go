package templates

import (
	"errors"

	"github.com/ThatBathroom/yagpdb/lib/discordgo"
	"github.com/ThatBathroom/yagpdb/lib/dstate"
)

// CtxChannel is almost a 1:1 copy of dstate.ChannelState, its needed because we cant axpose all those state methods
// we also cant use discordgo.Channel because that would likely break a lot of custom commands at this point.
type CtxChannel struct {
	// These fields never change
	ID        int64
	GuildID   int64
	IsPrivate bool
	IsThread  bool
	IsForum   bool

	Name                 string                           `json:"name"`
	Type                 discordgo.ChannelType            `json:"type"`
	Topic                string                           `json:"topic"`
	NSFW                 bool                             `json:"nsfw"`
	Position             int                              `json:"position"`
	Bitrate              int                              `json:"bitrate"`
	PermissionOverwrites []*discordgo.PermissionOverwrite `json:"permission_overwrites"`
	ParentID             int64                            `json:"parent_id"`
	OwnerID              int64                            `json:"owner_id"`

	AvailableTags  []discordgo.ForumTag      `json:"available_tags"`
	AppliedTags    []int64                   `json:"applied_tags"`
	Flags          discordgo.ChannelFlags    `json:"flags"`
	ThreadMetadata *discordgo.ThreadMetadata `json:"thread_metadata,omitempty"`
}

func (c *CtxChannel) Mention() (string, error) {
	if c == nil {
		return "", errors.New("channel not found")
	}
	return "<#" + discordgo.StrID(c.ID) + ">", nil
}

func CtxChannelFromCS(cs *dstate.ChannelState) *CtxChannel {

	cop := make([]*discordgo.PermissionOverwrite, len(cs.PermissionOverwrites))
	for i := 0; i < len(cs.PermissionOverwrites); i++ {
		cop[i] = &cs.PermissionOverwrites[i]
	}

	ctxChannel := &CtxChannel{
		ID:                   cs.ID,
		IsPrivate:            cs.IsPrivate(),
		IsThread:             cs.Type.IsThread(),
		IsForum:              cs.Type == discordgo.ChannelTypeGuildForum,
		GuildID:              cs.GuildID,
		Name:                 cs.Name,
		Type:                 cs.Type,
		Topic:                cs.Topic,
		NSFW:                 cs.NSFW,
		Position:             cs.Position,
		Bitrate:              cs.Bitrate,
		PermissionOverwrites: cop,
		ParentID:             cs.ParentID,
		OwnerID:              cs.OwnerID,
		AvailableTags:        cs.AvailableTags,
		AppliedTags:          cs.AppliedTags,
		Flags:                cs.Flags,
		ThreadMetadata:       cs.ThreadMetadata,
	}

	return ctxChannel
}

package verification

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"emperror.dev/errors"
	"github.com/ThatBathroom/yagpdb/analytics"
	"github.com/ThatBathroom/yagpdb/bot"
	"github.com/ThatBathroom/yagpdb/bot/eventsystem"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/common/scheduledevents2"
	seventsmodels "github.com/ThatBathroom/yagpdb/common/scheduledevents2/models"
	"github.com/ThatBathroom/yagpdb/common/templates"
	"github.com/ThatBathroom/yagpdb/lib/discordgo"
	"github.com/ThatBathroom/yagpdb/lib/dstate"
	"github.com/ThatBathroom/yagpdb/moderation"
	"github.com/ThatBathroom/yagpdb/verification/models"
	"github.com/ThatBathroom/yagpdb/web"
	"github.com/mediocregopher/radix/v3"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

const InTicketPerms = discordgo.PermissionSendMessages | discordgo.PermissionViewChannel

var _ bot.BotInitHandler = (*Plugin)(nil)

type VerificationEventData struct {
	UserID int64  `json:"user_id"`
	Token  string `json:"token"`
}

func (p *Plugin) BotInit() {
	eventsystem.AddHandlerAsyncLastLegacy(p, p.handleMemberJoin, eventsystem.EventGuildMemberAdd)
	eventsystem.AddHandlerAsyncLastLegacy(p, p.handleMemberUpdate, eventsystem.EventGuildMemberUpdate)
	eventsystem.AddHandlerAsyncLastLegacy(p, p.handleBanAdd, eventsystem.EventGuildBanAdd)
	scheduledevents2.RegisterHandler("verification_user_verified", int64(0), ScheduledEventMW(p.handleUserVerifiedScheduledEvent))
	scheduledevents2.RegisterHandler("verification_user_warn", VerificationEventData{}, ScheduledEventMW(p.handleWarnUserVerification))
	scheduledevents2.RegisterHandler("verification_user_kick", VerificationEventData{}, ScheduledEventMW(p.handleKickUser))

	go gcRecentGuildBansLoop()
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func VerificationPendingMembersKey(gID int64) string {
	return "verification_pending_members:" + strconv.FormatInt(gID, 10)
}

func memberPresentInVerificationPendingSet(guildID int64, userID int64) bool {
	var memberScore int
	err := common.RedisPool.Do(radix.Cmd(&memberScore, "ZSCORE", VerificationPendingMembersKey(guildID), strconv.FormatInt(userID, 10)))
	if err != nil {
		logger.WithError(err).Error("Failed fetching member from the verification pending set")
	}
	return memberScore != 0
}

// Function to check if member is present in verification pending set, and add if not present
func addMemberToVerificationPendingSet(guildID int64, userID int64) {
	if memberPresentInVerificationPendingSet(guildID, userID) {
		// Member is already in the set
		return
	}

	err := common.RedisPool.Do(radix.Cmd(nil, "ZADD", VerificationPendingMembersKey(guildID), "1", strconv.FormatInt(userID, 10)))
	if err != nil {
		logger.WithError(err).Error("Failed adding member to the verification pending set")
	}
}

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func (p *Plugin) handleVerificationAfterScreening(member *discordgo.Member) {
	conf, err := models.FindVerificationConfigG(context.Background(), member.GuildID)
	if err != nil {
		if err != sql.ErrNoRows {
			logger.WithError(err).WithField("guild", member.GuildID).WithField("user", member.User.ID).Error("unable to retrieve config")
		}
		// either no config or an error occured
		return
	}
	if !conf.Enabled {
		return
	}
	gs := bot.State.GetGuild(member.GuildID)
	roleInvalid := true
	for _, role := range gs.Roles {
		if role.ID == conf.VerifiedRole {
			roleInvalid = false
			break
		}
	}
	if roleInvalid {
		cop := *conf
		cop.Enabled = false
		cop.UpdateG(context.Background(), boil.Whitelist("enabled"))
		return
	}

	// Check if member is already verified, if yes then remove any scheduled events
	if common.ContainsInt64Slice(member.Roles, conf.VerifiedRole) {
		err = p.clearScheduledEvents(context.Background(), member.GuildID, member.User.ID)
		if err != nil {
			logger.WithError(err).WithField("guild", member.GuildID).WithField("user", member.User.ID).Error("failed clearing past scheduled warn/kick events")
		}
		return
	}

	go analytics.RecordActiveUnit(member.GuildID, p, "process_started")

	go p.startVerificationProcess(conf, member.GuildID, member.User)
}

func (p *Plugin) handleMemberJoin(evt *eventsystem.EventData) {
	addEvt := evt.GuildMemberAdd()

	if addEvt.User.Bot {
		return
	}
	if addEvt.Pending {
		// Membership screening is pending for this member, add to pending set and return
		addMemberToVerificationPendingSet(addEvt.GuildID, addEvt.User.ID)
		return
	}

	p.handleVerificationAfterScreening(addEvt.Member)
}

func (p *Plugin) handleMemberUpdate(evt *eventsystem.EventData) {
	updateEvt := evt.GuildMemberUpdate()

	if updateEvt.User.Bot {
		return
	}
	if updateEvt.Pending {
		// Membership screening is pending for this member, add to pending set and return
		addMemberToVerificationPendingSet(updateEvt.GuildID, updateEvt.User.ID)
		return
	}

	if memberPresentInVerificationPendingSet(updateEvt.GuildID, updateEvt.User.ID) {
		// Member was found in the verification pending set, remove from the set and assign role to the member
		err := common.RedisPool.Do(radix.Cmd(nil, "ZREM", VerificationPendingMembersKey(updateEvt.GuildID), strconv.FormatInt(updateEvt.User.ID, 10)))
		if err != nil {
			logger.WithError(err).Error("Failed removing member from the verification pending set")
		}
		p.handleVerificationAfterScreening(updateEvt.Member)
	}
}

func (p *Plugin) createVerificationSession(userID, guildID int64) (string, error) {
	for {
		token := RandStringRunes(32)
		model := &models.VerificationSession{
			Token:     token,
			UserID:    userID,
			GuildID:   guildID,
			CreatedAt: time.Now(),
		}

		err := model.InsertG(context.Background(), boil.Infer())
		if err == nil {
			return token, nil
		}

		if common.ErrPQIsUniqueViolation(err) {
			// somehow we made a duplicate token...
			continue
		}

		// otherwise an unknown error occured
		return token, err
	}
}

func (p *Plugin) startVerificationProcess(conf *models.VerificationConfig, guildID int64, target *discordgo.User) {

	token, err := p.createVerificationSession(target.ID, guildID)
	if err != nil {
		logger.WithError(err).WithField("user", target.ID).WithField("guild", guildID).Error("failed creating verification session")
		return
	}

	gs := bot.State.GetGuild(guildID)
	if gs == nil {
		logger.Error("guild not available")
		return
	}

	msg := conf.DMMessage
	if strings.TrimSpace(msg) == "" {
		msg = DefaultDMMessage
	}

	ms, err := bot.GetMember(guildID, target.ID)
	if err != nil {
		logger.WithError(err).Error("failed retrieving member")
		return
	}

	channel, err := common.BotSession.UserChannelCreate(ms.User.ID)
	if err != nil {
		logger.WithError(err).Error("failed creating user channel")
		return
	}

	cs := dstate.ChannelStateFromDgo(channel)

	tmplCTX := templates.NewContext(gs, &cs, ms)
	tmplCTX.Name = "dm_verification_message"
	tmplCTX.Data["Link"] = fmt.Sprintf("%s/public/%d/verify/%d/%s", web.BaseURL(), guildID, target.ID, token)

	err = tmplCTX.ExecuteAndSendWithErrors(msg, channel.ID)
	if err != nil {
		logger.WithError(err).WithField("guild", gs.ID).WithField("user", ms.User.ID).Error("failed sending verification dm message")
	}

	evt := &VerificationEventData{
		UserID: target.ID,
		Token:  token,
	}

	// schedule the kick and warnings
	err = p.clearScheduledEvents(context.Background(), gs.ID, ms.User.ID) //clear old scheduled events
	if err != nil {
		logger.WithError(err).WithField("guild", gs.ID).WithField("user", ms.User.ID).Error("failed clearing past scheduled warn/kick events.")
	}
	if conf.WarnUnverifiedAfter > 0 && conf.WarnMessage != "" {
		scheduledevents2.ScheduleEvent("verification_user_warn", guildID, time.Now().Add(time.Minute*time.Duration(conf.WarnUnverifiedAfter)), evt)
	}
	if conf.KickUnverifiedAfter > 0 {
		scheduledevents2.ScheduleEvent("verification_user_kick", guildID, time.Now().Add(time.Minute*time.Duration(conf.KickUnverifiedAfter)), evt)
	}

	p.logAction(guildID, conf.LogChannel, target, "New user joined waiting to be verified as a human", 0x47aaed)
}

func ScheduledEventMW(innerHandler func(ms *dstate.MemberState, guildID int64, conf *models.VerificationConfig, rawData interface{}) (bool, error)) func(evt *seventsmodels.ScheduledEvent, data interface{}) (retry bool, err error) {
	return func(evt *seventsmodels.ScheduledEvent, data interface{}) (retry bool, err error) {

		userID := int64(0)

		switch t := data.(type) {
		case *int64:
			userID = *t
		case *VerificationEventData:
			userID = t.UserID
		}

		conf, err := models.FindVerificationConfigG(context.Background(), evt.GuildID)
		if err != nil {
			if err != sql.ErrNoRows {
				logger.WithError(err).WithField("guild", evt.GuildID).WithField("user", userID).Error("unable to retrieve config")
				return true, err
			}

			// either no config anymore? shouldn't be possible
			return false, nil
		}

		ms, err := bot.GetMember(evt.GuildID, userID)
		if err != nil {
			return scheduledevents2.CheckDiscordErrRetry(err), errors.WithStackIf(err)
		}

		return innerHandler(ms, evt.GuildID, conf, data)
	}

}

func (p *Plugin) handleUserVerifiedScheduledEvent(ms *dstate.MemberState, guildID int64, conf *models.VerificationConfig, rawData interface{}) (retry bool, err error) {
	err = common.BotSession.GuildMemberRoleAdd(guildID, ms.User.ID, conf.VerifiedRole)
	if err != nil {
		return scheduledevents2.CheckDiscordErrRetry(err), err
	}

	model, err := models.FindVerifiedUserG(context.Background(), guildID, ms.User.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, err
		}
		return scheduledevents2.CheckDiscordErrRetry(err), err
	}

	err = p.clearScheduledEvents(context.Background(), guildID, ms.User.ID)
	if err != nil {
		return true, err
	}

	if !confVerificationTrackIPs.GetBool() || model.IP == "" {
		p.logAction(guildID, conf.LogChannel, &ms.User, "User successfully verified", 0x49ed47)
		return false, nil
	}

	// Check for IP conflicts
	conflicts, err := p.findIPConflicts(guildID, ms.User.ID, model.IP)
	if err != nil {
		return scheduledevents2.CheckDiscordErrRetry(err), err
	}

	if len(conflicts) < 1 {
		p.logAction(guildID, conf.LogChannel, &ms.User, "User successfully verified", 0x49ed47)
		return false, nil
	}

	// check if the user shares a IP with a banned user
	ban, err := p.CheckBanned(guildID, conflicts)
	if err != nil {
		return scheduledevents2.CheckDiscordErrRetry(err), err
	}

	if ban != nil {
		// shares a IP with a banned user
		banReason := "Alt-" + ban.User.Username + ": " + ban.Reason
		if utf8.RuneCountInString(banReason) > 512 {
			// trim ban reason
			r := []rune(banReason)
			r = r[:509]
			banReason = string(r) + "..."
		}

		err := moderation.BanUser(nil, guildID, nil, nil, common.BotUser, banReason, &ms.User, false)
		if err != nil {
			return scheduledevents2.CheckDiscordErrRetry(err), err
		}

		p.logAction(guildID, conf.LogChannel, &ms.User, fmt.Sprintf("User banned for sharing IP with banned user %s (%d)\nReason: %s",
			ban.User.String(), ban.User.ID, ban.Reason), 0xef4640)

		return false, nil
	}

	// Does not share the IP with a banned user, but warn about alt account
	var builder strings.Builder
	builder.WriteString("User verified but verified with the same IP as the following users: \n")

	for i, v := range conflicts {
		builder.WriteString(fmt.Sprintf("\n%s (%d)", v.String(), v.ID))
		if i >= 20 && len(conflicts) > 21 {
			builder.WriteString(fmt.Sprintf("\n\nAnd %d other users...", len(conflicts)-21))
			break
		}
	}

	p.logAction(guildID, conf.LogChannel, &ms.User, builder.String(), 0xff8228)
	return false, nil
}

func (p *Plugin) checkMemberAlreadyVerified(ms *dstate.MemberState, conf *models.VerificationConfig) bool {
	if !common.ContainsInt64Slice(ms.Member.Roles, conf.VerifiedRole) {
		return false
	}

	err := p.clearScheduledEvents(context.Background(), ms.GuildID, ms.User.ID)
	if err != nil {
		logger.WithError(err).WithField("guild", ms.GuildID).WithField("user", ms.User.ID).Error("failed clearing past scheduled warn/kick events")
	}
	return true
}

func (p *Plugin) clearScheduledEvents(ctx context.Context, guildID, userID int64) error {
	_, err := seventsmodels.ScheduledEvents(
		qm.Where("(event_name='verification_user_warn' OR event_name='verification_user_kick')"),
		qm.Where("guild_id = ?", guildID),
		qm.Where("(data->>'user_id')::bigint = ?", userID),
		qm.Where("processed = false")).DeleteAll(ctx, common.PQ)
	if err != nil {
		return errors.WithStackIf(err)
	}

	return nil
}

func (p *Plugin) findIPConflicts(guildID int64, userID int64, ip string) ([]*discordgo.User, error) {

	conflicts, err := models.VerifiedUsers(models.VerifiedUserWhere.GuildID.EQ(guildID), models.VerifiedUserWhere.IP.EQ(ip)).AllG(context.Background())
	if err != nil {
		return nil, err
	}

	if len(conflicts) < 2 {
		// this will include ourselves so ignore that
		return nil, nil
	}

	userIDs := make([]int64, 0, len(conflicts))
	for _, v := range conflicts {
		if v.UserID == userID {
			continue
		}

		userIDs = append(userIDs, v.UserID)
	}

	users := bot.GetUsers(guildID, userIDs...)
	return users, nil
}

func (p *Plugin) CheckBanned(guildID int64, users []*discordgo.User) (*discordgo.GuildBan, error) {
	for _, v := range users {
		ban, err := common.BotSession.GuildBan(guildID, v.ID)
		if err != nil {
			if cast, ok := err.(*discordgo.RESTError); ok && cast.Response != nil {
				if cast.Response.StatusCode == 404 {
					continue // Not banned, ban not found
				}
			}
			return nil, err
		}

		if ban != nil {
			return ban, nil
		}
	}

	return nil, nil
}

func (p *Plugin) handleWarnUserVerification(ms *dstate.MemberState, guildID int64, conf *models.VerificationConfig, rawData interface{}) (retry bool, err error) {
	if p.checkMemberAlreadyVerified(ms, conf) {
		return false, nil
	}

	gs := bot.State.GetGuild(guildID)
	if gs == nil {
		return false, nil
	}

	d := rawData.(*VerificationEventData)

	exists, err := models.VerificationSessions(
		models.VerificationSessionWhere.Token.EQ(d.Token),
		models.VerificationSessionWhere.SolvedAt.IsNotNull(),
	).ExistsG(context.Background())
	if err != nil {
		return scheduledevents2.CheckDiscordErrRetry(err), err
	}

	if exists {
		// User was verified
		return false, nil
	}

	err = p.sendWarning(ms, gs, d.Token, conf)
	return scheduledevents2.CheckDiscordErrRetry(err), err
}

func (p *Plugin) sendWarning(ms *dstate.MemberState, gs *dstate.GuildSet, token string, conf *models.VerificationConfig) error {

	msg := conf.WarnMessage
	if strings.TrimSpace(msg) == "" {
		return nil // no message to send
	}

	channel, err := common.BotSession.UserChannelCreate(ms.User.ID)
	if err != nil {
		return err
	}
	cs := dstate.ChannelStateFromDgo(channel)

	tmplCTX := templates.NewContext(gs, &cs, ms)
	tmplCTX.Name = "warn message"
	tmplCTX.Data["Link"] = fmt.Sprintf("%s/public/%d/verify/%d/%s", web.BaseURL(), gs.ID, ms.User.ID, token)

	err = tmplCTX.ExecuteAndSendWithErrors(msg, channel.ID)
	if err != nil {
		logger.WithError(err).WithField("guild", gs.ID).WithField("user", ms.User.ID).Error("failed sending warning message")
	}

	return nil
}

func (p *Plugin) handleKickUser(ms *dstate.MemberState, guildID int64, conf *models.VerificationConfig, rawData interface{}) (retry bool, err error) {
	if p.checkMemberAlreadyVerified(ms, conf) {
		return false, nil
	}

	dataCast := rawData.(*VerificationEventData)

	exists, err := models.VerificationSessions(
		models.VerificationSessionWhere.Token.EQ(dataCast.Token),
		models.VerificationSessionWhere.SolvedAt.IsNotNull(),
	).ExistsG(context.Background())
	if err != nil {
		return scheduledevents2.CheckDiscordErrRetry(err), err
	}

	if exists {
		// User was verified
		return false, nil
	}

	err = common.BotSession.GuildMemberDelete(guildID, ms.User.ID)
	if err == nil {
		p.logAction(guildID, conf.LogChannel, &ms.User, "Kicked for not verifying within deadline", 0xef4640)
	}

	return scheduledevents2.CheckDiscordErrRetry(err), err
}

func (p *Plugin) logAction(guildID int64, channelID int64, author *discordgo.User, action string, color int) {
	if channelID == 0 {
		return
	}

	_, err := common.BotSession.ChannelMessageSendEmbed(channelID, &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			IconURL: author.AvatarURL("128"),
			Name:    fmt.Sprintf("%s (%d)", author.String(), author.ID),
		},
		Description: action,
		Color:       color,
	})

	if err != nil {
		if common.IsDiscordErr(err, discordgo.ErrCodeMissingPermissions, discordgo.ErrCodeUnknownChannel) {
			go p.disableLogChannel(guildID)
		} else {
			logger.WithError(err).WithField("channel", channelID).Error("failed sending log message")
		}
	}
}

func (p *Plugin) disableLogChannel(guildID int64) {
	logger.WithField("guild", guildID).Warnf("disabling log channel due to it being unavailable or missing perms")

	const q = `UPDATE verification_configs SET log_channel=0 WHERE guild_id=$1`
	_, err := common.PQ.Exec(q, guildID)
	if err != nil {
		logger.WithField("guild", guildID).WithError(err).Error("failed disabling log channel")
	}
}

type RecentGuildBan struct {
	GuildID int64
	UserID  int64
	T       time.Time
}

// to avoid getting in a ban loop, we keep a cache of recently banned users by the bot
var (
	recentGuildBans   []*RecentGuildBan
	recentGuildBansmu sync.Mutex
)

func gcRecentGuildBansLoop() {
	tc := time.NewTicker(time.Minute)
	for {
		<-tc.C
		gcRecentGuildBans()
	}
}

func gcRecentGuildBans() {
	recentGuildBansmu.Lock()
	defer recentGuildBansmu.Unlock()

	if len(recentGuildBans) < 1 {
		return
	}

	newGuildBans := make([]*RecentGuildBan, 0, len(recentGuildBans))
	for _, v := range recentGuildBans {
		if time.Since(v.T) < time.Second*10 {
			newGuildBans = append(newGuildBans, v)
		}
	}

	recentGuildBans = newGuildBans
}

func wasRecentlyBannedByVerification(guildID int64, userID int64) bool {
	recentGuildBansmu.Lock()
	defer recentGuildBansmu.Unlock()

	for _, v := range recentGuildBans {
		if v.GuildID != guildID || v.UserID != userID {
			continue
		}
		if time.Since(v.T) > time.Second*10 {
			continue
		}

		return true
	}

	return false
}

func markRecentlyBannedByVerification(guildID int64, userID int64) {
	recentGuildBansmu.Lock()
	defer recentGuildBansmu.Unlock()

	for _, v := range recentGuildBans {
		if v.GuildID == guildID && v.UserID == userID {
			v.T = time.Now()
			return
		}
	}

	recentGuildBans = append(recentGuildBans, &RecentGuildBan{
		UserID:  userID,
		GuildID: guildID,
		T:       time.Now(),
	})
}

func (p *Plugin) handleBanAdd(evt *eventsystem.EventData) {
	ban := evt.GuildBanAdd()

	if !confVerificationTrackIPs.GetBool() {
		return
	}

	if wasRecentlyBannedByVerification(ban.GuildID, ban.User.ID) {
		return
	}

	model, err := models.FindVerifiedUserG(context.Background(), ban.GuildID, ban.User.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return
		}
		logger.WithError(err).Error("error finding verified user in banadd")
		return
	}

	if model.IP == "" {
		return
	}

	alts, err := p.findIPConflicts(ban.GuildID, ban.User.ID, model.IP)
	if err != nil {
		logger.WithError(err).Error("error finding ip conflicts in banadd")
		return
	}

	if len(alts) < 1 {
		return
	}

	go p.banAlts(ban, alts)
}

func (p *Plugin) banAlts(ban *discordgo.GuildBanAdd, alts []*discordgo.User) {
	for i, v := range alts {
		if i != 0 {
			time.Sleep(time.Second)
		}
		// check if they're already banned
		_, err := common.BotSession.GuildBan(ban.GuildID, v.ID)
		if err == nil {
			continue
		}

		if cast, ok := err.(*discordgo.RESTError); ok && cast.Response != nil {
			if cast.Response.StatusCode == 404 {
				// not banned
				logger.WithField("guild", ban.GuildID).WithField("user", v.ID).WithField("dupe-of", ban.User.ID).Info("banning alt account")
				reason := fmt.Sprintf("Alt of banned user (%s (%d))", ban.User.String(), ban.User.ID)
				markRecentlyBannedByVerification(ban.GuildID, v.ID)
				moderation.BanUser(nil, ban.GuildID, nil, nil, common.BotUser, reason, v, false)
				continue
			}
		}

		logger.WithError(err).Error("failed retrieving guild ban")
	}
}

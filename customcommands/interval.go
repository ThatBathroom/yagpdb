package customcommands

import (
	"context"
	"time"

	"emperror.dev/errors"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/common/scheduledevents2"
	schEventsModels "github.com/ThatBathroom/yagpdb/common/scheduledevents2/models"
	"github.com/ThatBathroom/yagpdb/customcommands/models"
	"github.com/robfig/cron/v3"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// CalcNextRunTime calculates the next run time for a custom command using the last ran time
func CalcNextRunTime(cc *models.CustomCommand, now time.Time) time.Time {
	if len(cc.TimeTriggerExcludingDays) >= 7 || len(cc.TimeTriggerExcludingHours) >= 24 {
		// this can never be ran...
		return time.Time{}
	}

	var tNext time.Time

	switch CommandTriggerType(cc.TriggerType) {
	case CommandTriggerInterval:
		if cc.TimeTriggerInterval < MinIntervalTriggerDurationMinutes || cc.TimeTriggerInterval > MaxIntervalTriggerDurationMinutes {
			// the interval is out of bounds and should never run
			return time.Time{}
		}

		tNext = cc.LastRun.Time.Add(time.Minute * time.Duration(cc.TimeTriggerInterval))
		// run it immedietely if this is the case
		if tNext.Before(now) {
			tNext = now
		}

		// ensure were dealing with utc
		tNext = tNext.UTC()

		// Check for blaclisted days and if we encountered a blacklisted day we reset the clock
		tNext = intervalCheckDays(cc, tNext, true)

		// check for blacklisted hours
		tNext = intervalCheckHours(cc, tNext)

		// its possible we went forward a day while checking for blacklisted hours, in that case check for blacklisted days again
		tNext = intervalCheckDays(cc, tNext, true)

		// AND finally if we
		// 1. blacklisted hour led to nextTime going to another day
		// 2. hour is now reset
		// 3. hour is now blacklisted
		// do a final check for #3
		//
		// should not be possible to land on a new day after this, so further checks are not needed
		tNext = intervalCheckHours(cc, tNext)
	case CommandTriggerCron:
		cronSchedule, err := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow).Parse(cc.TextTrigger)
		if err != nil {
			// somehow this got past validation, can't run
			return time.Time{}
		}
		specSchedule := cronSchedule.(*cron.SpecSchedule)
		const hoursInADay = 24
		const daysInAWeek = 7
		const starIsConfiguredBitset = 1<<63 // top bit set == "*"
		var newHoursScheduledBitset uint64
		var newDaysScheduledBitset uint64
		for hourOfDay := range hoursInADay {
			hourOfDayBitVal := uint64(1) << hourOfDay
			hourPresentInSchedule := specSchedule.Hour&hourOfDayBitVal == hourOfDayBitVal
			blacklistedHour := common.ContainsInt64Slice(cc.TimeTriggerExcludingHours, int64(hourOfDay))
			if hourPresentInSchedule && !blacklistedHour {
				newHoursScheduledBitset = newHoursScheduledBitset | hourOfDayBitVal
			}
		}
		monthOrDomConfigured := specSchedule.Month&starIsConfiguredBitset == 0 || specSchedule.Dom&starIsConfiguredBitset == 0
		dowConfigured := specSchedule.Dow&starIsConfiguredBitset == 0

		// if month/dom and dow are both configured, cron will process them
		// using OR. If user didn't want that, they wouldn't set both in
		// the exppression. but if using day of week blacklist, in the way
		// we do it here that would configure it, and then cron would use
		// OR and mess up the user's schedule if they explicitly avoided
		// this behaviour in their expression. For this reason, we only use
		// the below method if month/dom aren't configured, or if week is.
		skipBlacklistCheck := monthOrDomConfigured && !dowConfigured

		for dayOfWeek := range daysInAWeek {
			dayOfWeekBitVal := uint64(1) << dayOfWeek
			dayPresentInSchedule := specSchedule.Dow&dayOfWeekBitVal != 0
			blacklistedDay := common.ContainsInt64Slice(cc.TimeTriggerExcludingDays, int64(dayOfWeek))
			if dayPresentInSchedule && (skipBlacklistCheck || !blacklistedDay) {
				newDaysScheduledBitset = newDaysScheduledBitset | dayOfWeekBitVal
			}
		}

		const allHoursBitset = (1<<hoursInADay)-1
		const allDaysBitset = (1<<daysInAWeek)-1
		if newHoursScheduledBitset != allHoursBitset { // if all hours set, leave as is to distinguish between "0,1,2...,23" and "*"
			specSchedule.Hour = newHoursScheduledBitset
		}

		if newDaysScheduledBitset != allDaysBitset { // if all days set, leave as is to distinguish between "0,1,2...,6" and "*"
			specSchedule.Dow = newDaysScheduledBitset
		}

		if specSchedule.Hour == 0 || specSchedule.Dow == 0 {
			// this can never run
			return time.Time{}
		}

		timeNext := now.UTC()
		for i := 0; i < 200; i++ {
			// alternative method if month/dom are configured but dow isn't. in
			// the lucky scenarios where it's calculated right on the first
			// try, this won't need to loop.
			timeNext = specSchedule.Next(timeNext)
			timeNextWeekdayIsBlacklisted := common.ContainsInt64Slice(cc.TimeTriggerExcludingDays, int64(timeNext.Weekday()))
			if !timeNextWeekdayIsBlacklisted {
				break
			}
		}
		tNext = timeNext
	}

	return tNext
}

func intervalCheckDays(cc *models.CustomCommand, tNext time.Time, resetClock bool) time.Time {
	// check for blacklisted days
	if !common.ContainsInt64Slice(cc.TimeTriggerExcludingDays, int64(tNext.Weekday())) {
		return tNext
	}

	// find the next available day
	for {
		tNext = tNext.Add(time.Hour * 24)

		if !common.ContainsInt64Slice(cc.TimeTriggerExcludingDays, int64(tNext.Weekday())) {
			break
		}
	}

	if resetClock {
		// if we went forward a day, force the clock to 0 to run it as soon as possible
		h, m, s := tNext.Clock()
		tNext = tNext.Add((-time.Hour * time.Duration(h)))
		tNext = tNext.Add((-time.Minute * time.Duration(m)))
		tNext = tNext.Add((-time.Second * time.Duration(s)))
	}
	return tNext
}

func intervalCheckHours(cc *models.CustomCommand, tNext time.Time) time.Time {
	// check for blacklisted hours
	if !common.ContainsInt64Slice(cc.TimeTriggerExcludingHours, int64(tNext.Hour())) {
		return tNext
	}

	// find the next available hour
	for {
		tNext = tNext.Add(time.Hour)

		if !common.ContainsInt64Slice(cc.TimeTriggerExcludingHours, int64(tNext.Hour())) {
			break
		}
	}

	return tNext
}

type NextRunScheduledEvent struct {
	CmdID int64 `json:"cmd_id"`
}

func DelNextRunEvent(guildID int64, cmdID int64) error {
	_, err := schEventsModels.ScheduledEvents(qm.Where("event_name='cc_next_run' AND guild_id = ? AND (data->>'cmd_id')::bigint = ?", guildID, cmdID)).DeleteAll(context.Background(), common.PQ)
	return err
}

// TODO: Run this all in a transaction?
func UpdateCommandNextRunTime(cc *models.CustomCommand, updateLastRun bool, clearOld bool) error {
	if clearOld {
		// remove the old events
		err := DelNextRunEvent(cc.GuildID, cc.LocalID)
		if err != nil {
			return errors.WrapIf(err, "del_old_events")
		}
	}

	isIntervalOrCron := cc.TriggerType == int(CommandTriggerInterval) || cc.TriggerType == int(CommandTriggerCron)
	invalidInterval := cc.TriggerType == int(CommandTriggerInterval) && cc.TimeTriggerInterval < 1
	if !isIntervalOrCron || invalidInterval {
		return nil
	}

	// calculate the next run time
	nextRun := CalcNextRunTime(cc, time.Now())
	if nextRun.IsZero() {
		return nil
	}

	// update the command
	cc.NextRun = null.TimeFrom(nextRun)
	toUpdate := []string{"next_run"}
	if updateLastRun {
		toUpdate = append(toUpdate, "last_run")
	}
	_, err := cc.UpdateG(context.Background(), boil.Whitelist(toUpdate...))
	if err != nil {
		return errors.WrapIf(err, "update_cc")
	}

	evt := &NextRunScheduledEvent{
		CmdID: cc.LocalID,
	}

	// create a scheduled event to run the command again
	err = scheduledevents2.ScheduleEvent("cc_next_run", cc.GuildID, nextRun, evt)
	if err != nil {
		return errors.WrapIf(err, "schedule_event")
	}

	return nil
}

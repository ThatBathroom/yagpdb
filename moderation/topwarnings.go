package moderation

import (
	"database/sql"

	// "github.com/ThatBathroom/yagpdb/lib/discordgo"
	// "github.com/ThatBathroom/yagpdb/lib/dstate"
	"github.com/ThatBathroom/yagpdb/bot"
	"github.com/ThatBathroom/yagpdb/common"
)

type WarnRankEntry struct {
	Rank      int    `json:"rank"`
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	WarnCount int64  `json:"warn_count"`
}

func TopWarns(guildID int64, offset, limit int) ([]*WarnRankEntry, error) {
	const query = `SELECT rank, warn_count, user_id FROM
	(
		SELECT RANK() OVER (ORDER BY count(message) DESC) AS rank, count(*) as warn_count, user_id
		FROM moderation_warnings WHERE guild_id = $1 group by user_id
	) AS warns
	ORDER BY warn_count desc
	LIMIT $2 OFFSET $3`

	rows, err := common.PQ.Query(query, guildID, limit, offset)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*WarnRankEntry{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	result := make([]*WarnRankEntry, 0, limit)
	for rows.Next() {
		//var member []*discordgo.Member
		var rank int
		//var tmp []*dstate.MemberState
		var userID int64
		var warncount int64
		var err = rows.Scan(&rank, &warncount, &userID)
		if err != nil {
			return nil, err
		}

		userSlice := bot.GetUsers(guildID, userID)
		var username string
		for _, u := range userSlice {
			username = u.String()
			break
		}

		result = append(result, &WarnRankEntry{
			Rank:      rank,
			UserID:    userID,
			WarnCount: warncount,
			Username:  username,
		})
	}

	return result, nil
}

package inspire

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ThatBathroom/yagpdb/bot/paginatedmessages"
	"github.com/ThatBathroom/yagpdb/commands"
	"github.com/ThatBathroom/yagpdb/common"
	"github.com/ThatBathroom/yagpdb/lib/dcmd"
	"github.com/ThatBathroom/yagpdb/lib/discordgo"
)

var Command = &commands.YAGCommand{
	CmdCategory:         commands.CategoryFun,
	Name:                "Inspire",
	Aliases:             []string{"insp"},
	Description:         "Shows 'inspirational' quotes from inspirobot.me",
	RunInDM:             false,
	DefaultEnabled:      true,
	SlashCommandEnabled: true,
	Cooldown:            3,
	Arguments: []*dcmd.ArgDef{
		{Name: "Season", Type: dcmd.String, Default: ""},
	},
	ArgSwitches: []*dcmd.ArgDef{
		{Name: "mindfulness", Help: "Generates Mindful Quotes!"},
	},
	RunFunc: func(data *dcmd.Data) (interface{}, error) {
		available := map[string]bool{"xmas": true}
		inspireArray := []string{}
		var paginatedView bool
		season := strings.ToLower(data.Args[0].Str())
		if len(season) > 0 && !available[season] {
			return "Available seasons: `xmas`", nil
		}
		paginatedView = false
		if data.Switches["mindfulness"].Value != nil && data.Switches["mindfulness"].Value.(bool) {
			paginatedView = true
		}
		if paginatedView {
			result, err := inspireFromAPI(true, season)
			if err != nil {
				return nil, err
			}
			inspireArray = arrayMaker(inspireArray, result)
			return paginatedmessages.NewPaginatedResponse(
				data.GuildData.GS.ID, data.ChannelID, 1, 15, func(p *paginatedmessages.PaginatedMessage, page int) (*discordgo.MessageEmbed, error) {
					if page-1 == len(inspireArray) {
						result, err := inspireFromAPI(true, season)
						if err != nil {
							return nil, err
						}
						inspireArray = arrayMaker(inspireArray, result)
					}
					return createInspireEmbed(inspireArray[page-1], true), nil
				}), nil
		}
		inspData, err := inspireFromAPI(false, season)
		if err != nil {
			return fmt.Sprintf("%s\nInspiroBot wonky... sad times :/", err), err
		}
		embed := createInspireEmbed(inspData, false)
		return embed, nil
	},
}

func inspireFromAPI(mindfulnessMode bool, season string) (string, error) {
	query := "https://inspirobot.me/api?generate=true"
	if mindfulnessMode {
		query = fmt.Sprintf("https://inspirobot.me/api?generateFlow=1&sessionID=%d", time.Now().UTC().Unix())
	} else if len(season) > 0 {
		query = "https://inspirobot.me/api?generate=true&season=" + season
	}

	req, err := http.NewRequest("GET", query, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "YAGPDB.xyz (https://github.com/botlabs-gg/yagpdb)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", commands.NewPublicError("HTTP err: ", resp.StatusCode)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if mindfulnessMode {
		var mindful MindfulnessMode

		err := json.Unmarshal([]byte(body), &mindful)
		if err != nil {
			return "", err
		}

		mindfulness := mindful.Data[1].Text
		if len(mindfulness) > 4000 {
			mindfulness = common.CutStringShort(mindfulness, 4000)
		}

		return mindfulness, nil
	}

	return string(body), nil
}

func createInspireEmbed(data string, mindfulness bool) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{}

	if mindfulness {
		embed.Title = "Here's an inspirational quote (Mindfulness Mode):"
		embed.Description = "```\n" + data + "```"
		embed.Color = int(11413503)
	} else {
		embed.Color = int(rand.Int63n(0xffffff))
		embed.Description = "Here's an inspirational quote:"
		embed.Image = &discordgo.MessageEmbedImage{
			URL: data,
		}
	}
	return embed
}
func arrayMaker(list []string, data string) []string {
	if data != "" {
		re := regexp.MustCompile(`\[pause \d+\]`)
		list = append(list, re.ReplaceAllString(data, ""))
	}
	return list
}

type MindfulnessMode struct {
	Data []struct {
		Duration *float64 `json:"duration,omitempty"`
		Image    *string  `json:"image,omitempty"`
		Type     *string  `json:"type"`
		Time     *float64 `json:"time"`
		Text     string   `json:"text,omitempty"`
	} `json:"data"`
}

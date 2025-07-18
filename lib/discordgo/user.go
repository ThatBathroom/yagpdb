package discordgo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ThatBathroom/yagpdb/lib/gojay"
)

// A User stores all data for an individual Discord user.
type User struct {
	// The ID of the user.
	ID int64 `json:"id,string"`

	// The user's username.
	Username string `json:"username"`

	// The user's display name on discord
	Globalname string `json:"global_name"`

	// The hash of the user's avatar. Use Session.UserAvatar
	// to retrieve the avatar itself.
	Avatar string `json:"avatar"`

	// The user's chosen language option.
	Locale string `json:"locale"`

	// The discriminator of the user (4 numbers after name).
	Discriminator string `json:"discriminator"`

	// Whether the user is a bot.
	Bot bool `json:"bot"`
}

// String returns a unique identifier of the form username#discriminator
func (u *User) String() string {
	if u.Discriminator == "0" {
		return u.Username
	}
	return fmt.Sprintf("%s#%s", u.Username, u.Discriminator)
}

// Mention return a string which mentions the user
func (u *User) Mention() string {
	return fmt.Sprintf("<@%d>", u.ID)
}

// implement gojay.UnmarshalerJSONObject
func (u *User) UnmarshalJSONObject(dec *gojay.Decoder, key string) error {
	switch key {
	case "id":
		return DecodeSnowflake(&u.ID, dec)
	case "username":
		return dec.String(&u.Username)
	case "global_name":
		return dec.String(&u.Globalname)
	case "avatar":
		return dec.String(&u.Avatar)
	case "locale":
		return dec.String(&u.Locale)
	case "discriminator":
		return dec.String(&u.Discriminator)
	case "bot":
		return dec.Bool(&u.Bot)
	}

	return nil
}

func (u *User) NKeys() int {
	return 0
}

// AvatarURL returns a URL to the user's avatar.
//
//	size:    The size of the user's avatar as a power of two
//	         if size is an empty string, no size parameter will
//	         be added to the URL.
func (u *User) AvatarURL(size string) string {
	var URL string
	if u.Avatar == "" {
		// See https://discord.com/developers/docs/reference#image-formatting-cdn-endpoints:
		//
		// "For users on the new username system, `index` will be `(user_id >> 22) % 6`.
		// For users on the legacy username system, `index` will be `discriminator % 5`."
		var index int
		if u.Discriminator == "0" {
			index = int((u.ID >> 22) % 6)
		} else {
			discrim, _ := strconv.Atoi(u.Discriminator)
			index = discrim % 5
		}
		URL = EndpointDefaultUserAvatar(index)
	} else if strings.HasPrefix(u.Avatar, "a_") {
		URL = EndpointUserAvatarAnimated(u.ID, u.Avatar)
	} else {
		URL = EndpointUserAvatar(u.ID, u.Avatar)
	}

	if size != "" {
		return URL + "?size=" + size
	}
	return URL
}

// A SelfUser stores user data about the token owner.
// Includes a few extra fields than a normal user struct.
type SelfUser struct {
	*User
	Token string `json:"token"`
}

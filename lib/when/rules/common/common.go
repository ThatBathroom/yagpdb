package common

import "github.com/ThatBathroom/yagpdb/v2/lib/when/rules"

var All = []rules.Rule{
	SlashDMY(rules.Override),
}

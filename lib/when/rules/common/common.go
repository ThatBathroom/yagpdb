package common

import "github.com/ThatBathroom/yagpdb/lib/when/rules"

var All = []rules.Rule{
	SlashDMY(rules.Override),
}

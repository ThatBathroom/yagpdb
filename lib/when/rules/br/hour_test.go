package br_test

import (
	"testing"
	"time"

	"github.com/ThatBathroom/yagpdb/lib/when"
	"github.com/ThatBathroom/yagpdb/lib/when/rules"
	"github.com/ThatBathroom/yagpdb/lib/when/rules/br"
)

func TestHour(t *testing.T) {
	fixt := []Fixture{
		{"5pm", 0, "5pm", 17 * time.Hour},
		{"at 5 pm", 3, "5 pm", 17 * time.Hour},
		{"at 5 P.", 3, "5 P.", 17 * time.Hour},
		{"at 12 P.", 3, "12 P.", 12 * time.Hour},
		{"at 1 P.", 3, "1 P.", 13 * time.Hour},
		{"at 5 am", 3, "5 am", 5 * time.Hour},
		{"at 5A", 3, "5A", 5 * time.Hour},
		{"at 5A.", 3, "5A.", 5 * time.Hour},
		{"5A.", 0, "5A.", 5 * time.Hour},
		{"11 P.M.", 0, "11 P.M.", 23 * time.Hour},
	}

	w := when.New(nil)
	w.Add(br.Hour(rules.Override))

	ApplyFixtures(t, "br.Hour", w, fixt)
}

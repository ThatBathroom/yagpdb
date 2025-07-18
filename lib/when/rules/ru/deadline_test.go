package ru_test

import (
	"testing"
	"time"

	"github.com/ThatBathroom/yagpdb/lib/when"
	"github.com/ThatBathroom/yagpdb/lib/when/rules"
	"github.com/ThatBathroom/yagpdb/lib/when/rules/ru"
)

func TestDeadline(t *testing.T) {
	fixt := []Fixture{
		{"нужно сделать это в течении получаса", 33, "в течении получаса", time.Hour / 2},
		{"нужно сделать это в течении одного часа", 33, "в течении одного часа", time.Hour},
		{"нужно сделать это за один час", 33, "за один час", time.Hour},
		{"за 5 минут", 0, "за 5 минут", time.Minute * 5},
		{"Через 5 минут я пойду домой.", 0, "Через 5 минут", time.Minute * 5},
		{"Нам необходимо сделать это за 10 дней.", 50, "за 10 дней", 10 * 24 * time.Hour},
		{"Нам необходимо сделать это за пять дней.", 50, "за пять дней", 5 * 24 * time.Hour},
		{"Нам необходимо сделать это через 5 дней.", 50, "через 5 дней", 5 * 24 * time.Hour},
		{"Через 5 секунд нужно убрать машину", 0, "Через 5 секунд", 5 * time.Second},
		{"за две недели", 0, "за две недели", 14 * 24 * time.Hour},
		{"через месяц", 0, "через месяц", 31 * 24 * time.Hour},
		{"за месяц", 0, "за месяц", 31 * 24 * time.Hour},
		{"за несколько месяцев", 0, "за несколько месяцев", 91 * 24 * time.Hour},
		{"за один год", 0, "за один год", 366 * 24 * time.Hour},
		{"за неделю", 0, "за неделю", 7 * 24 * time.Hour},
	}

	w := when.New(nil)
	w.Add(ru.Deadline(rules.Skip))

	ApplyFixtures(t, "ru.Deadline", w, fixt)
}
